//go:build js && wasm

// The wasm command exposes the mvm interpreter to the browser. It registers
// these JS globals and then parks:
//
//	mvmRun(source [, {traceLine, traceOp}]) -> {stdout, stderr, error}
//	mvmListSamples()                        -> [name, ...]
//	mvmGetSample(name)                      -> source string
//	mvmReplReset()                          -> (clears the REPL session)
//	mvmReplEval(line [, {traceLine, traceOp}]) -> {stdout, stderr, result, more, error}
package main

import (
	"bytes"
	"fmt"
	"syscall/js"

	"github.com/mvm-sh/mvm-playground/playground"
)

func optBool(args []js.Value, idx int, key string) bool {
	if len(args) <= idx {
		return false
	}
	o := args[idx]
	if o.Type() != js.TypeObject {
		return false
	}
	v := o.Get(key)
	return v.Type() == js.TypeBoolean && v.Bool()
}

func runMVM(_ js.Value, args []js.Value) (ret any) {
	var stdout, stderr bytes.Buffer
	var errMsg string
	defer func() {
		if r := recover(); r != nil {
			errMsg = fmt.Sprintf("interpreter panic: %v", r)
		}
		ret = js.ValueOf(map[string]any{
			"stdout": stdout.String(),
			"stderr": stderr.String(),
			"error":  errMsg,
		})
	}()

	if len(args) < 1 {
		errMsg = "mvmRun: missing source argument"
		return
	}
	i := playground.NewInterpreter(&stdout, &stderr)
	if optBool(args, 1, "traceLine") {
		i.SetTracing(true)
	}
	if optBool(args, 1, "traceOp") {
		i.SetTraceOps(true)
	}
	if _, err := i.Eval("m:playground", args[0].String()); err != nil {
		errMsg = err.Error()
	}
	return
}

func listSamples(js.Value, []js.Value) any {
	names := playground.Samples()
	out := make([]any, len(names))
	for i, n := range names {
		out[i] = n
	}
	return js.ValueOf(out)
}

func getSample(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.ValueOf("")
	}
	return js.ValueOf(playground.Sample(args[0].String()))
}

var repl *playground.Repl

func replReset(js.Value, []js.Value) any {
	repl = playground.NewRepl()
	return nil
}

func replEval(_ js.Value, args []js.Value) (ret any) {
	var stdout, stderr, resStr string
	var more bool
	var errMsg string
	defer func() {
		if r := recover(); r != nil {
			errMsg = fmt.Sprintf("interpreter panic: %v", r)
		}
		ret = js.ValueOf(map[string]any{
			"stdout": stdout,
			"stderr": stderr,
			"result": resStr,
			"more":   more,
			"error":  errMsg,
		})
	}()

	if len(args) < 1 {
		errMsg = "mvmReplEval: missing line argument"
		return
	}
	if repl == nil {
		repl = playground.NewRepl()
	}
	repl.SetTrace(optBool(args, 1, "traceLine"), optBool(args, 1, "traceOp"))
	stdout, stderr, resStr, more = repl.Eval(args[0].String())
	return
}

func main() {
	js.Global().Set("mvmRun", js.FuncOf(runMVM))
	js.Global().Set("mvmListSamples", js.FuncOf(listSamples))
	js.Global().Set("mvmGetSample", js.FuncOf(getSample))
	js.Global().Set("mvmReplReset", js.FuncOf(replReset))
	js.Global().Set("mvmReplEval", js.FuncOf(replEval))
	select {}
}
