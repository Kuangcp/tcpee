package format

import (
	"reflect"
	"strconv"
	"unicode/utf8"

	"codeberg.org/gruf/go-byteutil"
)

const (
	// Flag bit constants, note they are prioritised in this order.
	IsKeyBit = uint8(1) << 0 // set to indicate key formatting
	VboseBit = uint8(1) << 1 // set to indicate verbose formatting
	IsValBit = uint8(1) << 2 // set to indicate value formatting
	PanicBit = uint8(1) << 3 // set after panic to prevent recursion)
)

// Format provides formatting of values into a Buffer.
type Format struct {
	// Flags are the currently set value flags.
	Flags uint8

	// Derefs is the current value dereference count.
	Derefs uint8

	// CurDepth is the current Format iterator depth.
	CurDepth uint8

	// VType is the current value type.
	VType string

	// Config is the set Formatter config (MUST NOT be nil).
	Config *Formatter

	// Buffer is the currently set output buffer.
	Buffer *byteutil.Buffer
}

// AtMaxDepth returns whether format is currently at max depth.
func (f Format) AtMaxDepth() bool {
	return f.CurDepth > f.Config.MaxDepth
}

// Key returns whether the isKey flag is set.
func (f Format) Key() bool {
	return (f.Flags & IsKeyBit) != 0
}

// Value returns whether the isVal flag is set.
func (f Format) Value() bool {
	return (f.Flags & IsValBit) != 0
}

// Verbose returns whether the verbose flag is set.
func (f Format) Verbose() bool {
	return (f.Flags & VboseBit) != 0
}

// Panic returns whether the panic flag is set.
func (f Format) Panic() bool {
	return (f.Flags & PanicBit) != 0
}

// SetKey returns format instance with the IsKey bit set to true,
// note this resets the dereference count.
func (f Format) SetKey() Format {
	flags := f.Flags | IsKeyBit
	flags &= ^IsValBit
	return Format{
		Flags:    flags,
		CurDepth: f.CurDepth,
		Config:   f.Config,
		Buffer:   f.Buffer,
	}
}

// SetValue returns format instance with the IsVal bit set to true,
// note this resets the dereference count.
func (f Format) SetValue() Format {
	flags := f.Flags | IsValBit
	flags &= ^IsKeyBit
	return Format{
		Flags:    flags,
		CurDepth: f.CurDepth,
		Config:   f.Config,
		Buffer:   f.Buffer,
	}
}

// SetVerbose returns format instance with the Vbose bit set to true,
// note this resets the dereference count.
func (f Format) SetVerbose() Format {
	return Format{
		Flags:    f.Flags | VboseBit,
		CurDepth: f.CurDepth,
		Config:   f.Config,
		Buffer:   f.Buffer,
	}
}

// SetPanic returns format instance with the panic bit set to true,
// note this resets the dereference count and sets IsVal (unsetting IsKey) bit.
func (f Format) SetPanic() Format {
	flags := f.Flags | PanicBit
	flags |= IsValBit
	flags &= ^IsKeyBit
	return Format{
		Flags:    flags,
		CurDepth: f.CurDepth,
		Config:   f.Config,
		Buffer:   f.Buffer,
	}
}

// IncrDepth returns format instance with depth incremented and derefs reset.
func (f Format) IncrDepth() Format {
	return Format{
		Flags:    f.Flags,
		Derefs:   f.Derefs,
		CurDepth: f.CurDepth + 1,
		Config:   f.Config,
		Buffer:   f.Buffer,
	}
}

// IncrDerefs returns format instance with dereference count incremented.
func (f Format) IncrDerefs() Format {
	return Format{
		Flags:    f.Flags,
		Derefs:   f.Derefs + 1,
		CurDepth: f.CurDepth,
		Config:   f.Config,
		Buffer:   f.Buffer,
	}
}

func (f Format) AppendType() {
	for i := uint8(0); i < f.Derefs; i++ {
		// add each dereference
		f.Buffer.WriteByte('*')
	}
	f.Buffer.WriteString(f.VType)
}

