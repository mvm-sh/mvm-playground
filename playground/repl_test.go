package playground

import (
	"strings"
	"testing"
)

func TestRepl(t *testing.T) {
	r := NewRepl()

	// (line, want continuation?, substring expected in stdout, value expected)
	type step struct {
		line     string
		more     bool
		out      string
		result   string // checked only when non-empty
		checkRes bool   // assert result == "" (no value produced)
	}
	steps := []step{
		{line: "x := 5"},
		{line: "x * 8", result: "40"},
		{line: `len("abcd")`, result: "4"},
		{line: "y := 0", checkRes: false}, // x:= echoes its RHS in mvm; don't pin it
		{line: "for i := 0; i < 3; i++ {", more: true},
		{line: "y += i }", more: false},
		{line: "y", result: "3"},
		{line: "x + y", result: "8"},
	}
	for n, s := range steps {
		stdout, stderr, result, more := r.Eval(s.line)
		if more != s.more {
			t.Errorf("step %d (%q): more=%v, want %v", n, s.line, more, s.more)
		}
		if s.result != "" && result != s.result {
			t.Errorf("step %d (%q): result=%q, want %q", n, s.line, result, s.result)
		}
		if s.checkRes && result != "" {
			t.Errorf("step %d (%q): result=%q, want empty", n, s.line, result)
		}
		if s.out != "" && !strings.Contains(stdout, s.out) {
			t.Errorf("step %d (%q): stdout=%q, want contains %q", n, s.line, stdout, s.out)
		}
		if stderr != "" {
			t.Errorf("step %d (%q): unexpected stderr %q", n, s.line, stderr)
		}
	}

	// A panicking line must not kill the session.
	if _, stderr, _, _ := r.Eval("1 / 0"); stderr == "" {
		t.Errorf("expected stderr for 1/0")
	}
	if _, _, result, _ := r.Eval("x"); result != "5" {
		t.Errorf("session lost after panic: x=%q, want 5", result)
	}
}
