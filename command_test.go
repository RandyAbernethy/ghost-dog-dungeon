package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestEscapeReturnsToMapFromCommandScreensForFree(t *testing.T) {
	for _, screen := range []string{"i", "c", "m", "book", "equipment", "challenges"} {
		t.Run(screen, func(t *testing.T) {
			g := openArena()
			g.player.WebbedTurns, g.player.HexedTurns, g.player.ShieldTurns = 1, 2, 3
			g.stats.Turns = 12
			rat := newMonster(monsterRat)
			rat.Pos = pos{6, 5}
			g.current().Monsters = []*monster{&rat}
			beforePlayer, beforeRat, beforeStats, beforeRNG := g.player, rat, g.stats, g.rng.State
			var out bytes.Buffer
			if err := g.loop(strings.NewReader(screen+"\n\x1b\n"), &out); err != nil {
				t.Fatal(err)
			}
			if strings.Count(out.String(), "=== Ghost Dog Dungeon:") != 2 || strings.Contains(out.String(), "Unknown command") {
				t.Fatal("Esc should redraw the map after the command screen")
			}
			if !reflect.DeepEqual(g.player, beforePlayer) || rat != beforeRat || g.stats != beforeStats || g.rng.State != beforeRNG {
				t.Fatal("returning to the map must not spend a turn, advance monsters, or change status effects")
			}
		})
	}
}

func TestEscapeParsingPreservesArrowKeysAndFollowingCommands(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"\x1b[A", "w"}, {"\x1b[B", "s"}, {"\x1b[C", "d"}, {"\x1b[D", "a"},
		{"\x1b[H", "q"}, {"\x1b[5~", "e"}, {"\x1b[F", "z"}, {"\x1b[6~", "x"},
		{"\x1bOH", "q"}, {"\x1bi", "map"}, {":drop 5\x1bi", "map"},
	} {
		t.Run(fmt.Sprintf("%q", tc.input), func(t *testing.T) {
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer read.Close()
			defer write.Close()
			if _, err := io.WriteString(write, tc.input); err != nil {
				t.Fatal(err)
			}
			in := &rawInput{Reader: bufio.NewReader(read), file: read}
			cmd, eof, err := readRawCommand(in, io.Discard)
			if err != nil || eof || cmd != tc.want {
				t.Fatalf("readRawCommand: cmd=%q eof=%t err=%v, want %q", cmd, eof, err, tc.want)
			}
			if strings.HasSuffix(tc.input, "i") {
				cmd, eof, err = readRawCommand(in, io.Discard)
				if err != nil || eof || cmd != "inventory" {
					t.Fatal("Esc must leave the following key available for the next command")
				}
			}
		})
	}
}

func TestStandaloneEscapeInRawTerminalReturnsWithoutAnotherKey(t *testing.T) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	var unlock, number uint32
	for _, request := range []struct {
		code uintptr
		arg  *uint32
	}{{syscall.TIOCSPTLCK, &unlock}, {syscall.TIOCGPTN, &number}} {
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), request.code, uintptr(unsafe.Pointer(request.arg)))
		if errno != 0 {
			t.Fatal(errno)
		}
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	g := openArena()
	g.player.WebbedTurns, g.player.HexedTurns, g.player.ShieldTurns = 1, 2, 3
	g.stats.Turns = 17
	rat := newMonster(monsterRat)
	rat.Pos = pos{6, 5}
	g.current().Monsters = []*monster{&rat}
	beforePlayer, beforeRat, beforeStats, beforeRNG := g.player, rat, g.stats, g.rng.State
	out := &commandPromptWriter{prompts: make(chan struct{}, 8)}
	done := make(chan error, 1)
	go func() { done <- g.loop(slave, out) }()
	awaitPrompt := func() {
		t.Helper()
		select {
		case <-out.prompts:
		case err := <-done:
			t.Fatalf("game ended before the next prompt: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatal("the command reader waited for another key instead of returning to the map")
		}
	}
	awaitPrompt()
	for _, keys := range []string{"i", "\x1b", ":drop 5\x1b"} {
		if _, err := io.WriteString(master, keys); err != nil {
			t.Fatal(err)
		}
		awaitPrompt()
	}
	if _, err := io.WriteString(master, "\x03"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("game did not finish after Ctrl-C")
	}
	if strings.Count(out.String(), "=== Ghost Dog Dungeon:") != 3 || !strings.Contains(out.String(), "=== Inventory & Equipment ===") {
		t.Fatal("each standalone Esc should display the map immediately")
	}
	if !reflect.DeepEqual(g.player, beforePlayer) || rat != beforeRat || g.stats != beforeStats || g.rng.State != beforeRNG || len(g.current().Items) != 0 {
		t.Fatal("Esc must cancel the pending drop and leave game time and state unchanged")
	}
}

type commandPromptWriter struct {
	bytes.Buffer
	prompts chan struct{}
}

func (out *commandPromptWriter) Write(p []byte) (int, error) {
	n, err := out.Buffer.Write(p)
	if strings.HasSuffix(string(p), "Command> ") {
		out.prompts <- struct{}{}
	}
	return n, err
}
