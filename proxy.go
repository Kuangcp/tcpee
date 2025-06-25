package tcpee

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"codeberg.org/gruf/go-kv"
	"codeberg.org/gruf/go-logger/v2/log"
)

// ErrProxyClosed will be returned upon proxy close.
var ErrProxyClosed = errors.New("tcpee: proxy closed")

type TCPProxy struct {
	// Name is the name of this proxy server, used when
	// logging via the supplied logger
	Name string

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
	ppool   sync.Pool        // ppool is the proxy proto buffer pool
	open    int64            // open tracks the no. open proxy connections

	// 流量统计字段
	bytesIn  uint64 // 入站流量统计(字节)
	bytesOut uint64 // 出站流量统计(字节)
	
	// 统计锁
	statsMutex sync.RWMutex
	
	// 统计定时器
	statsTimer *time.Timer
}

func (proxy *TCPProxy) init() {
	proxy.doOnce.Do(func() {
		// If no name set, use default
		if len(proxy.Name) < 1 {
			proxy.Name = "proxy"
		}

		// Setup the listener cfg and dialer
		proxy.lnCfg = net.ListenConfig{
			KeepAlive: proxy.ClientKeepAlive,
		}
		proxy.dialer = net.Dialer{
			KeepAlive: proxy.ServerKeepAlive,
			Timeout:   proxy.DialTimeout,
		}

		// Setup proxy proto buffer pool
		proxy.ppool.New = func() interface{} {
			// 107 = worstcase scenario buflen
			return make([]byte, 0, 107)
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

	// Start stats timer
	proxy.startStatsTimer()

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
					log.ErrorKVs(kv.Fields{
						{K: "proxy", V: proxy.Name},
						{K: "error", V: err},
						{K: "msg", V: "temp. accept error"},
					}...)
					time.Sleep(time.Second)
					continue inner
				}

				if errors.Is(err, io.EOF) {
					// EOF is NOT an error
					err = nil
				} else {
					// Log all other errors
					log.ErrorKVs(kv.Fields{
						{K: "proxy", V: proxy.Name},
						{K: "error", V: err},
						{K: "msg", V: "accept error"},
					}...)
				}

				return err
			}

			// Successful connection
			break inner
		}

		// Start tracking serve routine
		proxy.serveWg.Add(1)
		atomic.AddInt64(&proxy.open, 1)

		// Serve this connection
		go proxy.serve(conn, dst)
	}
}

