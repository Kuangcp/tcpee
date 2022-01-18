package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"codeberg.org/gruf/go-config"
	"codeberg.org/gruf/go-errors"
	"codeberg.org/gruf/go-logger"
	"codeberg.org/gruf/tcpee"
)

// log is the global logger instance.
var log = logger.New(os.Stdout)

// usage prints usage string and exits with code.
func usage(code int) {
	fmt.Printf("Usage: %s [-c|--config $file]\n", os.Args[0])
	os.Exit(code)
}

// closeAll will block until all proxies have closed.
func closeAll(proxies []*tcpee.TCPProxy) {
	wg := sync.WaitGroup{}
	for _, proxy := range proxies {
		wg.Add(1)
		go func(p *tcpee.TCPProxy) {
			p.Close()
			wg.Done()
		}(proxy)
	}
	wg.Wait()
}

func main() {
	// Default configuration file location
	configFile := "/etc/tcpee.conf"

	// If we have arguments to handle, do so!
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-c", "--config":
			if len(os.Args) != 3 {
				usage(1)
			}
			configFile = os.Args[2]
		default:
			usage(1)
		}
	}

	// Read config from file
	tree := make(config.Tree)
	proxies := tree.Wildcard("*", map[string]interface{}{
		"server-timeout":   "",
		"client-timeout":   "",
		"server-keepalive": "",
		"client-keepalive": "",
		"proxy":            []interface{}{},
		"transparent":      false,
		"proxy-proto":      false,
	}, false, true)
	tree.Parse(configFile)
	tree = nil // to the GC with you!

	running := []*tcpee.TCPProxy{}
	for name, details := range *proxies {
		// Define used values
		var sTimeout, cTimeout time.Duration
		var sKeepAlive, cKeepAlive time.Duration
		var proxyProto bool
		var str string
		var err error

		// Parse provided details
		str, _ = details["server-timeout"].(string)
		sTimeout, err = time.ParseDuration(str)
		if err != nil {
			log.Fatalf("Failed parsing server-timeout: %v", err)
		}
		str, _ = details["client-timeout"].(string)
		cTimeout, err = time.ParseDuration(str)
		if err != nil {
			log.Fatalf("Failed parsing client-timeout: %v", err)
		}
		str, _ = details["server-keepalive"].(string)
		sKeepAlive, err = time.ParseDuration(str)
		if err != nil {
			log.Fatalf("Failed parsing server-keepalive: %v", err)
		}
		str, _ = details["client-keepalive"].(string)
		cKeepAlive, err = time.ParseDuration(str)
		if err != nil {
			log.Fatalf("Failed parsing client-keepalive: %v", err)
		}
		proxyProto, _ = details["proxy-proto"].(bool)

		// Create new proxy server
		log.Printf("Starting proxy \"%s\"", name)
		proxy := tcpee.TCPProxy{
			Name:            name,
			Logger:          log,
			ProxyProto:      proxyProto,
			ClientKeepAlive: cKeepAlive,
			ServerKeepAlive: sKeepAlive,
			ClientTimeout:   cTimeout,
			ServerTimeout:   sTimeout,
		}

		// Iter supplied proxying addresses
		for _, entry := range details["proxy"].([]interface{}) {
			entry, _ := entry.(string)

			// Separate src + dst addresses
			split := strings.Split(entry, " -> ")
			if len(split) != 2 {
				log.Fatal(`Bad proxy configuration, expect "{src} -> {dst}"`)
			}

			// Start proxying!
			go func() {
				err := proxy.Proxy(split[0], split[1])
				if err != nil &&
					!errors.Is(err, tcpee.ErrProxyClosed) {
					closeAll(running)
					logger.Fatal(err)
				}
			}()
		}

		// Add to running proxies
		running = append(running, &proxy)
	}

	// Wait on OS signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	sig := <-signals

	log.Printf("Signal %v received, closing proxies...", sig)
	go func() {
		// Close all + exit
		closeAll(running)
		os.Exit(0)
	}()

	// Give time to finish
	sleep := 30 * time.Second
	time.Sleep(sleep)
	log.Fatal("Proxies still running after %v, forcibly closing", sleep)
}
