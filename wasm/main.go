//go:build js && wasm

// The wasm command exposes the mvm interpreter to the browser.
//
// It registers three globals:
//
//	mvmRun(source)       -> {stdout, stderr, error}
//	mvmListSamples()     -> [name, ...]
//	mvmGetSample(name)   -> source string
package main

import (
	"bytes"
	"fmt"
	"syscall/js"

	"github.com/mvm-sh/mvm-playground/playground"
)

func runMVM(this js.Value, args []js.Value) (ret any) {
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

func getSample(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.ValueOf("")
	}
	return js.ValueOf(playground.Sample(args[0].String()))
}

func main() {
	js.Global().Set("mvmRun", js.FuncOf(runMVM))
	js.Global().Set("mvmListSamples", js.FuncOf(listSamples))
	js.Global().Set("mvmGetSample", js.FuncOf(getSample))
	select {}
}