func (f Format) AppendNil() {
	if !f.Verbose() {
		f.Buffer.WriteString(`nil`)
		return
	}

	// Append nil with type
	f.Buffer.WriteByte('(')
	f.AppendType()
	f.Buffer.WriteString(`)(nil)`)
}

func (f Format) AppendByte(b byte) {
	switch {
	// Always quoted
	case f.Key():
		f.Buffer.WriteString(`'` + byte2str(b) + `'`)

	// Always quoted ASCII with type
	case f.Verbose():
		f._AppendPrimitiveTyped(func(f Format) {
			f.Buffer.WriteString(`'` + byte2str(b) + `'`)
		})

	// Always quoted
	case f.Value():
		f.Buffer.WriteString(`'` + byte2str(b) + `'`)

	// Append as raw byte
	default:
		f.Buffer.WriteByte(b)
	}
}

func (f Format) AppendBytes(b []byte) {
	switch {
	// Bytes CAN be nil formatted
	case b == nil:
		f.AppendNil()

	// Handle bytes as string key
	case f.Key():
		f.AppendStringKey(b2s(b))

	// Append as separate ASCII quoted bytes in slice
	case f.Verbose():
		f._AppendArrayTyped(func(f Format) {
			for i := 0; i < len(b); i++ {
				f.Buffer.WriteString(`'` + byte2str(b[i]) + `'`)
				f.Buffer.WriteByte(',')
			}
			if len(b) > 0 {
				f.Buffer.Truncate(1)
			}
		})

	// Append as quoted string
	case f.Value():
		f.AppendStringQuoted(b2s(b))

	// Append as raw bytes
	default:
		f.Buffer.Write(b)
	}
}

func (f Format) AppendRune(r rune) {
	switch {
	// Quoted only if spaces/requires escaping
	case f.Key():
		f.AppendRuneKey(r)

	// Always quoted ASCII with type
	case f.Verbose():
		f._AppendPrimitiveTyped(func(f Format) {
			f.Buffer.B = strconv.AppendQuoteRuneToASCII(f.Buffer.B, r)
		})

	// Always quoted value
	case f.Value():
		f.Buffer.B = strconv.AppendQuoteRune(f.Buffer.B, r)

	// Append as raw rune
	default:
		f.Buffer.WriteRune(r)
	}
}

func (f Format) AppendRuneKey(r rune) {
	if utf8.RuneLen(r) > 1 && (r < ' ' && r != '\t') || r == '`' || r == '\u007F' {
		// Quote and escape this rune
		f.Buffer.B = strconv.AppendQuoteRuneToASCII(f.Buffer.B, r)
	} else {
		// Simply append rune
		f.Buffer.WriteRune(r)
	}
}

func (f Format) AppendRunes(r []rune) {
	switch {
	// Runes CAN be nil formatted
	case r == nil:
		f.AppendNil()

	// Handle bytes as string key
	case f.Key():
		f.AppendStringKey(string(r))

	// Append as separate ASCII quoted bytes in slice
	case f.Verbose():
		f._AppendArrayTyped(func(f Format) {
			for i := 0; i < len(r); i++ {
				f.Buffer.B = strconv.AppendQuoteRuneToASCII(f.Buffer.B, r[i])
				f.Buffer.WriteByte(',')
			}
			if len(r) > 0 {
				f.Buffer.Truncate(1)
			}
		})

	// Append as quoted string
	case f.Value():
		f.AppendStringQuoted(string(r))

	// Append as raw bytes
	default:
		for i := 0; i < len(r); i++ {
			f.Buffer.WriteRune(r[i])
		}
	}
}

func (f Format) AppendString(s string) {
	switch {
	// Quoted only if spaces/requires escaping
	case f.Key():
		f.AppendStringKey(s)

	// Always quoted with type
	case f.Verbose():
		f._AppendPrimitiveTyped(func(f Format) {
			f.AppendStringQuoted(s)
		})

	// Always quoted string
	case f.Value():
		f.AppendStringQuoted(s)

	// All else
	default:
		f.Buffer.WriteString(s)
	}
}

