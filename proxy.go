package tcpee

import (
	"context"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"codeberg.org/gruf/go-errors"
)

// ErrProxyClosed will be returned upon proxy close
var ErrProxyClosed = errors.New("tcpee: proxy closed")

type TCPProxy struct {
	// Name is the name of this proxy server, used when
	// logging via the supplied logger
	Name string

	// Logger is the Logger implementation this instance
	// will use, or nil to disable
	Logger Logger

	// ProxyProto determines whether to write proxy protocol headers
	// for each proxied TCP connection. The header is proxy protocol
	// v1 compatible, and more information can be found here:
	// https://www.haproxy.org/download/2.6/doc/proxy-protocol.txt
	ProxyProto bool

	// DialTimeout is the maximum time a dial will wait for a
	// connection to complete
	DialTimeout time.Duration

	// ClientTimeout is the maximum time a client conn may idle before
	// being forcibly closed. Note that longer timeout periods
	// will be more efficient as they require less-frequent checks
	ClientTimeout time.Duration

	// ServerTimeout is the maximum time a server conn may idle before
	// being forcibly closed. Note that longer timeout periods
	// will be more efficient as they require less-frequent checks
	ServerTimeout time.Duration

	// ClientKeepAlive specifies the keep-alive period for conns from
	// the client. If zero, keep-alives are enabled with a default
	// value.If negative, keep-alives are disabled.
	ClientKeepAlive time.Duration

	// ServerKeepAlive specifies the keep-alive period for conns to
	// the server. If zero, keep-alives are enabled with a default
	// value.If negative, keep-alives are disabled.
	ServerKeepAlive time.Duration

	lnCfg   net.ListenConfig // lnCfg is the set listener config
	dialer  net.Dialer       // dialer is the set dialer we use
	cancel  func()           // cancel is the proxy context cancel
	baseCtx context.Context  // baseCtx is the proxy base context
	serveWg sync.WaitGroup   // serveWg tracks running serve routines
	doOnce  sync.Once        // doOnce is the proxy init routine protector
	open    int32            // open tracks the no. open proxy connections
}

func (proxy *TCPProxy) init() {
	proxy.doOnce.Do(func() {
		// If no name set, use default
		if len(proxy.Name) < 1 {
			proxy.Name = "proxy"
		}

		// If no logger provided, use nil
		if proxy.Logger == nil {
			proxy.Logger = &nopLogger{}
		}

		// Setup the listener cfg and dialer
		proxy.lnCfg = net.ListenConfig{
			KeepAlive: proxy.ClientKeepAlive,
		}
		proxy.dialer = net.Dialer{
			KeepAlive: proxy.ServerKeepAlive,
			Timeout:   proxy.DialTimeout,
		}

		// Setup proxy base context
		proxy.baseCtx, proxy.cancel = context.WithCancel(context.Background())
	})
}

// printf is just shorthand for proxy.Logger.Printf().
func (proxy *TCPProxy) printf(s string, a ...interface{}) {
	proxy.Logger.Printf(s, a...)
}

// dial dials a TCP connection to supplied address
func (proxy *TCPProxy) dial(dst string) (net.Conn, error) {
	return proxy.dialer.DialContext(proxy.baseCtx, "tcp", dst)
}

// listen starts a TCP listener on on supplied address
func (proxy *TCPProxy) listen(src string) (net.Listener, error) {
	return proxy.lnCfg.Listen(proxy.baseCtx, "tcp", src)
}

// Close closes the TCPProxy, waiting for all serve routines to finish
func (proxy *TCPProxy) Close() {
	proxy.cancel()
	proxy.serveWg.Wait()
}

// Proxy starts a proxy handler listening on the supplied src address, and
// proxying it to the supplied dst address
func (proxy *TCPProxy) Proxy(src string, dst string) error {
	// Ensure initialized
	proxy.init()

	// Ensure we can dial-out
	conn, err := proxy.dial(dst)
	if err != nil {
		return err
	}
	err = conn.Close()
	if err != nil {
		return err
	}

	// Start TCP listener
	ln, err := proxy.listen(src)
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		// Break-out on close
		select {
		case <-proxy.baseCtx.Done():
			return ErrProxyClosed
		default:
		}

		// Define pre-loop
		var conn net.Conn
		var err error

	inner:
		for {
			// Accept next connection
			conn, err = ln.Accept()
			if err != nil {
				// Check for temporary errors
				if nErr, ok := err.(net.Error); ok && nErr.Temporary() {
					proxy.printf("%s: temp. accept error: %v", proxy.Name, err)
					time.Sleep(time.Second)
					continue inner
				}

				if errors.Is(err, io.EOF) {
					// EOF is NOT an error
					err = nil
				} else {
					// Log all other errors
					proxy.printf("%s: accept error: %v", proxy.Name, err)
				}

				return err
			}

			// Successful connection
			break inner
		}

		// Start tracking serve routine
		proxy.serveWg.Add(1)
		atomic.AddInt32(&proxy.open, 1)

		// Serve this connection
		go proxy.serve(conn, dst)
	}
}

