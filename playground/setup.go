// Package playground wires the mvm interpreter for the browser playground:
// running a source snippet (optionally traced), running a remote package's
// test suite, and an interactive REPL session.
package playground

import (
	"io"
	"runtime"
	"strings"

	"github.com/mvm-sh/mvm/interp"
	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/modfs"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all" // register every stdlib binding
	"github.com/mvm-sh/mvm/stdlib/stdmod"
)

// wireFS plugs an in-memory module filesystem into the interpreter, mirroring
// the mvm CLI's wiring: the embedded std snapshot satisfies stdlib-shaped
// imports from source and the bundled third-party zips satisfy the showcased
// remote imports. In the browser (GOOS=js) the proxy is disabled -- live
// fetches there fail on CORS and would only stall -- so only injected modules
// resolve; off-browser the public proxy is left configured as a fallback.
func wireFS(i *interp.Interp) {
	mfs := modfs.New(modfs.Options{
		Proxy:   modfs.DefaultProxy,
		Offline: runtime.GOOS == "js",
	})
	if err := mfs.Inject(stdmod.ModulePath, stdmod.Version, stdlib.EmbeddedStd()); err != nil {
		panic("modfs inject embedded std: " + err.Error())
	}
	for _, m := range bundledModules() {
		if err := mfs.Inject(m.Path, m.Version, m.Zip); err != nil {
			panic("modfs inject " + m.Path + ": " + err.Error())
		}
	}
	i.SetStdlibFS(stdmod.FS(mfs))
	i.SetRemoteFS(mfs)
}

// newInterp builds a fully wired interpreter writing to the given streams,
// mirroring the mvm CLI's setup order.
func newInterp(stdout, stderr io.Writer) *interp.Interp {
	i := interp.NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	wireFS(i)
	i.SetIO(strings.NewReader(""), stdout, stderr)
	i.AutoImportPackages()
	return i
}

// NewInterpreter returns an mvm interpreter pre-configured for the playground.
// Callers that want execution tracing should call SetTracing/SetTraceOps on
// the result before Eval.
func NewInterpreter(stdout, stderr io.Writer) *interp.Interp {
	return newInterp(stdout, stderr)
}
