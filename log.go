package tcpee

type Logger interface {
	Printf(string, ...interface{})
}

type nopLogger struct{}

func (l *nopLogger) Printf(string, ...interface{}) {}