// serve is the main proxy routine that manages serving data between conns
func (proxy *TCPProxy) serve(sConn net.Conn, dst string) {
	defer func() {
		// Untrack serve routine
		atomic.AddInt32(&proxy.open, -1)
		proxy.serveWg.Done()
	}()

	// Note: closing is handled by each of the
	//       copyConn goroutines below. this allows
	//       writes to finish if we have buffered
	//       but read has closed

	// Dial-out to destination address
	dConn, err := proxy.dial(dst)
	if err != nil {
		proxy.printf("%s: dial error: %v", proxy.Name, err)
		return
	}

	// Cast our connections and addrs
	dTCPConn := dConn.(*net.TCPConn)
	sTCPConn := sConn.(*net.TCPConn)
	dTCPAddr := dTCPConn.RemoteAddr().(*net.TCPAddr)
	sTCPAddr := sTCPConn.RemoteAddr().(*net.TCPAddr)
	dstIP := dTCPAddr.IP.String()
	srcIP := sTCPAddr.IP.String()

	// Log proxying
	proxy.Logger.Printf(
		"%s:%d [%d] %s -> %s",
		proxy.Name,
		dTCPAddr.Port,
		atomic.LoadInt32(&proxy.open),
		dstIP,
		srcIP,
	)

	// Set proxy header if required
	if proxy.ProxyProto {
		// 107=worstcase IPv6 scenario
		hdr := make([]byte, 0, 107)
		hdr = append(hdr, `PROXY `...)

		sTCPAddr.IP.To4()

		// Append protocol version
		if isIPv4(sTCPAddr.IP) {
			hdr = append(hdr, `TCP `...)
		} else {
			hdr = append(hdr, `TCP6 `...)
		}

		// Append src + dst addresses
		hdr = append(hdr, srcIP...)
		hdr = append(hdr, ' ')
		hdr = append(hdr, dstIP...)

		// Append src + dst ports, then final CRLF
		hdr = strconv.AppendInt(hdr, int64(sTCPAddr.Port), 10)
		hdr = append(hdr, ' ')
		hdr = strconv.AppendInt(hdr, int64(dTCPAddr.Port), 10)
		hdr = append(hdr, '\r', '\n')

		// Finally write proxy header
		_, err := dTCPConn.Write(hdr)
		if err != nil {
			proxy.printf("%s: output error: %v", proxy.Name, err)
			return
		}
	}

	// Setup error channels
	errIn := make(chan error, 1)
	errOut := make(chan error, 1)

	// Prepare the timeout-setting functions
	clientTimeout := timeoutFunc(proxy.ClientTimeout, sTCPConn.SetReadDeadline)
	serverTimeout := timeoutFunc(proxy.ServerTimeout, sTCPConn.SetWriteDeadline)

	// Start handling proxying
	go copyConn(dTCPConn, sTCPConn, errIn, clientTimeout)
	go copyConn(sTCPConn, dTCPConn, errOut, serverTimeout)

	select {
	// Wait on input error
	case err := <-errIn:
		if err != nil {
			proxy.printf("%s: input error: %v", proxy.Name, err)
		}

	// Wait on output* error
	case err := <-errOut:
		if err != nil {
			proxy.printf("%s: output error: %v", proxy.Name, err)
		}

	// Server ctx cancelled
	case <-proxy.baseCtx.Done():
		dTCPConn.Close()
		sTCPConn.Close()
	}
}

// copyConn copies from once TCPConn to another, using TCPConn's ReadFrom implementation
// to take advantage of the splice optimization. this also handles connection timeouts
func copyConn(dst *net.TCPConn, src *net.TCPConn, errChan chan error, setTimeout func()) {
	defer func() {
		// Ensure dst conn and error chan
		// closed on function close (even panic)
		close(errChan)
		dst.Close()
	}()

	for {
		// Set timeout
		setTimeout()

		// Copy from source to destination
		n, err := dst.ReadFrom(src)
		if err == nil || err == io.EOF || err == net.ErrClosed {
			// EOF / conn close -- no error
			break
		}

		// Timeouts are expected. This indicates we
		// should check the data transfer-rate
		if err, ok := err.(net.Error); ok && err.Timeout() {
			if n < 1 {
				break
			}

			// Rate is acceptable, keep-going
			continue
		}

		errChan <- err
		break
	}
}

// timeoutFunc returns a valid timeout function for copyConn() only if d > 0.
func timeoutFunc(d time.Duration, fn func(time.Time) error) func() {
	if d < 1 {
		return func() {}
	}
	return func() {
		fn(time.Now().Add(d))
	}
}

// isIPv4 returns whether IP is IPv4, logic from ip.ToV4()
func isIPv4(ip net.IP) bool {
	return (len(ip) == net.IPv4len) ||
		(len(ip) == net.IPv6len && isZeros(ip[0:10]) && ip[10] == 0xff && ip[11] == 0xff)
}

// Is p all zeros?
func isZeros(ip net.IP) bool {
	for i := 0; i < len(ip); i++ {
		if ip[i] != 0 {
			return false
		}
	}
	return true
}
