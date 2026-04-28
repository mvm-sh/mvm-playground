package playground

import (
	"embed"
	"sort"
	"strings"
)

//go:embed _samples/*.go
var samplesFS embed.FS

// Samples returns the names of the embedded sample files (filename only),
// sorted alphabetically.
func Samples() []string {
	entries, err := samplesFS.ReadDir("_samples")
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

// Sample returns the source of the named sample. Returns an empty string for
// missing names or names that contain path separators.
func Sample(name string) string {
	if name == "" || strings.ContainsAny(name, "/\\") {
		return ""
	}
	data, err := samplesFS.ReadFile("_samples/" + name)
	if err != nil {
		return ""
	}
	return string(data)
}
