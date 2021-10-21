package tcpee

import (
	"context"
	"io"
	"net"
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

	// Transparent determines whether to enable Linux transparent
	// proxying on connections. Note that this must be done in
	// conjuction with iptables / nf_tables
	Transparent bool

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
	open    int32            // open tracks the no. open proxy connections
	cancel  func()           // cancel is the proxy context cancel
	baseCtx context.Context  // baseCtx is the proxy base context
	serveWg sync.WaitGroup   // serveWg tracks running serve routines
	doOnce  sync.Once        // doOnce is the proxy init routine protector
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

		// If transparent proxy requested, set
		if proxy.Transparent {
			proxy.lnCfg.Control = SetSocketIPTransparent
			proxy.dialer.Control = SetSocketIPTransparent
		}

		// Setup proxy base context
		proxy.baseCtx, proxy.cancel = context.WithCancel(context.Background())
	})
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
					proxy.Logger.Printf("%s: temp. accept error: %v", proxy.Name, err)
					time.Sleep(time.Second)
					continue inner
				}

				// EOF is not an error
				if errors.Is(err, io.EOF) {
					err = nil
				}

				// Return now
				proxy.Logger.Printf("%s: accept error: %v", proxy.Name, err)
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
		proxy.Logger.Printf("%s: dial error: %v", proxy.Name, err)
		return
	}

	// Log proxying
	proxy.Logger.Printf(
		"%s: [%d] %s -> %s",
		proxy.Name,
		atomic.LoadInt32(&proxy.open),
		sConn.RemoteAddr().String(),
		dConn.RemoteAddr().String(),
	)

	// Cast our connections
	dTCPConn := dConn.(*net.TCPConn)
	sTCPConn := sConn.(*net.TCPConn)

	// Setup error channels
	errIn := make(chan error, 1)
	errOut := make(chan error, 1)

	// Setup timeout-setting functions
	clientTimeout, serverTimeout := noTimeout, noTimeout
	if proxy.ClientTimeout > 0 {
		clientTimeout = func() {
			sTCPConn.SetReadDeadline(time.Now().Add(proxy.ClientTimeout))
		}
	}
	if proxy.ServerTimeout > 0 {
		serverTimeout = func() {
			sTCPConn.SetWriteDeadline(time.Now().Add(proxy.ServerTimeout))
		}
	}

	// Start handling proxying
	go copyConn(dTCPConn, sTCPConn, errIn, clientTimeout)
	go copyConn(sTCPConn, dTCPConn, errOut, serverTimeout)

	select {
	// Wait on input error
	case err := <-errIn:
		if err != nil {
			proxy.Logger.Printf("%s: input error: %v", proxy.Name, err)
		}

	// Wait on output error
	case err := <-errOut:
		if err != nil {
			proxy.Logger.Printf("%s: output error: %v", proxy.Name, err)
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

// noTimeout is an empty timeout-setting function
func noTimeout() {}
