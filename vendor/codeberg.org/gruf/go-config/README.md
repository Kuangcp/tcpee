# go-config

A very light wrapper around pelletier's [go-toml library](https://github.com/pelletier/go-toml)
for parsing and tracking values from a configuration file.

Mostly mimics the official Go [flag library](https://golang.org/pkg/flag/) in usage.

Possible types for tracking include:
- bool
- int64
- uint64
- float64
- string
- time.Time
- time.Duration
- []interface{} (any of the above)
