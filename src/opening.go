package main

import (
	_ "embed"
	"fmt"
	"io"
)

// Embed the reusable art so binaries work without an external asset directory.
//
//go:embed assets/puppy.txt
var puppyArt string

func renderOpeningArt(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprint(w, puppyArt)
}
