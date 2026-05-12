package playground

import (
	"fmt"
	"io"

	"github.com/mvm-sh/mvm/interp"
)

// FormatListing writes a disassembly of i's compiled bytecode to w. Each
// instruction is printed as "<IP>: <OP> [A [B]]". Where source-position
// information is available it is shown as a comment line just before the
// run of instructions it produced, interleaving Go source with bytecode:
//
//	; fib.go:3  func fib(i int) int {
//	  fib:
//	      0: GetLocal 1
//	      1: Push 2
//	      ...
//
// Function labels (from DebugInfo.Labels) are printed when a new label
// starts at the current IP, mirroring an assembly listing.
func FormatListing(w io.Writer, i *interp.Interp) {
	di := i.BuildDebugInfo()
	if di == nil {
		return
	}
	var lastFile string
	var lastLine int
	for ip, ins := range i.Code {
		if lbl := di.Labels[ip]; lbl != "" {
			fmt.Fprintf(w, "\n%s:\n", lbl)
		}
		if ins.Pos != 0 && len(di.Sources) > 0 {
			file, line, _ := di.Sources.Resolve(int(ins.Pos))
			if file != "" && (file != lastFile || line != lastLine) {
				text := di.Sources.LineText(int(ins.Pos))
				fmt.Fprintf(w, "    ; %s:%d  %s\n", file, line, text)
				lastFile, lastLine = file, line
			}
		}
		fmt.Fprintf(w, "    %5d: %s", ip, ins.Op)
		if ins.A != 0 || ins.B != 0 {
			fmt.Fprintf(w, " %d", ins.A)
		}
		if ins.B != 0 {
			fmt.Fprintf(w, " %d", ins.B)
		}
		fmt.Fprintln(w)
	}
}
