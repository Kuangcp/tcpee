package format_test

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"codeberg.org/gruf/go-format"
)

// A is a test structure.
type A struct {
	A string
	B *string
}

// B is a test structure with unexported fields.
type B struct {
	a string
	b *string
}

// C is a test structure with fmt.Stringer implementation.
type C struct {
}

func (c C) String() string {
	return "c.String()"
}

// D is a test structure with format.Formattable implementation.
type D struct {
}

func (d D) AppendFormat(b []byte) []byte {
	return append(b, `d.AppendFormat()`...)
}

// PanicTest is a test structure with a fmt.Stringer implementation that panics.
type PanicTest int

func (t PanicTest) String() string {
	panic(`oh no! a panic`)
}

// appendTests are just a list of arguments to
// format and append to buffer, these test results
// are not checked these are to ensure safety
var appendTests = []interface{}{}

// printfTests provide a list of format strings, with
// single argument and expected results. Intended to
// test that operands produce expected results.
var printfTests = []struct {
	Fmt string
	Arg interface{}
	Out string
}{
	// default format
	{
		Fmt: `{}`,
		Arg: nil,
		Out: `nil`,
	},
	{
		Fmt: `{}`,
		Arg: (*int)(nil),
		Out: `nil`,
	},
	{
		Fmt: `{}`,
		Arg: time.Second,
		Out: time.Second.String(),
	},
	{
		Fmt: `{}`,
		Arg: `hello`,
		Out: `hello`,
	},
	{
		Fmt: `{}`,
		Arg: `hello world`,
		Out: `hello world`,
	},
	{
		Fmt: `{}`,
		Arg: "hello\nworld\n",
		Out: "hello\nworld\n",
	},
	{
		Fmt: `{}`,
		Arg: errors.New("error!"),
		Out: `error!`,
	},
	{
		Fmt: `{}`,
		Arg: A{},
		Out: `{A="" B=nil}`,
	},
	{
		Fmt: `{}`,
		Arg: B{},
		Out: `{a="" b=nil}`,
	},
	{
		Fmt: `{}`,
		Arg: C{},
		Out: `c.String()`,
	},
	{
		Fmt: `{}`,
		Arg: D{},
		Out: `d.AppendFormat()`,
	},
	{
		Fmt: `{}`,
		Arg: PanicTest(0),
		Out: `!{PANIC="oh no! a panic"}`,
	},
	{
		Fmt: `{}`,
		Arg: int8(0),
		Out: `0`,
	},
	{
		Fmt: `{}`,
		Arg: int16(0),
		Out: `0`,
	},
	{
		Fmt: `{}`,
		Arg: int32(0),
		Out: `0`,
	},
	{
		Fmt: `{}`,
		Arg: int64(0),
		Out: `0`,
	},
	{
		Fmt: `{}`,
		Arg: uint8(0), // uint8=byte
		Out: "\x00",   // formatted as byte
	},
	{
		Fmt: `{}`,
		Arg: uint16(0),
		Out: `0`,
	},
	{
		Fmt: `{}`,
		Arg: uint32(0),
		Out: `0`,
	},
	{
		Fmt: `{}`,
		Arg: uint64(0),
		Out: `0`,
	},
	{
		Fmt: `{}`,
		Arg: float32(0.0),
		Out: `0`,
	},
	{
		Fmt: `{}`,
		Arg: float64(0.0),
		Out: `0`,
	},
	{
		Fmt: `{}`,
		Arg: complex64(0),
		Out: `0+0i`,
	},
	{
		Fmt: `{}`,
		Arg: complex128(0),
		Out: `0+0i`,
	},
	{
		Fmt: `{}`,
		Arg: []byte(`hello world`),
		Out: `[h,e,l,l,o, ,w,o,r,l,d]`,
	},
	{
		Fmt: `{}`,
		Arg: byte('?'),
		Out: `?`,
	},

	// key format
	{
		Fmt: `{:k}`,
		Arg: nil,
		Out: `nil`,
	},
	{
		Fmt: `{:k}`,
		Arg: (*int)(nil),
		Out: `nil`,
	},
	{
		Fmt: `{:k}`,
		Arg: time.Second,
		Out: time.Second.String(),
	},
	{
		Fmt: `{:k}`,
		Arg: `hello`,
		Out: `hello`,
	},
	{
		Fmt: `{:k}`,
		Arg: `hello world`,
		Out: `"hello world"`,
	},
	{
		Fmt: `{:k}`,
		Arg: "hello\nworld\n",
		Out: `"hello\nworld\n"`,
	},
	{
		Fmt: `{:k}`,
		Arg: errors.New("error!"),
		Out: `error!`,
	},
	{
		Fmt: `{:k}`,
		Arg: A{},
		Out: `{A="" B=nil}`,
	},
	{
		Fmt: `{:k}`,
		Arg: B{},
		Out: `{a="" b=nil}`,
	},
	{
		Fmt: `{:k}`,
		Arg: C{},
		Out: `c.String()`,
	},
	{
		Fmt: `{:k}`,
		Arg: D{},
		Out: `d.AppendFormat()`,
	},
	{
		Fmt: `{:k}`,
		Arg: PanicTest(0),
		Out: `!{PANIC="oh no! a panic"}`,
	},
	{
		Fmt: `{:k}`,
		Arg: []byte(`hello world`),
		Out: `[h,e,l,l,o, ,w,o,r,l,d]`,
	},
	{
		Fmt: `{:k}`,
		Arg: byte('?'),
		Out: `?`,
	},

	// value format
	{
		Fmt: `{:v}`,
		Arg: nil,
		Out: `nil`,
	},
	{
		Fmt: `{:v}`,
		Arg: (*int)(nil),
		Out: `nil`,
	},
	{
		Fmt: `{:v}`,
		Arg: time.Second,
		Out: `"` + time.Second.String() + `"`,
	},
	{
		Fmt: `{:v}`,
		Arg: `hello`,
		Out: `"hello"`,
	},
	{
		Fmt: `{:v}`,
		Arg: `hello world`,
		Out: `"hello world"`,
	},
	{
		Fmt: `{:v}`,
		Arg: "hello\nworld\n",
		Out: `"hello\nworld\n"`,
	},
	{
		Fmt: `{:v}`,
		Arg: errors.New("error!"),
		Out: `"error!"`,
	},
	{
		Fmt: `{:v}`,
		Arg: A{},
		Out: `{A="" B=nil}`,
	},
	{
		Fmt: `{:v}`,
		Arg: B{},
		Out: `{a="" b=nil}`,
	},
	{
		Fmt: `{:v}`,
		Arg: C{},
		Out: `"c.String()"`,
	},
	{
		Fmt: `{:v}`,
		Arg: D{},
		Out: `d.AppendFormat()`,
	},
	{
		Fmt: `{:v}`,
		Arg: PanicTest(0),
		Out: `!{PANIC="oh no! a panic"}`,
	},
	{
		Fmt: `{:v}`,
		Arg: []byte(`hello world`),
		Out: `[h,e,l,l,o, ,w,o,r,l,d]`,
	},
	{
		Fmt: `{:v}`,
		Arg: byte('?'),
		Out: `'?'`,
	},

	// verbose format
	{
		Fmt: `{:?}`,
		Arg: nil,
		Out: `nil`,
	},
	{
		Fmt: `{:?}`,
		Arg: (*int)(nil),
		Out: `(*int)(nil)`,
	},
	{
		Fmt: `{:?}`,
		Arg: time.Second,
		Out: strconv.FormatInt(int64(time.Second), 10),
	},
	{
		Fmt: `{:?}`,
		Arg: `hello`,
		Out: `"hello"`,
	},
	{
		Fmt: `{:?}`,
		Arg: `hello world`,
		Out: `"hello world"`,
	},
	{
		Fmt: `{:?}`,
		Arg: "hello\nworld\n",
		Out: "\"hello\nworld\n\"",
	},
	{
		Fmt: `{:?}`,
		Arg: errors.New("error!"), //nolint
		Out: `*errors.errorString{s="error!"}`,
	},
	{
		Fmt: `{:?}`,
		Arg: A{},
		Out: `format_test.A{A="" B=(*string)(nil)}`,
	},
	{
		Fmt: `{:?}`,
		Arg: B{},
		Out: `format_test.B{a="" b=(*string)(nil)}`,
	},
	{
		Fmt: `{:?}`,
		Arg: C{},
		Out: `format_test.C{}`,
	},
	{
		Fmt: `{:?}`,
		Arg: D{},
		Out: `format_test.D{}`,
	},
	{
		Fmt: `{:?}`,
		Arg: PanicTest(0),
		Out: `0`,
	},
	{
		Fmt: `{:?}`,
		Arg: []byte(`hello world`),
		Out: `[h,e,l,l,o, ,w,o,r,l,d]`,
	},
	{
		Fmt: `{:?}`,
		Arg: byte('?'),
		Out: `'?'`,
	},
}

