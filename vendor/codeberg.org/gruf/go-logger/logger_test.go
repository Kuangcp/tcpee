package logger_test

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
	"unsafe"

	"codeberg.org/gruf/go-logger"
	"github.com/sirupsen/logrus"
)

var (
	testShortString = "hello world!"
	testLongString  = `
	GNU AFFERO GENERAL PUBLIC LICENSE
	Version 3, 19 November 2007

Copyright (C) 2007 Free Software Foundation, Inc. <https://fsf.org/>
Everyone is permitted to copy and distribute verbatim copies
of this license document, but changing it is not allowed.

		 Preamble

The GNU Affero General Public License is a free, copyleft license for
software and other kinds of works, specifically designed to ensure
cooperation with the community in the case of network server software.

The licenses for most software and other practical works are designed
to take away your freedom to share and change the works.  By contrast,
our General Public Licenses are intended to guarantee your freedom to
share and change all versions of a program--to make sure it remains free
software for all its users.

When we speak of free software, we are referring to freedom, not
price.  Our General Public Licenses are designed to make sure that you
have the freedom to distribute copies of free software (and charge for
them if you wish), that you receive source code or can get it if you
want it, that you can change the software or use pieces of it in new
free programs, and that you know you can do these things.

Developers that use our General Public Licenses protect your rights
with two steps: (1) assert copyright on the software, and (2) offer
you this License which gives you legal permission to copy, distribute
and/or modify the software.

A secondary benefit of defending all users' freedom is that
improvements made in alternate versions of the program, if they
receive widespread use, become available for other developers to
incorporate.  Many developers of free software are heartened and
encouraged by the resulting cooperation.  However, in the case of
software used on network servers, this result may fail to come about.
The GNU General Public License permits making a modified version and
letting the public access it on a server without ever releasing its
source code to the public.

The GNU Affero General Public License is designed specifically to
ensure that, in such cases, the modified source code becomes available
to the community.  It requires the operator of a network server to
provide the source code of the modified version running there to the
users of that server.  Therefore, public use of a modified version, on
a publicly accessible server, gives the public access to the source
code of the modified version.

An older license, called the Affero General Public License and
published by Affero, was designed to accomplish similar goals.  This is
a different license, not a version of the Affero GPL, but Affero has
released a new version of the Affero GPL which permits relicensing under
this license.
`

	testSlice1 = []string{"one", "two", "three", "four", "five"}
	testSlice2 = []int{1, 2, 3, 4, 5, 6, 7, 8}
	testSlice3 = []bool{true, false, false, true, false}
	testSlice4 = []interface{}{"one", 2, "three", false, true, -1.0, -2.0, -3.0}

	testMap1 = map[string]string{"key": "value", "value": "key", "wait": "that's not how this works", "ohno": "ohyes"}
	testMap2 = map[string]int{"1": 1, "2": 2, "3": 3, "4": 4, "5": 5}
	testMap3 = map[string]bool{"true": true, "false": false, "nottrue": false, "notfalse": true}
	testMap4 = map[string]interface{}{"hello": "world", "1": 1, "true": true, "slice": []int{1, 2, 3, 4}, "nil": nil, "weird": map[string]string{"recursion": "recursion..."}}

	testInt1   = 999
	testInt2   = -999
	testFloat1 = 1000.0
	testFloat2 = -1000.0

	testShortArgs1 = []interface{}{testInt1, testSlice2, testFloat1}
	testShortArgs2 = []interface{}{testShortString, testFloat2}

	testLongArgs1 = []interface{}{testLongString, testMap4, testInt1, testInt2, testSlice1, testSlice3}
	testLongArgs2 = []interface{}{testShortString, testFloat1, testInt2, testMap1, testMap2, testSlice4, testSlice3, testFloat2}

	testShortFmt1        = "Hello world! Here's %d an int. Now float %f"
	testShortFmt2        = "And again... %s a string! Also a bool slice %v"
	testShortFormat1     = "Hello world! Here's {} an int. Now float {}"
	testShortFormat2     = "And again... {} a string! Also a bool slice {}"
	testShortFormat1Args = []interface{}{testInt1, testFloat1}
	testShortFormat2Args = []interface{}{testShortString, testSlice3}

	testLongFmt1        = "Oh boy %v here is %v a bunch of %v args %s. This is wild! %v"
	testLongFmt2        = "%d %f %v not even trying with this one are we %v %v %s"
	testLongFormat1     = "Oh boy {} here is {} a bunch of {} args {}. This is wild! {}"
	testLongFormat2     = "{} {} {} not even trying with this one are we {} {} {}"
	testLongFormat1Args = []interface{}{testMap1, testSlice4, testSlice2, testLongString, testMap4}
	testLongFormat2Args = []interface{}{testInt2, testFloat2, testMap3, testMap2, testSlice1, testShortString}
)

func newLogger() *logger.Logger {
	return logger.NewWith(0, true, logger.Fmt(), 512, logger.AddSafety(io.Discard))
}

func newStdLogger() *log.Logger {
	return log.New(ioutil.Discard, "", log.LstdFlags)
}

func newLogrusLogger() *logrus.Logger {
	l := logrus.New()
	l.Out = ioutil.Discard
	return l
}

func TestArgPrinting(t *testing.T) {
	l := newLogger()

	l.Entry().Append(
		uint8(1),
		uint16(1),
		uint32(1),
		uint64(1),
		int8(1),
		int16(1),
		int32(1),
		int64(1),
		float32(1),
		float64(1),
		"hello",
		[]byte("hello"),
		[]string{"hello", "world"},
		[]interface{}{"hello", "world", 42},
		map[string]string{"hello": "world"},
		time.Time{},
		&time.Time{},
		(*time.Time)(nil),
		(***string)(nil),
		nil,
		unsafe.Pointer(&testSlice1),
		unsafe.Pointer(nil),
		(unsafe.Pointer)(nil),
		uintptr(0),
		errors.New("error"),
		time.Duration(0),
		http.Request{},
		&http.Request{},
		func() {},
		(func())(nil),
		make(chan struct{}),
		(chan struct{})(nil),
		([]byte)(nil),
		[]byte{},
		http.Server{},
	).Send()
}

