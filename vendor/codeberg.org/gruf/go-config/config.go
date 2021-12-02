package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/pelletier/go-toml"
)

func getKeyName(path, key string) string {
	if len(path) > 0 {
		key = path + "." + key
	}
	return key
}

// Tree is the config's key to tracked value pointer map. TOML values are parsed into here
type Tree map[string]interface{}

// Parse attempts to parse the configuration file at path, panicking on undefined keys
func (c Tree) Parse(path string) {
	// Try parse file at path
	undefined, err := c.ParseDefined(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config file %s: %v\n", path, err)
		os.Exit(2)
	}

	// Fail on any undefined keys found
	if len(undefined) > 0 {
		s := "Undefined keys in config file '" + path + "':\n"
		for key := range undefined {
			s += "\t" + key + "\n"
		}
		fmt.Fprint(os.Stderr, s)
		os.Exit(1)
	}
}

// ParseDefined attempts to parse the configuration file at Path, returning
// a map of any undefined keys
func (c Tree) ParseDefined(path string) (map[string]interface{}, error) {
	// Read TOML file
	tree, err := toml.LoadFile(path)
	if err != nil {
		return nil, err
	}

	// Recurse the tree!
	undefined := map[string]interface{}{}
	return undefined, c.recurseTree(undefined, "", tree)
}

func typeCheck(check interface{}, target interface{}) bool {
	ct := reflect.TypeOf(check)
	tt := reflect.TypeOf(target)
	return ct == tt
}