// printfMultiTests provide a list of more complex format
// strings, with any number of arguments and expected results.
// Intended to test that string format parsing and formatting
// produces expected results.
var printfMultiTests = []struct {
	Fmt string
	Arg []interface{}
	Out string
}{}

func TestAppend(t *testing.T) {
	buf := format.Buffer{}
	for _, arg := range appendTests {
		format.Append(&buf, arg)
		buf.Reset()
	}
}

func TestPrintf(t *testing.T) {
	for _, test := range printfTests {
		out := format.Sprintf(test.Fmt, test.Arg)
		if out != test.Out {
			t.Fatalf("printf did not produce expected results\n"+
				"input={%q, %#v}\n"+
				"expect=%s\n"+
				"actual=%s\n",
				test.Fmt,
				test.Arg,
				test.Out,
				out,
			)
		}
	}
}

func TestPrintfMulti(t *testing.T) {
	for _, test := range printfMultiTests {
		out := format.Sprintf(test.Fmt, test.Arg...)
		if out != test.Out {
			t.Fatalf("printf (multi arg) did not produce expected results\n"+
				"input={%q, %#v}\n"+
				"expect=%s\n"+
				"actual=%s\n",
				test.Fmt,
				test.Arg,
				test.Out,
				out,
			)
		}
	}
}

// func BenchmarkFprintf(b *testing.B) {
// 	b.ReportAllocs()
// 	b.ResetTimer()
// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			for i := range fmtTests {
// 				// Perform both non-verbose and verbose
// 				format.Fprintf(out, "{}", fmtTests[i].val)
// 				format.Fprintf(out, "{:?}", fmtTests[i].val)
// 			}
// 		}
// 	})
// }

// func BenchmarkFmtFprintf(b *testing.B) {
// 	b.ReportAllocs()
// 	b.ResetTimer()
// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			for i := range fmtTests {
// 				// Perform both non-verbose and verbose
// 				fmt.Fprintf(out, "%v", fmtTests[i].val)
// 				fmt.Fprintf(out, "%#v", fmtTests[i].val)
// 			}
// 		}
// 	})
// }
