package playground

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"

	"github.com/mvm-sh/mvm/interp"
	"github.com/mvm-sh/mvm/scan"
)

// Repl is a persistent interactive session: successive Eval calls share one
// interpreter, so definitions made on one line are visible on the next. It is
// the line-at-a-time core of interp.Repl, without owning the prompt loop, and
// it recovers interpreter panics so one bad line doesn't kill the session.
type Repl struct {
	i    *interp.Interp
	out  *bytes.Buffer
	errb *bytes.Buffer
	acc  string // accumulated source of an unterminated block
}

// NewRepl returns a fresh REPL session.
func NewRepl() *Repl {
	r := &Repl{out: &bytes.Buffer{}, errb: &bytes.Buffer{}}
	r.i = newInterp(r.out, r.errb)
	return r
}

// SetTrace enables or disables line / bytecode tracing for subsequent Eval
// calls. Trace output is written to the session's stderr stream.
func (r *Repl) SetTrace(line, op bool) {
	r.i.SetTracing(line)
	r.i.SetTraceOps(op)
}

// Interp exposes the underlying interpreter so callers can introspect
// loaded sources, compiled bytecode, or debug info. The interpreter is
// shared with future Eval calls; callers should treat it as read-only.
func (r *Repl) Interp() *interp.Interp { return r.i }

// Eval evaluates one input line. It returns whatever the line wrote to stdout
// and stderr, the printed form of the line's value (empty if none), and more,
// which reports that the input is an unterminated block and the next call
// should supply its continuation.
func (r *Repl) Eval(line string) (stdout, stderr, result string, more bool) {
	r.out.Reset()
	r.errb.Reset()
	r.acc += line + "\n"

	res, err := r.evalGuarded(r.acc)
	switch {
	case err == nil:
		if res.IsValid() {
			result = fmt.Sprint(res)
		}
		r.acc = ""
	case errors.Is(err, scan.ErrBlock):
		more = true // keep r.acc; wait for the rest of the block
	default:
		fmt.Fprintln(r.errb, err)
		r.acc = ""
	}
	return r.out.String(), r.errb.String(), result, more
}

func (r *Repl) evalGuarded(srcText string) (res reflect.Value, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			res, err = reflect.Value{}, fmt.Errorf("panic: %v", rec)
		}
	}()
	return r.i.Eval("repl", srcText)
}