// serve is the main proxy routine that manages serving data between conns
func (proxy *TCPProxy) serve(sConn net.Conn, dst string) {
	defer func() {
		// Untrack serve routine
		atomic.AddInt64(&proxy.open, -1)
		proxy.serveWg.Done()
	}()

	// Note: closing is handled by each of the
	//       copyConn goroutines below. this allows
	//       writes to finish if we have buffered
	//       but read has closed

	// Dial-out to destination address
	dConn, err := proxy.dial(dst)
	if err != nil {
		log.ErrorKVs(kv.Fields{
			{K: "proxy", V: proxy.Name},
			{K: "error", V: err},
			{K: "msg", V: "dial error"},
		}...)
		return
	}

	// Cast our connections and addrs
	dTCPConn := dConn.(*net.TCPConn)
	sTCPConn := sConn.(*net.TCPConn)
	dTCPAddr := dTCPConn.RemoteAddr().(*net.TCPAddr)
	sTCPAddr := sTCPConn.RemoteAddr().(*net.TCPAddr)
	srcIP := sTCPAddr.IP.String()
	dstIP := dTCPAddr.IP.String()
	dstPort := strconv.Itoa(dTCPAddr.Port)

	// Log proxying
	log.InfoKVs(kv.Fields{
		// {K: "proxy", V: proxy.Name},
		{K: "count", V: atomic.LoadInt64(&proxy.open)},
		{K: "src", V: srcIP},
		{K: "dst", V: dstIP + ":" + dstPort},
	}...)

	// Set proxy header if required
	if proxy.ProxyProto {
		// Acquire header buffer
		hdr := proxy.ppool.Get().([]byte)
		defer func() {
			hdr = hdr[:0]
			proxy.ppool.Put(hdr)
		}()

		// Append protocol version
		hdr = append(hdr, `PROXY `...)
		if isIPv4(sTCPAddr.IP) {
			hdr = append(hdr, `TCP4 `...)
		} else {
			hdr = append(hdr, `TCP6 `...)
		}

		// Append src + dst addresses
		hdr = append(hdr, srcIP...)
		hdr = append(hdr, ' ')
		hdr = append(hdr, dstIP...)
		hdr = append(hdr, ' ')

		// Append src + dst ports, then final CRLF
		hdr = strconv.AppendInt(hdr, int64(sTCPAddr.Port), 10)
		hdr = append(hdr, ' ')
		hdr = append(hdr, dstPort...)
		hdr = append(hdr, '\r', '\n')

		// Finally write proxy header
		_, err := dTCPConn.Write(hdr)
		if err != nil {
			log.ErrorKVs(kv.Fields{
				{K: "proxy", V: proxy.Name},
				{K: "error", V: err},
				{K: "msg", V: "output error"},
			}...)
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
	go copyConn(dTCPConn, sTCPConn, errIn, clientTimeout, proxy, true)
	go copyConn(sTCPConn, dTCPConn, errOut, serverTimeout, proxy, false)

	select {
	// Wait on input error
	case err := <-errIn:
		if err != nil {
			log.ErrorKVs(kv.Fields{
				{K: "proxy", V: proxy.Name},
				{K: "error", V: err},
				{K: "msg", V: "input error"},
			}...)
		}

	// Wait on output error
	case err := <-errOut:
		if err != nil {
			log.ErrorKVs(kv.Fields{
				{K: "proxy", V: proxy.Name},
				{K: "error", V: err},
				{K: "msg", V: "output error"},
			}...)
		}

	// Server ctx cancelled
	case <-proxy.baseCtx.Done():
		dTCPConn.Close()
		sTCPConn.Close()
	}
}

// copyConn copies from once TCPConn to another, using TCPConn's ReadFrom implementation
// to take advantage of the splice optimization. this also handles connection timeouts
func copyConn(dst *net.TCPConn, src *net.TCPConn, errChan chan error, setTimeout func(), proxy *TCPProxy, isClientToServer bool) {
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
			// 统计流量
			if isClientToServer {
				proxy.addBytesIn(n)
			} else {
				proxy.addBytesOut(n)
			}
			
			// EOF / conn close -- no error
			break
		}

		// Timeouts are expected. This indicates we
		// should check the data transfer-rate
		if err, ok := err.(net.Error); ok && err.Timeout() {
			if n < 1 {
				break
			}

			// 统计流量
			if isClientToServer {
				proxy.addBytesIn(n)
			} else {
				proxy.addBytesOut(n)
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

// addBytesIn 增加入站流量统计
func (proxy *TCPProxy) addBytesIn(n int64) {
	proxy.statsMutex.Lock()
	proxy.bytesIn += uint64(n)
	proxy.statsMutex.Unlock()
}

// addBytesOut 增加出站流量统计
func (proxy *TCPProxy) addBytesOut(n int64) {
	proxy.statsMutex.Lock()
	proxy.bytesOut += uint64(n)
	proxy.statsMutex.Unlock()
}

// getStats 获取当前统计信息
func (proxy *TCPProxy) getStats() (bytesIn uint64, bytesOut uint64, connections int64) {
	proxy.statsMutex.RLock()
	bytesIn = proxy.bytesIn
	bytesOut = proxy.bytesOut
	connections = atomic.LoadInt64(&proxy.open)
	proxy.statsMutex.RUnlock()
	return
}

// formatBytes 格式化字节数为人类可读格式
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// startStatsTimer 启动统计定时器
func (proxy *TCPProxy) startStatsTimer() {
	proxy.statsTimer = time.NewTimer(time.Minute)
	go func() {
		for {
			select {
			case <-proxy.baseCtx.Done():
				if proxy.statsTimer != nil {
					proxy.statsTimer.Stop()
				}
				return
			case <-proxy.statsTimer.C:
				bytesIn, bytesOut, conns := proxy.getStats()
				log.InfoKVs(kv.Fields{
					{K: "proxy", V: proxy.Name},
					{K: "bytes_in", V: formatBytes(bytesIn)},
					{K: "bytes_out", V: formatBytes(bytesOut)},
					{K: "active_connections", V: conns},
					{K: "msg", V: "stats"},
				}...)
				proxy.statsTimer.Reset(time.Minute)
			}
		}
	}()
}
