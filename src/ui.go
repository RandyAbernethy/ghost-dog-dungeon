package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// A conservative fallback also fits command-and-Enter terminals and redirected
// output. COLUMNS permits a narrower layout without changing gameplay state.
func uiWidth() int {
	if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && n > 0 {
		return max(mapWidth, min(80, n))
	}
	return max(mapWidth, min(80, consoleColumns()))
}

func uiText(w io.Writer, text string) {
	wrappedText(w, text, uiWidth(), "", "")
}

// SGR colors occupy no terminal columns and must not affect word wrapping.
var textColorEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func textColumns(text string) int {
	return utf8.RuneCountInString(textColorEscape.ReplaceAllString(text, ""))
}

func wrappedText(w io.Writer, text string, width int, prefix, continuation string) {
	line, columns := "", 0
	for _, word := range strings.Fields(text) {
		wordColumns := textColumns(word)
		if line != "" && textColumns(prefix)+columns+1+wordColumns > width {
			fmt.Fprintln(w, prefix+line)
			prefix = continuation
			line, columns = "", 0
		}
		if line != "" {
			line += " "
			columns++
		}
		line += word
		columns += wordColumns
	}
	fmt.Fprintln(w, prefix+line)
}
