//go:build !linux && !darwin

package main

import "os"

func consoleColumns() int { return 60 }

// Windows and other systems use the existing command-and-Enter input mode.
func enableRawMode(_ *os.File) (func(), bool) { return nil, false }

func (in *rawInput) escapeByteReady() (bool, error) {
	return in.Buffered() > 0, nil
}