func (f Format) AppendStringKey(s string) {
	if !strconv.CanBackquote(s) {
		// Requires quoting AND escaping
		f.Buffer.B = strconv.AppendQuote(f.Buffer.B, s)
	} else if ContainsDoubleQuote(s) {
		// Contains double quotes, needs escaping
		f.Buffer.B = AppendEscape(f.Buffer.B, s)
	} else if len(s) < 1 || ContainsSpaceOrTab(s) {
		// Contains space, needs quotes
		f.Buffer.WriteString(`"` + s + `"`)
	} else {
		// All else write as-is
		f.Buffer.WriteString(s)
	}
}

func (f Format) AppendStringQuoted(s string) {
	if !strconv.CanBackquote(s) {
		// Requires quoting AND escaping
		f.Buffer.B = strconv.AppendQuote(f.Buffer.B, s)
	} else if ContainsDoubleQuote(s) {
		// Contains double quotes, needs escaping
		f.Buffer.B = append(f.Buffer.B, '"')
		f.Buffer.B = AppendEscape(f.Buffer.B, s)
		f.Buffer.B = append(f.Buffer.B, '"')
	} else {
		// Simply append with quotes
		f.Buffer.WriteString(`"` + s + `"`)
	}
}

func (f Format) AppendBool(b bool) {
	if f.Verbose() {
		// Append as bool with type information
		f._AppendPrimitiveTyped(func(f Format) {
			f.Buffer.B = strconv.AppendBool(f.Buffer.B, b)
		})
	} else {
		// Simply append as bool
		f.Buffer.B = strconv.AppendBool(f.Buffer.B, b)
	}
}

func (f Format) AppendInt(i int64) {
	f._AppendPrimitiveType(func(f Format) {
		f.Buffer.B = strconv.AppendInt(f.Buffer.B, i, 10)
	})
}

func (f Format) AppendUint(u uint64) {
	f._AppendPrimitiveType(func(f Format) {
		f.Buffer.B = strconv.AppendUint(f.Buffer.B, u, 10)
	})
}

func (f Format) AppendFloat(l float64) {
	f._AppendPrimitiveType(func(f Format) {
		f.AppendFloatValue(l)
	})
}

func (f Format) AppendFloatValue(l float64) {
	f.Buffer.B = strconv.AppendFloat(f.Buffer.B, l, 'f', -1, 64)
}

func (f Format) AppendComplex(c complex128) {
	f._AppendPrimitiveType(func(f Format) {
		f.AppendFloatValue(real(c))
		f.Buffer.WriteByte('+')
		f.AppendFloatValue(imag(c))
		f.Buffer.WriteByte('i')
	})
}

func (f Format) AppendPtr(u uint64) {
	f._AppendPtrType(func(f Format) {
		if u == 0 {
			// Append as nil
			f.Buffer.WriteString(`nil`)
		} else {
			// Append as hex number
			f.Buffer.WriteString(`0x`)
			f.Buffer.B = strconv.AppendUint(f.Buffer.B, u, 16)
		}
	})
}

func (f Format) AppendInterfaceOrReflect(i interface{}) {
	if !f.AppendInterface(i) {
		// Interface append failed, used reflected
		f.AppendReflectValue(reflect.ValueOf(i))
	}
}

func (f Format) AppendInterfaceOrReflectNext(v reflect.Value) {
	// Check we haven't hit max
	if f.AtMaxDepth() {
		f.Buffer.WriteString("...")
		return
	}

	// Incr the depth
	f = f.IncrDepth()

	// Make actual call
	f.AppendReflectOrInterface(v)
}

func (f Format) AppendReflectOrInterface(v reflect.Value) {
	if !v.CanInterface() ||
		!f.AppendInterface(v.Interface()) {
		// Interface append failed, use reflect
		f.AppendReflectValue(v)
	}
}

