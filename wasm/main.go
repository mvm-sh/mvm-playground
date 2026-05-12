//go:build js && wasm

// The wasm command exposes the mvm interpreter to the browser. It registers
// these JS globals and then parks:
//
//	mvmRun(source [, {traceLine, traceOp, name}]) -> {stdout, stderr, error, name, sources: [{name, lines, bytes}, ...]}
//	mvmListSamples()                        -> [name, ...]
//	mvmGetSample(name)                      -> source string
//	mvmLastSource(name)                     -> content of the named source from the last run
//	mvmVersion()                            -> "<mvm-ver> <go-ver> <os>/<arch>"
//	mvmReplReset()                          -> (clears the REPL session)
//	mvmReplEval(line [, {traceLine, traceOp}]) -> {stdout, stderr, result, more, error}
package main

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/debug"
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

func optString(args []js.Value, idx int, key, def string) string {
	if len(args) <= idx {
		return def
	}
	o := args[idx]
	if o.Type() != js.TypeObject {
		return def
	}
	v := o.Get(key)
	if v.Type() != js.TypeString {
		return def
	}
	if s := v.String(); s != "" {
		return s
	}
	return def
}

// lastSources holds the content of every source loaded by the most recent
// runMVM call, keyed by source name. It backs mvmLastSource so the browser
// can show imported package files in a tab without keeping the whole
// interpreter alive between runs.
var lastSources map[string]string

func runMVM(_ js.Value, args []js.Value) (ret any) {
	var stdout, stderr bytes.Buffer
	var errMsg string
	var sources []any
	name := optString(args, 1, "name", "main.go")
	defer func() {
		if r := recover(); r != nil {
			errMsg = fmt.Sprintf("interpreter panic: %v", r)
		}
		ret = js.ValueOf(map[string]any{
			"stdout":  stdout.String(),
			"stderr":  stderr.String(),
			"error":   errMsg,
			"name":    name,
			"sources": sources,
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
	if _, err := i.Eval(name, args[0].String()); err != nil {
		errMsg = err.Error()
	}

	lastSources = make(map[string]string, len(i.Sources)+1)
	sources = make([]any, 0, len(i.Sources)+1)
	for k := range i.Sources {
		s := &i.Sources[k]
		lastSources[s.Name] = s.Content()
		sources = append(sources, map[string]any{
			"name":  s.Name,
			"lines": s.Lines(),
			"bytes": s.Len,
		})
	}

	// Add a synthetic listing entry; the playground UI surfaces it as a tab
	// like any other source, but it's a disassembly of the compiled bytecode
	// interleaved with the Go lines that produced it.
	var asm bytes.Buffer
	playground.FormatListing(&asm, i)
	if asm.Len() > 0 {
		const asmName = "<bytecode>"
		lastSources[asmName] = asm.String()
		lines := 0
		for _, b := range asm.Bytes() {
			if b == '\n' {
				lines++
			}
		}
		sources = append(sources, map[string]any{
			"name":  asmName,
			"lines": lines,
			"bytes": asm.Len(),
		})
	}
	return
}

func lastSource(_ js.Value, args []js.Value) any {
	if len(args) < 1 || lastSources == nil {
		return js.ValueOf("")
	}
	return js.ValueOf(lastSources[args[0].String()])
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
	var sources []any
	defer func() {
		if r := recover(); r != nil {
			errMsg = fmt.Sprintf("interpreter panic: %v", r)
		}
		ret = js.ValueOf(map[string]any{
			"stdout":  stdout,
			"stderr":  stderr,
			"result":  resStr,
			"more":    more,
			"error":   errMsg,
			"sources": sources,
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

	i := repl.Interp()
	lastSources = make(map[string]string, len(i.Sources)+1)
	sources = make([]any, 0, len(i.Sources)+1)
	for k := range i.Sources {
		s := &i.Sources[k]
		lastSources[s.Name] = s.Content()
		sources = append(sources, map[string]any{
			"name":  s.Name,
			"lines": s.Lines(),
			"bytes": s.Len,
		})
	}
	var asm bytes.Buffer
	playground.FormatListing(&asm, i)
	if asm.Len() > 0 {
		const asmName = "<bytecode>"
		lastSources[asmName] = asm.String()
		lines := 0
		for _, b := range asm.Bytes() {
			if b == '\n' {
				lines++
			}
		}
		sources = append(sources, map[string]any{
			"name":  asmName,
			"lines": lines,
			"bytes": asm.Len(),
		})
	}
	return
}

// mvmVersionString returns "<mvm-version> <go-version> js/wasm", mirroring
// the format of `mvm version`. The mvm version is taken from the dependency
// entry in this binary's build info; a local `replace` directive surfaces as
// "(devel)".
func mvmVersionString() string {
	mvmVer, goVer := "(unknown)", runtime.Version()
	if bi, ok := debug.ReadBuildInfo(); ok {
		goVer = bi.GoVersion
		for _, d := range bi.Deps {
			if d.Path == "github.com/mvm-sh/mvm" {
				mvmVer = d.Version
				if d.Replace != nil && d.Replace.Version != "" {
					mvmVer = d.Replace.Version
				}
				break
			}
		}
	}
	return fmt.Sprintf("%.12s %s %s/%s", mvmVer, goVer, runtime.GOOS, runtime.GOARCH)
}

func mvmVersion(js.Value, []js.Value) any { return js.ValueOf(mvmVersionString()) }

func main() {
	js.Global().Set("mvmRun", js.FuncOf(runMVM))
	js.Global().Set("mvmListSamples", js.FuncOf(listSamples))
	js.Global().Set("mvmGetSample", js.FuncOf(getSample))
	js.Global().Set("mvmLastSource", js.FuncOf(lastSource))
	js.Global().Set("mvmReplReset", js.FuncOf(replReset))
	js.Global().Set("mvmReplEval", js.FuncOf(replEval))
	js.Global().Set("mvmVersion", js.FuncOf(mvmVersion))
	select {}
}
