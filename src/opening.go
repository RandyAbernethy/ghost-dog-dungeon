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

var chapelIntroduction = [...]string{
	"After coming upon an old chapel, you find an entrance to a dark cellar in the chapel yard. Your adventurous spirit drives you to explore.",
	"The wind howls through the bell tower, like a far away hound longing for home.",
	"You climb down into the cellar below the old chapel and the door slams shut behind you. Some great evil is afoot and clearly you must face it to escape the catacombs.",
}
