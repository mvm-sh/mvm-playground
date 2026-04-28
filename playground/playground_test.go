package playground

import (
	"bytes"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestSamples runs each embedded sample through the playground interpreter and
// compares its output (or error) against the trailing comment block:
//
//	// Output:
//	// expected stdout
//
// or
//
//	// Error:
//	// expected error substring
//
// or
//
//	// skip: reason
//
// The comment-block convention mirrors mvm/interp/file_test.go.
func TestSamples(t *testing.T) {
	for _, name := range Samples() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			src := Sample(name)
			want, isErr, skip := commentData(name, []byte(src))
			if skip {
				t.Skip()
			}

			var stdout, stderr bytes.Buffer
			i := NewInterpreter(&stdout, &stderr)
			_, err := i.Eval("m:"+name, src)

			if isErr {
				if err == nil {
					t.Fatalf("got nil error, want: %q", want)
				}
				if got := strings.TrimSpace(err.Error()); !strings.Contains(got, want) {
					t.Errorf("got error %q, want contains %q", got, want)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
			}
			if got := stdout.String(); got != want {
				t.Errorf("stdout mismatch\ngot:  %q\nwant: %q", got, want)
			}
		})
	}
}

func commentData(name string, src []byte) (text string, isErr, skip bool) {
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, name, src, parser.ParseComments)
	if f == nil || len(f.Comments) == 0 {
		return
	}
	last := f.Comments[len(f.Comments)-1].Text()
	switch {
	case strings.HasPrefix(last, "skip:"):
		return "", false, true
	case strings.HasPrefix(last, "Error:\n"):
		return strings.TrimPrefix(last, "Error:\n"), true, false
	case strings.HasPrefix(last, "Output:\n"):
		return strings.TrimPrefix(last, "Output:\n"), false, false
	}
	return
}
