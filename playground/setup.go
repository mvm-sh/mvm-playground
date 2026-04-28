// Package playground wires the mvm interpreter for the playground.
package playground

import (
	"io"
	"strings"

	"github.com/mvm-sh/mvm/interp"
	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/core"
	_ "github.com/mvm-sh/mvm/stdlib/jsonx"
)

// NewInterpreter returns an mvm interpreter pre-configured for playground.
func NewInterpreter(stdout, stderr io.Writer) *interp.Interp {
	i := interp.NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.SetIO(strings.NewReader(""), stdout, stderr)
	i.AutoImportPackages()
	return i
}