func (f Format) AppendInterface(i interface{}) bool {
	switch i := i.(type) {
	// Reflect types
	case reflect.Type:
		f.AppendReflectType(i)
	case reflect.Value:
		f.Buffer.WriteString(`reflect.Value`)
		f.Buffer.WriteByte('(')
		f.Buffer.WriteString(i.String())
		f.Buffer.WriteByte(')')

	// Bytes, runes and string types
	case rune:
		f.VType = `int32`
		f.AppendRune(i)
	case []rune:
		f.VType = `[]int32`
		f.AppendRunes(i)
	case byte:
		f.VType = `uint8`
		f.AppendByte(i)
	case []byte:
		f.VType = `[]uint8`
		f.AppendBytes(i)
	case string:
		f.VType = `string`
		f.AppendString(i)

	// Int types
	case int:
		f.VType = `int`
		f.AppendInt(int64(i))
	case int8:
		f.VType = `int8`
		f.AppendInt(int64(i))
	case int16:
		f.VType = `int16`
		f.AppendInt(int64(i))
	case int64:
		f.VType = `int64`
		f.AppendInt(int64(i))

	// Uint types
	case uint:
		f.VType = `uint`
		f.AppendUint(uint64(i))
	case uint16:
		f.VType = `uint16`
		f.AppendUint(uint64(i))
	case uint32:
		f.VType = `uint32`
		f.AppendUint(uint64(i))
	case uint64:
		f.VType = `uint64`
		f.AppendUint(uint64(i))

	// Float types
	case float32:
		f.VType = `float32`
		f.AppendFloat(float64(i))
	case float64:
		f.VType = `float64`
		f.AppendFloat(float64(i))

	// Bool type
	case bool:
		f.VType = `bool`
		f.AppendBool(i)

	// Complex types
	case complex64:
		f.VType = `complex64`
		f.AppendComplex(complex128(i))
	case complex128:
		f.VType = `complex128`
		f.AppendComplex(complex128(i))

	// Method types
	case error:
		return f._AppendMethodType(func() string {
			return i.Error()
		}, i)
	case interface{ String() string }:
		return f._AppendMethodType(func() string {
			return i.String()
		}, i)

	// No quick handler
	default:
		return false
	}

	return true
}

func (f Format) AppendReflectType(t reflect.Type) {
	f.VType = `reflect.Type`
	switch {
	case isNil(t) /* safer nil check */ :
		f.AppendNil()
	case f.Verbose():
		f.AppendType()
		f.Buffer.WriteString(`(` + t.String() + `)`)
	default:
		f.Buffer.WriteString(t.String())
	}
}

func (f Format) AppendReflectValue(v reflect.Value) {
	switch v.Kind() {
	// String/byte types
	case reflect.String:
		f.VType = v.Type().String()
		f.AppendString(v.String())
	case reflect.Uint8:
		f.VType = v.Type().String()
		f.AppendByte(byte(v.Uint()))
	case reflect.Int32:
		f.VType = v.Type().String()
		f.AppendRune(rune(v.Int()))

	// Float tpyes
	case reflect.Float32, reflect.Float64:
		f.VType = v.Type().String()
		f.AppendFloat(v.Float())

	// Int types
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int64:
		f.VType = v.Type().String()
		f.AppendInt(v.Int())

	// Uint types
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		f.VType = v.Type().String()
		f.AppendUint(v.Uint())

	// Complex types
	case reflect.Complex64, reflect.Complex128:
		f.VType = v.Type().String()
		f.AppendComplex(v.Complex())

	// Bool type
	case reflect.Bool:
		f.VType = v.Type().String()
		f.AppendBool(v.Bool())

	// Slice and array types
	case reflect.Array:
		f.AppendArray(v)
	case reflect.Slice:
		f.AppendSlice(v)

	// Map types
	case reflect.Map:
		f.AppendMap(v)

	// Struct types
	case reflect.Struct:
		f.AppendStruct(v)

	// Interface type
	case reflect.Interface:
		if v.IsNil() {
			// Append nil ptr type
			f.VType = v.Type().String()
			f.AppendNil()
		} else {
			// Append interface
			f.AppendReflectOrInterface(v.Elem())
		}

	// Deref'able ptr type
	case reflect.Ptr:
		if v.IsNil() {
			// Append nil ptr type
			f.VType = v.Type().String()
			f.AppendNil()
		} else {
			// Deref to next level
			f = f.IncrDerefs()
			f.AppendReflectOrInterface(v.Elem())
		}

	// 'raw' pointer types
	case reflect.UnsafePointer, reflect.Func, reflect.Chan:
		f.VType = v.Type().String()
		f.AppendPtr(uint64(v.Pointer()))
	case reflect.Uintptr:
		f.VType = v.Type().String()
		f.AppendPtr(v.Uint())

	// Zero reflect value
	case reflect.Invalid:
		f.Buffer.WriteString(`nil`)

	// All others
	default:
		f.VType = v.Type().String()
		f.AppendType()
	}
}

