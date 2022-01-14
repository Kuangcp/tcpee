package logger

import (
	"fmt"

	"codeberg.org/gruf/go-format"
)

// Formatter represents a means of print formatted values and format strings to an io.Writer.
type Formatter interface {
	Append(*format.Buffer, ...interface{})
	Appendf(*format.Buffer, string, ...interface{})
}

// Fmt returns a `fmt` (from the std library) based Formatter instance.
func Fmt() Formatter {
	return defaultFmt
}

// Format returns a `go-format` based Formatter instance, please note this
// uses Rust-style printf formatting directives and will not be compatible
// with existing format statements.
func Format() Formatter {
	return defaultFormat
}

// defaultFmt is the global fmtIface instance.
var defaultFmt = &fmtIface{}

// defaultFormat is the global format.Formatter instance.
var defaultFormat = &formatterIface{}

type fmtIface struct{}

func (*fmtIface) Append(buf *format.Buffer, v ...interface{}) {
	fmt.Fprint(buf, v...)
}

func (*fmtIface) Appendf(buf *format.Buffer, s string, a ...interface{}) {
	fmt.Fprintf(buf, s, a...)
}

type formatterIface struct{}

func (*formatterIface) Append(buf *format.Buffer, v ...interface{}) {
	format.Append(buf, v...)
}

func (*formatterIface) Appendf(buf *format.Buffer, s string, a ...interface{}) {
	format.Appendf(buf, s, a...)
}
