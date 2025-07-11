//go:build kvformat
// +build kvformat

package kv

import (
	"codeberg.org/gruf/go-byteutil"
	"codeberg.org/gruf/go-kv/format"
)

// AppendFormat will append formatted format of Field to 'buf'. See .String() for details.
func (f Field) AppendFormat(buf *byteutil.Buffer) {
	var fmtstr string
	if f.X /* verbose */ {
		fmtstr = "{:?}"
	} else /* regular */ {
		fmtstr = "{:v}"
	}
	appendQuoteKey(buf, f.K)
	buf.WriteByte('=')
	format.Appendf(buf, fmtstr, f.V)
}

// Value returns the formatted value string of this Field.
func (f Field) Value() string {
	var fmtstr string
	if f.X /* verbose */ {
		fmtstr = "{:?}"
	} else /* regular */ {
		fmtstr = "{:v}"
	}
	buf := byteutil.Buffer{B: make([]byte, 0, bufsize/2)}
	format.Appendf(&buf, fmtstr, f.V)
	return buf.String()
}