func (f Format) AppendSlice(v reflect.Value) {
	t := v.Type()

	// Get slice value type
	f.VType = t.String()

	if t.Elem().Kind() == reflect.Uint8 {
		// This is a byte slice
		f.AppendBytes(v.Bytes())
		return
	}

	if v.IsNil() {
		// Nil slice
		f.AppendNil()
		return
	}

	if f.Verbose() {
		// Append array with type information
		f._AppendArrayTyped(func(f Format) {
			f.AppendArrayElems(v)
		})
	} else {
		// Simply append array as elems
		f._AppendArray(func(f Format) {
			f.AppendArrayElems(v)
		})
	}
}

func (f Format) AppendArray(v reflect.Value) {
	// Get array value type
	f.VType = v.Type().String()

	if f.Verbose() {
		// Append array with type information
		f._AppendArrayTyped(func(f Format) {
			f.AppendArrayElems(v)
		})
	} else {
		// Simply append array as elems
		f._AppendArray(func(f Format) {
			f.AppendArrayElems(v)
		})
	}
}

func (f Format) AppendArrayElems(v reflect.Value) {
	// Get no. elems
	n := v.Len()

	// Append values
	for i := 0; i < n; i++ {
		f.SetValue().AppendInterfaceOrReflectNext(v.Index(i))
		f.Buffer.WriteByte(',')
	}

	// Drop last comma
	if n > 0 {
		f.Buffer.Truncate(1)
	}
}

func (f Format) AppendMap(v reflect.Value) {
	// Get value type
	t := v.Type()
	f.VType = t.String()

	if v.IsNil() {
		// Nil map -- no fields
		f.AppendNil()
		return
	}

	// Append field formatted map fields
	f._AppendFieldType(func(f Format) {
		f.AppendMapFields(v)
	})
}

func (f Format) AppendMapFields(v reflect.Value) {
	// Get a map iterator
	r := v.MapRange()
	n := v.Len()

	// Iterate pairs
	for r.Next() {
		f.SetKey().AppendInterfaceOrReflectNext(r.Key())
		f.Buffer.WriteByte('=')
		f.SetValue().AppendInterfaceOrReflectNext(r.Value())
		f.Buffer.WriteByte(' ')
	}

	// Drop last space
	if n > 0 {
		f.Buffer.Truncate(1)
	}
}

func (f Format) AppendStruct(v reflect.Value) {
	// Get value type
	t := v.Type()
	f.VType = t.String()

	// Append field formatted struct fields
	f._AppendFieldType(func(f Format) {
		f.AppendStructFields(t, v)
	})
}

func (f Format) AppendStructFields(t reflect.Type, v reflect.Value) {
	// Get field no.
	n := v.NumField()

	// Iterate struct fields
	for i := 0; i < n; i++ {
		vfield := v.Field(i)
		tfield := t.Field(i)

		// Append field name
		f.AppendStringKey(tfield.Name)
		f.Buffer.WriteByte('=')
		f.SetValue().AppendInterfaceOrReflectNext(vfield)

		// Append separator
		f.Buffer.WriteByte(' ')
	}

	// Drop last space
	if n > 0 {
		f.Buffer.Truncate(1)
	}
}

