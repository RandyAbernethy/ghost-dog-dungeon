package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestEscapeReturnsToMapFromCommandScreensForFree(t *testing.T) {
	for _, screen := range []string{"i", "c", "m", "book", "equipment", "challenges"} {
		t.Run(screen, func(t *testing.T) {
			g := openArena()
			g.player.BoundTurns, g.player.HexedTurns, g.player.ShieldTurns = 1, 2, 3
			g.stats.Turns = 12
			g.messages = []string{"You strike an enemy.", "The enemy recoils."}
			beforeLog := append([]string(nil), g.messages...)
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
			if !reflect.DeepEqual(g.player, beforePlayer) || rat != beforeRat || g.stats != beforeStats || g.rng.State != beforeRNG || !reflect.DeepEqual(g.messages, beforeLog) {
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