func BenchmarkLoggerMultiString(b *testing.B) {
	l := newLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Info(testShortString)
			l.Info(testLongString)
			l.Info(testShortString)
			l.Info(testLongString)
			l.Info(testShortArgs1...)
			l.Info(testLongArgs1...)
			l.Info(testShortArgs2...)
			l.Info(testLongArgs2...)
			l.Infof(testShortFormat1, testShortFormat1Args...)
			l.Infof(testLongFormat1, testLongFormat1Args...)
			l.Infof(testShortFormat2, testShortFormat2Args...)
			l.Infof(testLongFormat2, testLongFormat2Args...)
		}
	})
}

func BenchmarkLogMultiString(b *testing.B) {
	l := newStdLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Print(testShortString)
			l.Print(testLongString)
			l.Print(testShortString)
			l.Print(testLongString)
			l.Print(testShortArgs1...)
			l.Print(testLongArgs1...)
			l.Print(testShortArgs2...)
			l.Print(testLongArgs2...)
			l.Printf(testShortFmt1, testShortFormat1Args...)
			l.Printf(testLongFmt1, testLongFormat1Args...)
			l.Printf(testShortFmt2, testShortFormat2Args...)
			l.Printf(testLongFmt2, testLongFormat2Args...)
		}
	})
}

func BenchmarkLogrusMultiString(b *testing.B) {
	l := newLogrusLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Print(testShortString)
			l.Print(testLongString)
			l.Print(testShortString)
			l.Print(testLongString)
			l.Print(testShortArgs1...)
			l.Print(testLongArgs1...)
			l.Print(testShortArgs2...)
			l.Print(testLongArgs2...)
			l.Printf(testShortFormat1, testShortFormat1Args...)
			l.Printf(testLongFmt1, testLongFormat1Args...)
			l.Printf(testShortFmt2, testShortFormat2Args...)
			l.Printf(testLongFmt2, testLongFormat2Args...)
		}
	})
}

func BenchmarkLoggerShortString(b *testing.B) {
	l := newLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Info(testShortString)
			l.Info(testShortString)
		}
	})
}

func BenchmarkLogShortString(b *testing.B) {
	l := newStdLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Print(testShortString)
			l.Print(testShortString)
		}
	})
}

func BenchmarkLogrusShortString(b *testing.B) {
	l := newLogrusLogger()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Print(testShortString)
			l.Print(testShortString)
		}
	})
}

func BenchmarkLoggerLongString(b *testing.B) {
	l := newLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Info(testLongString)
			l.Info(testLongString)
		}
	})
}

func BenchmarkLogLongString(b *testing.B) {
	l := newStdLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Print(testLongString)
			l.Print(testLongString)
		}
	})
}

func BenchmarkLogrusLongString(b *testing.B) {
	l := newLogrusLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Print(testLongString)
			l.Print(testLongString)
		}
	})
}

func BenchmarkLoggerShortArgs(b *testing.B) {
	l := newLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Info(testShortArgs1...)
			l.Info(testShortArgs2...)
		}
	})
}

func BenchmarkLogShortArgs(b *testing.B) {
	l := newStdLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Print(testShortArgs1...)
			l.Print(testShortArgs2...)
		}
	})
}

func BenchmarkLogrusShortArgs(b *testing.B) {
	l := newLogrusLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Print(testShortArgs1...)
			l.Print(testShortArgs2...)
		}
	})
}

func BenchmarkLoggerLongArgs(b *testing.B) {
	l := newLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Info(testLongArgs1...)
			l.Info(testLongArgs2...)
		}
	})
}

func BenchmarkLogLongArgs(b *testing.B) {
	l := newStdLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Print(testLongArgs1...)
			l.Print(testLongArgs2...)
		}
	})
}

func BenchmarkLogrusLongArgs(b *testing.B) {
	l := newLogrusLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Print(testLongArgs1...)
			l.Print(testLongArgs2...)
		}
	})
}

func BenchmarkLoggerShortFormat(b *testing.B) {
	l := newLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Infof(testShortFormat1, testShortFormat1Args...)
			l.Infof(testShortFormat2, testShortFormat2Args...)
		}
	})
}

func BenchmarkLogShortFormat(b *testing.B) {
	l := newStdLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Printf(testShortFmt1, testShortFormat1Args...)
			l.Printf(testShortFmt2, testShortFormat2Args...)
		}
	})
}

func BenchmarkLogrusShortFormat(b *testing.B) {
	l := newLogrusLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Printf(testShortFmt1, testShortFormat1Args...)
			l.Printf(testShortFmt2, testShortFormat2Args...)
		}
	})
}

func BenchmarkLoggerLongFormat(b *testing.B) {
	l := newLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Infof(testLongFormat1, testLongFormat1Args...)
			l.Infof(testLongFormat2, testLongFormat2Args...)
		}
	})
}

func BenchmarkLogLongFormat(b *testing.B) {
	l := newStdLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Printf(testLongFmt1, testLongFormat1Args...)
			l.Printf(testLongFmt2, testLongFormat2Args...)
		}
	})
}

func BenchmarkLogrusLongFormat(b *testing.B) {
	l := newLogrusLogger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Printf(testLongFmt1, testLongFormat1Args...)
			l.Printf(testLongFmt2, testLongFormat2Args...)
		}
	})
}