func (c Tree) recurseTree(undefined map[string]interface{}, current string, tree *toml.Tree) error {
	for _, key := range tree.Keys() {
		// Get total key name
		totalKey := getKeyName(current, key)

		// Get key value
		val := tree.Get(key)

		// Handle the case that this is a subtree
		if sub, ok := val.(*toml.Tree); ok {

			// Check for wildcard in our value ptrs
			if ptr, have := c[getKeyName(current, "*")]; have {
				wild, ok := ptr.(*wildcard)
				if !ok {
					return fmt.Errorf("unexpected type %T for key %s", ptr, totalKey)
				}

				// Get subtree as map
				m := sub.ToMap()

				// Check key + value types
				// against the expected map
				found := 0
				for mkey, mval := range m {
					// Check wildcard for key in subtree
					check, ok := wild.expect[mkey]
					if ok {
						// Check is expected value type
						if !typeCheck(mval, check) {
							return fmt.Errorf("unexpected type %T for key %s in wildcard with key %s", mval, mkey, totalKey)
						}

						// Iter
						found++
						continue
					}

					// Only error in strict mode
					if wild.strict {
						return fmt.Errorf("undefined key %s in wildcard with key %s", mkey, totalKey)
					}
				}

				// Ensure all keys defined
				if wild.require && found < len(wild.expect) {
					return fmt.Errorf("missing key definitions in wildcard with key %s", totalKey)
				}

				// Store wildcard entry
				wild.values[key] = m
				continue
			}

			// Regular subtree, recurse
			err := c.recurseTree(undefined, totalKey, sub)
			if err != nil {
				return err
			}
			continue
		}

		// Get our own value ptr for key
		ptr, have := c[totalKey]
		if !have {
			undefined[totalKey] = val
			continue
		}

		// Handle type of value pointer
		switch ptr := ptr.(type) {
		// String pointer, try get string out of TOML value
		case *string:
			s, ok := val.(string)
			if !ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			*ptr = s

		// Uint64 pointer, try get uint64 (or failing that, int64) out of TOML value
		case *uint64:
			u, ok := val.(uint64)
			if !ok {
				i, ok := val.(int64)
				if !ok {
					return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
				}
				u = uint64(i)
			}
			*ptr = u

		// Int64 pointer, try get int64 out of TOML value
		case *int64:
			i, ok := val.(int64)
			if !ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			*ptr = i

		// Float64 pointer, try get float64 out of TOML value
		case *float64:
			f, ok := val.(float64)
			if !ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			*ptr = f

		// Bool pointer, try get bool out of TOML value
		case *bool:
			b, ok := val.(bool)
			if !ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			*ptr = b

		// time.Time pointer, try get time.Time out of TOML value
		case *time.Time:
			t, ok := val.(time.Time)
			if !ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			*ptr = t

		// time.Duration pointer, try parse time.Duration out of TOML value
		case *time.Duration:
			// We parse whatever the value is into a string for time.ParseDuration
			d, err := time.ParseDuration(fmt.Sprint(val))
			if err != nil {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			*ptr = d

		// String slice pointer, try get string slice out of TOML value
		case *[]string:
			m, ok := val.([]interface{})
			if !ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			out := make([]string, len(m))
			for i := range m {
				s, ok := m[i].(string)
				if !ok {
					return fmt.Errorf("unexpected type %T in array for key %s", m[i], totalKey)
				}
				out[i] = s
			}
			*ptr = out

		// Uint64 slice pointer, try get uint64 slice out of TOML value
		case *[]uint64:
			m, ok := val.([]interface{})
			if !ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			out := make([]uint64, len(m))
			for i := range m {
				s, ok := m[i].(uint64)
				if !ok {
					u, ok := m[i].(int64)
					if !ok {
						return fmt.Errorf("unexpected type %T in array for key %s", m[i], totalKey)
					}
					s = uint64(u)
				}
				out[i] = s
			}
			*ptr = out

		// Int64 slice pointer, try get in64 slice out of TOML value
		case *[]int64:
			m, ok := val.([]interface{})
			if !ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			out := make([]int64, len(m))
			for i := range m {
				s, ok := m[i].(int64)
				if !ok {
					return fmt.Errorf("unexpected type %T in array for key %s", m[i], totalKey)
				}
				out[i] = s
			}
			*ptr = out

		// Float64 slice pointer, try get float64 slice out of TOML value
		case *[]float64:
			m, ok := val.([]interface{})
			if !ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			out := make([]float64, len(m))
			for i := range m {
				s, ok := m[i].(float64)
				if !ok {
					return fmt.Errorf("unexpected type %T in array for key %s", m[i], totalKey)
				}
				out[i] = s
			}
			*ptr = out

		// Bool slice pointer, try get bool slice out of TOML value
		case *[]bool:
			m, ok := val.([]interface{})
			if !ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			out := make([]bool, len(m))
			for i := range m {
				s, ok := m[i].(bool)
				if !ok {
					return fmt.Errorf("unexpected type %T in array for key %s", m[i], totalKey)
				}
				out[i] = s
			}
			*ptr = out

		// time.Time slice pointer, try get time.Time slice out of TOML value
		case *[]time.Time:
			m, ok := val.([]interface{})
			if ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			out := make([]time.Time, len(m))
			for i := range m {
				t, ok := m[i].(time.Time)
				if !ok {
					return fmt.Errorf("unexpected type %T in array for key %s", m[i], totalKey)
				}
				out[i] = t
			}
			*ptr = out

		// time.Duration slice pointer, try parse time.Duration slice out of TOML value
		case *[]time.Duration:
			m, ok := val.([]interface{})
			if ok {
				return fmt.Errorf("unexpected type %T for key %s", val, totalKey)
			}
			out := make([]time.Duration, len(m))
			for i := range m {
				// We take whatever m[i] is as a string to parse
				d, err := time.ParseDuration(fmt.Sprint(m[i]))
				if err != nil {
					return fmt.Errorf("unexpected type %T in array for key %s", m[i], totalKey)
				}
				out[i] = d
			}
			*ptr = out

		// Default (not a type we deal with) is panic
		default:
			panic("Unexpected type in config.Tree map for key: " + totalKey)
		}
	}
	return nil
}

func (c Tree) put(key string, ptr interface{}) {
	_, already := c[key]
	if already {
		panic("Already tracking config entry with key: " + key)
	}
	c[key] = ptr
}

// wildcard is our own wrapper type for storing
// and parsing wildcard value types
type wildcard struct {
	values  map[string](map[string]interface{})
	expect  map[string]interface{}
	require bool
	strict  bool
}

// Wildcard defines a TOML tree with expected keys to track. Strict mode: throws errors on undefined (extra) keys. Require mode: throws errors on missing keys
func (c Tree) Wildcard(key string, defaultValue map[string]interface{}, require bool, strict bool) *map[string](map[string]interface{}) {
	m := make(map[string](map[string]interface{}))
	ptr := &m
	c.WildcardVar(ptr, key, defaultValue, require, strict)
	return ptr
}

// WildcardVar defines a TOML tree with expected keys to track. Strict mode: throws errors on undefined (extra) keys. Require mode: throws errors on missing keys. Stores value in provided ptr
func (c Tree) WildcardVar(ptr *map[string](map[string]interface{}), key string, defaultValue map[string]interface{}, require bool, strict bool) {
	if !strings.HasSuffix(key, "*") {
		panic("Wildcard keys must end in a '*'")
	}
	c.put(key, &wildcard{
		values:  *ptr,
		expect:  defaultValue,
		require: require,
		strict:  strict,
	})
}

// String defines a TOML key to string value to track
func (c Tree) String(key string, defaultValue string) *string {
	s := new(string)
	c.StringVar(s, key, defaultValue)
	return s
}

// StringVar defines a TOML key to string value to track, storing value in provided pointer
func (c Tree) StringVar(ptr *string, key string, defaultValue string) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// Int64 defines a TOML key to int64 value to track
func (c Tree) Int64(key string, defaultValue int64) *int64 {
	i := new(int64)
	c.Int64Var(i, key, defaultValue)
	return i
}

// Int64Var defines a TOML key to int64 value to track, storing value in provided pointer
func (c Tree) Int64Var(ptr *int64, key string, defaultValue int64) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// Uint64 defines a TOML key to uint64 value to track
func (c Tree) Uint64(key string, defaultValue uint64) *uint64 {
	u := new(uint64)
	c.Uint64Var(u, key, defaultValue)
	return (*uint64)(u)
}

// Uint64Var defines a TOML key to uint64 value to track, storing value in provided pointer
func (c Tree) Uint64Var(ptr *uint64, key string, defaultValue uint64) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// Float64 defines a TOML key to float64 value to track
func (c Tree) Float64(key string, defaultValue float64) *float64 {
	f := new(float64)
	c.Float64Var(f, key, defaultValue)
	return f
}

// Float64Var defines a TOML key to float64 value to track, storing value in provided pointer
func (c Tree) Float64Var(ptr *float64, key string, defaultValue float64) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// Bool defines a TOML key to bool value to track
func (c Tree) Bool(key string, defaultValue bool) *bool {
	b := new(bool)
	c.BoolVar(b, key, defaultValue)
	return b
}

// BoolVar defines a TOML key to bool value to track, storing value in provided pointer
func (c Tree) BoolVar(ptr *bool, key string, defaultValue bool) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// Time defines a TOML key to time.Time value to track
func (c Tree) Time(key string, defaultValue time.Time) *time.Time {
	t := new(time.Time)
	c.TimeVar(t, key, defaultValue)
	return t
}

// TimeVar defines a TOML key to time.Time value to track, storing value in provided pointer
func (c Tree) TimeVar(ptr *time.Time, key string, defaultValue time.Time) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// Duration defines a TOML key to time.Duration value to track
func (c Tree) Duration(key string, defaultValue time.Duration) *time.Duration {
	d := new(time.Duration)
	c.DurationVar(d, key, defaultValue)
	return d
}

// DurationVar defines a TOML key to time.Duration value to track, storing value in provided pointer
func (c Tree) DurationVar(ptr *time.Duration, key string, defaultValue time.Duration) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// StringArray defines a TOML key to string array value to track
func (c Tree) StringArray(key string, defaultValue []string) *[]string {
	s := new([]string)
	c.StringArrayVar(s, key, defaultValue)
	return s
}

// StringArrayVar defines a TOML key to string array value to track, storing value in provided pointer
func (c Tree) StringArrayVar(ptr *[]string, key string, defaultValue []string) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// Int64Array defines a TOML key to int64 array value to track
func (c Tree) Int64Array(key string, defaultValue []int64) *[]int64 {
	i := new([]int64)
	c.Int64ArrayVar(i, key, defaultValue)
	return i
}

// Int64ArrayVar defines a TOML key to int64 array value to track, storing value in provided pointer
func (c Tree) Int64ArrayVar(ptr *[]int64, key string, defaultValue []int64) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// Uint64Array defines a TOML key to uint64 array value to track
func (c Tree) Uint64Array(key string, defaultValue []uint64) *[]uint64 {
	u := new([]uint64)
	c.Uint64ArrayVar(u, key, defaultValue)
	return u
}

// Uint64ArrayVar defines a TOML key to uint64 array value to track, storing value in provided pointer
func (c Tree) Uint64ArrayVar(ptr *[]uint64, key string, defaultValue []uint64) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// Float64Array defines a TOML key to float64 array value to track
func (c Tree) Float64Array(key string, defaultValue []float64) *[]float64 {
	f := new([]float64)
	c.Float64ArrayVar(f, key, defaultValue)
	return f
}

// Float64ArrayVar defines a TOML key to float64 array value to track, storing value in provided pointer
func (c Tree) Float64ArrayVar(ptr *[]float64, key string, defaultValue []float64) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// BoolArray defines a TOML key to bool array value to track
func (c Tree) BoolArray(key string, defaultValue []bool) *[]bool {
	b := new([]bool)
	c.BoolArrayVar(b, key, defaultValue)
	return b
}

// BoolArrayVar defines a TOML key to bool array value to track, storing value in provided pointer
func (c Tree) BoolArrayVar(ptr *[]bool, key string, defaultValue []bool) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// TimeArray defines a TOML key to time.Time array value to track
func (c Tree) TimeArray(key string, defaultValue []time.Time) *[]time.Time {
	t := new([]time.Time)
	c.TimeArrayVar(t, key, defaultValue)
	return t
}

// TimeArrayVar defines a TOML key to time.Time array value to track, storing value in provided pointer
func (c Tree) TimeArrayVar(ptr *[]time.Time, key string, defaultValue []time.Time) {
	*ptr = defaultValue
	c.put(key, ptr)
}

// DurationArray defines a TOML key to time.Duration array value to track
func (c Tree) DurationArray(key string, defaultValue []time.Duration) *[]time.Duration {
	d := new([]time.Duration)
	c.DurationArrayVar(d, key, defaultValue)
	return d
}

// DurationArrayVar defines a TOML key to time.Duration array value to track, storing value in provided pointer
func (c Tree) DurationArrayVar(ptr *[]time.Duration, key string, defaultValue []time.Duration) {
	*ptr = defaultValue
	c.put(key, ptr)
}

var (
	// globalTree is the global config.Tree instance
	globalTree Tree

	// once is used to init globalTree
	once sync.Once
)

// get fetches the globalTree, creating if nil
func get() Tree {
	once.Do(func() {
		globalTree = make(Tree)
	})
	return globalTree
}

// Parse attempts to parse the configuration file at path
func Parse(path string) {
	get().Parse(path)
}

// ParseDefined attempts to parse the configuration file at Path, returning a map of any undefined keys
func ParseDefined(path string) (map[string]interface{}, error) {
	return get().ParseDefined(path)
}

// String defines a TOML key to string value to track
func String(key string, defaultValue string) *string {
	return get().String(key, defaultValue)
}

// StringVar defines a TOML key to string value to track, storing value in provided pointer
func StringVar(ptr *string, key string, defaultValue string) {
	get().StringVar(ptr, key, defaultValue)
}

// Int64 defines a TOML key to int64 value to track
func Int64(key string, defaultValue int64) *int64 {
	return get().Int64(key, defaultValue)
}

// Int64Var defines a TOML key to int64 value to track, storing value in provided pointer
func Int64Var(ptr *int64, key string, defaultValue int64) {
	get().Int64Var(ptr, key, defaultValue)
}

// Uint64 defines a TOML key to uint64 value to track
func Uint64(key string, defaultValue uint64) *uint64 {
	return get().Uint64(key, defaultValue)
}

// Uint64Var defines a TOML key to uint64 value to track, storing value in provided pointer
func Uint64Var(ptr *uint64, key string, defaultValue uint64) {
	get().Uint64Var(ptr, key, defaultValue)
}

// Float64 defines a TOML key to float64 value to track
func Float64(key string, defaultValue float64) *float64 {
	return get().Float64(key, defaultValue)
}

// Float64Var defines a TOML key to float64 value to track, storing value in provided pointer
func Float64Var(ptr *float64, key string, defaultValue float64) {
	get().Float64Var(ptr, key, defaultValue)
}

// Bool defines a TOML key to bool value to track
func Bool(key string, defaultValue bool) *bool {
	return get().Bool(key, defaultValue)
}

// BoolVar defines a TOML key to bool value to track, storing value in provided pointer
func BoolVar(ptr *bool, key string, defaultValue bool) {
	get().BoolVar(ptr, key, defaultValue)
}

// Time defines a TOML key to time.Time value to track
func Time(key string, defaultValue time.Time) *time.Time {
	return get().Time(key, defaultValue)
}

// TimeVar defines a TOML key to time.Time value to track, storing value in provided pointer
func TimeVar(ptr *time.Time, key string, defaultValue time.Time) {
	get().TimeVar(ptr, key, defaultValue)
}

// Duration defines a TOML key to time.Duration value to track
func Duration(key string, defaultValue time.Duration) *time.Duration {
	return get().Duration(key, defaultValue)
}

// DurationVar defines a TOML key to time.Duration value to track, storing value in provided pointer
func DurationVar(ptr *time.Duration, key string, defaultValue time.Duration) {
	get().DurationVar(ptr, key, defaultValue)
}

// StringArray defines a TOML key to string array value to track
func StringArray(key string, defaultValue []string) *[]string {
	return get().StringArray(key, defaultValue)
}

// StringArrayVar defines a TOML key to string array value to track, storing value in provided pointer
func StringArrayVar(ptr *[]string, key string, defaultValue []string) {
	get().StringArrayVar(ptr, key, defaultValue)
}

// Int64Array defines a TOML key to int64 array value to track
func Int64Array(key string, defaultValue []int64) *[]int64 {
	return get().Int64Array(key, defaultValue)
}

// Int64ArrayVar defines a TOML key to int64 array value to track, storing value in provided pointer
func Int64ArrayVar(ptr *[]int64, key string, defaultValue []int64) {
	get().Int64ArrayVar(ptr, key, defaultValue)
}

// Uint64Array defines a TOML key to uint64 array value to track
func Uint64Array(key string, defaultValue []uint64) *[]uint64 {
	return get().Uint64Array(key, defaultValue)
}

// Uint64ArrayVar defines a TOML key to uint64 array value to track, storing value in provided pointer
func Uint64ArrayVar(ptr *[]uint64, key string, defaultValue []uint64) {
	get().Uint64ArrayVar(ptr, key, defaultValue)
}

// Float64Array defines a TOML key to float64 array value to track
func Float64Array(key string, defaultValue []float64) *[]float64 {
	return get().Float64Array(key, defaultValue)
}

// Float64ArrayVar defines a TOML key to float64 array value to track, storing value in provided pointer
func Float64ArrayVar(ptr *[]float64, key string, defaultValue []float64) {
	get().Float64ArrayVar(ptr, key, defaultValue)
}

// BoolArray defines a TOML key to bool array value to track
func BoolArray(key string, defaultValue []bool) *[]bool {
	return get().BoolArray(key, defaultValue)
}

// BoolArrayVar defines a TOML key to bool array value to track, storing value in provided pointer
func BoolArrayVar(ptr *[]bool, key string, defaultValue []bool) {
	get().BoolArrayVar(ptr, key, defaultValue)
}

// TimeArray defines a TOML key to time.Time array value to track
func TimeArray(key string, defaultValue []time.Time) *[]time.Time {
	return get().TimeArray(key, defaultValue)
}

// TimeArrayVar defines a TOML key to time.Time array value to track, storing value in provided pointer
func TimeArrayVar(ptr *[]time.Time, key string, defaultValue []time.Time) {
	get().TimeArrayVar(ptr, key, defaultValue)
}

// DurationArray defines a TOML key to time.Duration array value to track
func DurationArray(key string, defaultValue []time.Duration) *[]time.Duration {
	return get().DurationArray(key, defaultValue)
}

// DurationArrayVar defines a TOML key to time.Duration array value to track, storing value in provided pointer
func DurationArrayVar(ptr *[]time.Duration, key string, defaultValue []time.Duration) {
	get().DurationArrayVar(ptr, key, defaultValue)
}