func (f Format) _AppendMethodType(method func() string, i interface{}) (ok bool) {
	// Verbose -- no methods
	if f.Verbose() {
		return false
	}

	// Catch nil type
	if isNil(i) {
		f.AppendNil()
		return true
	}

	// Catch any panics
	defer func() {
		if r := recover(); r != nil {
			// DON'T recurse catchPanic()
			if f.Panic() {
				panic(r)
			}

			// Attempt to decode panic into buf
			f.Buffer.WriteString(`!{PANIC=`)
			f.SetPanic().AppendInterfaceOrReflect(r)
			f.Buffer.WriteByte('}')

			// Ensure no further attempts
			// to format after return
			ok = true
		}
	}()

	// Get method result
	result := method()

	switch {
	// Append as key formatted
	case f.Key():
		f.AppendStringKey(result)

	// Append as always quoted
	case f.Value():
		f.AppendStringQuoted(result)

	// Append as-is
	default:
		f.Buffer.WriteString(result)
	}

	return true
}

// _AppendPrimitiveType is a helper to append prefix/suffix for primitives (numbers/bools/bytes/runes).
func (f Format) _AppendPrimitiveType(appendPrimitive func(Format)) {
	if f.Verbose() {
		// Append value with type information
		f._AppendPrimitiveTyped(appendPrimitive)
	} else {
		// Append simply as-is
		appendPrimitive(f)
	}
}

// _AppendPrimitiveTyped is a helper to append prefix/suffix for primitives (numbers/bools/bytes/runes) with their types (if deref'd).
func (f Format) _AppendPrimitiveTyped(appendPrimitive func(Format)) {
	if f.Derefs > 0 {
		// Is deref'd, append type info
		f.Buffer.WriteByte('(')
		f.AppendType()
		f.Buffer.WriteString(`)(`)
		appendPrimitive(f)
		f.Buffer.WriteByte(')')
	} else {
		// Simply append value
		appendPrimitive(f)
	}
}

// _AppendPtrType is a helper to append prefix/suffix for ptr types (with type if necessary).
func (f Format) _AppendPtrType(appendPtr func(Format)) {
	if f.Verbose() {
		// Append value with type information
		f.Buffer.WriteByte('(')
		f.AppendType()
		f.Buffer.WriteString(`)(`)
		appendPtr(f)
		f.Buffer.WriteByte(')')
	} else {
		// Append simply as-is
		appendPtr(f)
	}
}

// _AppendArray is a helper to append prefix/suffix for array-types.
func (f Format) _AppendArray(appendArray func(Format)) {
	f.Buffer.WriteByte('[')
	appendArray(f)
	f.Buffer.WriteByte(']')
}

// _AppendArrayTyped is a helper to append prefix/suffix for array-types with their types.
func (f Format) _AppendArrayTyped(appendArray func(Format)) {
	f.AppendType()
	f.Buffer.WriteByte('{')
	appendArray(f)
	f.Buffer.WriteByte('}')
}

// _AppendFields is a helper to append prefix/suffix for field-types (with type if necessary).
func (f Format) _AppendFieldType(appendFields func(Format)) {
	if f.Verbose() {
		f.AppendType()
	}
	f.Buffer.WriteByte('{')
	appendFields(f)
	f.Buffer.WriteByte('}')
}

// byte2str returns 'c' as a string, escaping if necessary.
func byte2str(c byte) string {
	switch c {
	case '\a':
		return `\a`
	case '\b':
		return `\b`
	case '\f':
		return `\f`
	case '\n':
		return `\n`
	case '\r':
		return `\r`
	case '\t':
		return `\t`
	case '\v':
		return `\v`
	case '\'':
		return `\\`
	default:
		if c < ' ' {
			const hex = "0123456789abcdef"
			return `\x` +
				string(hex[c>>4]) +
				string(hex[c&0xF])
		}
		return string(c)
	}
}
