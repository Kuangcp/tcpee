A simple multi-threaded TCP proxy in Go with support for setting
server / client keepalives and timeouts. Supports multiple configurations
and multiple blocks of configurations, e.g. different settings per
service you are proxying to.

See `example.conf` for an example configuration file.