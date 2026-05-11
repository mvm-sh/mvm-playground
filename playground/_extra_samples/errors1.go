// Demonstrates a remote import: github.com/pkg/errors is fetched through the
// Go module proxy. errors.Wrap / WithMessage add context, errors.Cause walks
// back to the root error.
package main

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

func main() {
	err := errors.Wrap(io.EOF, "reading config")
	err = errors.WithMessage(err, "startup")

	fmt.Println(err)
	fmt.Println(errors.Cause(err))
	fmt.Println(errors.Cause(err) == io.EOF)
}

// Output:
// startup: reading config: EOF
// EOF
// true
