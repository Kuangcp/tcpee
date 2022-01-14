package logger

import (
	"context"

	"codeberg.org/gruf/go-format"
)

// KV represents the key-value.
type KV struct {
	K string
	V interface{}
}

// Entry defines an entry in the log, it is NOT safe for concurrent use.
type Entry struct {
	ctx context.Context
	lvl LEVEL
	buf *format.Buffer
	log *Logger
}

// Context returns the current set Entry context.Context.
func (e *Entry) Context() context.Context {
	return e.ctx
}

// WithContext updates Entry context value to the supplied.
func (e *Entry) WithContext(ctx context.Context) *Entry {
	e.ctx = ctx
	return e
}

// Level appends the supplied level to the log entry, and sets the entry level.
// Please note this CAN be called and append log levels multiple times.
func (e *Entry) Level(lvl LEVEL) *Entry {
	e.buf.AppendString(`[` + e.log.Levels.Get(lvl) + `] `)
	e.lvl = lvl
	return e
}

// Timestamp appends the current timestamp to the log entry. Please note this
// CAN be called and append the timestamp multiple times.
func (e *Entry) Timestamp() *Entry {
	e.buf.AppendString(clock.NowFormat() + ` `)
	return e
}

// TimestampIf performs Entry.Timestamp() only IF timestamping is enabled for the Logger.
// Please note this CAN be called multiple times.
func (e *Entry) TimestampIf() *Entry {
	if e.log.Timestamp {
		e.Timestamp()
	}
	return e
}

// Hooks applies currently set Hooks to the Entry. Please note this CAN be
// called and perform the Hooks multiple times.
func (e *Entry) Hooks() *Entry {
	for _, hook := range e.log.Hooks {
		hook.Do(e)
	}
	return e
}

// Fields appends a map of key-value pairs to the log entry, these are formatted
// using the `go-format` library and the key / value format directives.
func (e *Entry) Fields(kv ...KV) *Entry {
	for i := range kv {
		Format().Appendf(e.buf, `{:k}={:v} `, kv[i].K, kv[i].V)
	}
	return e
}

// Append will append the given args formatted using fmt.Sprint(a...) to the Entry.
func (e *Entry) Append(a ...interface{}) *Entry {
	e.log.Format.Append(e.buf, a...)
	e.buf.AppendByte(' ')
	return e
}

// Appendf will append the given format string and args using fmt.Sprintf(s, a...) to the Entry.
func (e *Entry) Appendf(s string, a ...interface{}) *Entry {
	e.log.Format.Appendf(e.buf, s, a...)
	e.buf.AppendByte(' ')
	return e
}

// Msg appends the fmt.Sprint() formatted final message to the log and calls .Send()
func (e *Entry) Msg(a ...interface{}) {
	e.log.Format.Append(e.buf, a...)
	e.Send()
}

// Msgf appends the fmt.Sprintf() formatted final message to the log and calls .Send()
func (e *Entry) Msgf(s string, a ...interface{}) {
	e.log.Format.Appendf(e.buf, s, a...)
	e.Send()
}

// Send triggers write of the log entry, skipping if the entry's log LEVEL
// is below the currently set Logger level, and releases the Entry back to
// the Logger's Entry pool. So it is NOT safe to continue using this Entry
// object after calling .Send(), .Msg() or .Msgf()
func (e *Entry) Send() {
	// If nothing to do, return
	if e.lvl < e.log.Level || e.buf.Len() < 1 {
		e.reset()
		return
	}

	// Trim the final space from buf left by our funcs.
	if e.buf.Len() > 1 && e.buf.B[e.buf.Len()-1] == ' ' {
		e.buf.Truncate(1)
	}

	// Ensure a final new line
	if e.buf.B[e.buf.Len()-1] != '\n' {
		e.buf.AppendByte('\n')
	}

	// Write, reset and release
	e.log.Output.Write(e.buf.B)
	e.reset()
}

// reset will empty the Entry and release to pool.
func (e *Entry) reset() {
	// Reset all
	e.ctx = nil
	e.buf.Reset()
	e.lvl = unset

	// Release to pool
	e.log.pool.Put(e)
}
