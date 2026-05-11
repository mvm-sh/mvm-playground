// Demonstrates a remote import: github.com/google/uuid is fetched through
// the Go module proxy and interpreted alongside this program.
package main

import (
	"fmt"

	"github.com/google/uuid"
)

func main() {
	// A freshly generated UUID is always random (version 4).
	fmt.Println("New:", uuid.New().Version(), uuid.New().Variant())

	// Parsing keeps the encoded version and variant.
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fmt.Println("Parsed:", id, id.Version(), id.Variant())

	// Name-based (version 5, SHA-1) UUIDs are deterministic.
	v5 := uuid.NewSHA1(uuid.NameSpaceURL, []byte("https://github.com/mvm-sh/mvm"))
	fmt.Println("SHA1:", v5, v5.Version())
}

// Output:
// New: VERSION_4 RFC4122
// Parsed: 123e4567-e89b-12d3-a456-426614174000 VERSION_1 RFC4122
// SHA1: 992541d9-ef81-52a1-b4ef-404a5fdf26d3 VERSION_5
