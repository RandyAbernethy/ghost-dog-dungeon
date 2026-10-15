package main

import (
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
	g.player.BoundTurns, g.player.HexedTurns, g.player.ShieldTurns = 1, 2, 3
	g.stats.Turns = 17
	g.messages = []string{"You strike an enemy.", "The enemy recoils."}
	beforeLog := append([]string(nil), g.messages...)
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
	if !strings.HasSuffix(out.String(), fmt.Sprintf("Dungeon seed: %d\n", g.seed)) {
		t.Fatal("quitting with Ctrl-C should finish by reporting the dungeon seed")
	}
	if !reflect.DeepEqual(g.player, beforePlayer) || rat != beforeRat || g.stats != beforeStats || g.rng.State != beforeRNG || len(g.current().Items) != 0 || !reflect.DeepEqual(g.messages, beforeLog) {
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
