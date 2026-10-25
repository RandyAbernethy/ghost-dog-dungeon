package main

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOpeningArtIsPortableASCII(t *testing.T) {
	if puppyArt == "" || !strings.HasSuffix(puppyArt, "\n") {
		t.Fatal("the puppy asset must be nonempty and end with a line feed")
	}
	for _, b := range []byte(puppyArt) {
		if b != '\n' && (b < ' ' || b > '~') {
			t.Fatalf("the puppy asset contains a non-printable ASCII byte: %#x", b)
		}
	}
	lines := strings.Split(strings.TrimSuffix(puppyArt, "\n"), "\n")
	if len(lines) > 24 {
		t.Fatal("keep the opening portrait within 24 lines")
	}
	for _, line := range lines {
		if len(line) > 60 {
			t.Fatal("keep the opening portrait within 60 columns")
		}
	}
	var out bytes.Buffer
	renderOpeningArt(&out)
	if got, want := out.String(), "\n"+puppyArt; got != want {
		t.Fatal("rendering must preserve the plain-text asset's spacing exactly")
	}
}

func TestOpeningArtAppearsOnceWithoutSpendingATurn(t *testing.T) {
	for _, turns := range []int{0, 17} {
		t.Run(fmt.Sprintf("turn_%d", turns), func(t *testing.T) {
			g := openArena()
			g.recordsFile = filepath.Join(t.TempDir(), "records.json")
			g.stats.Turns = turns
			g.messages = []string{"The dungeon waits."}
			beforePlayer, beforeStats, beforeRNG := g.player, g.stats, g.rng.State
			beforeMessages := append([]string(nil), g.messages...)
			var out bytes.Buffer
			if err := g.loop(strings.NewReader("map\ni\nmap\nquit\n"), &out); err != nil {
				t.Fatal(err)
			}
			output := out.String()
			prefix := fmt.Sprintf("Ghost Dog Dungeon version %s\nDungeon seed: %d\n\n%s", gameVersion, g.seed, puppyArt)
			if !strings.HasPrefix(output, prefix) || strings.Count(output, puppyArt) != 1 {
				t.Fatal("the puppy must appear once after the version and seed, not on map redraws")
			}
			if !strings.Contains(output, "=== Inventory & Equipment ===") || strings.Count(output, "=== Ghost Dog Dungeon:") != 3 || !g.quit {
				t.Fatal("the opening portrait must not consume or change any commands")
			}
			if !reflect.DeepEqual(g.player, beforePlayer) || g.stats != beforeStats || g.rng.State != beforeRNG || !reflect.DeepEqual(g.messages, beforeMessages) {
				t.Fatal("the opening portrait and reading screens must not change gameplay state")
			}
		})
	}
}

func TestInformationalFlagsDoNotShowOpeningArt(t *testing.T) {
	for _, flag := range []string{"--help", "--version"} {
		var stdout, stderr bytes.Buffer
		if err := run([]string{flag}, strings.NewReader(""), &stdout, &stderr); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(stdout.String()+stderr.String(), puppyArt) {
			t.Fatalf("%s must not show the opening portrait", flag)
		}
	}
}

func TestNewGameOpeningUsesChapelIntroduction(t *testing.T) {
	g := newGame(42)
	text := strings.Join(g.messages, " ")
	want := "After coming upon an old chapel, you find an entrance to a dark cellar in the chapel yard. Your adventurous spirit drives you to explore. The wind howls through the bell tower, like a far away hound longing for home. You climb down into the cellar below the old chapel and the door slams shut behind you. Some great evil is afoot and clearly you must face it to escape the catacombs."
	if text != want || g.stats.Turns != 0 {
		t.Fatal("new games must show the requested chapel introduction before the first turn")
	}
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(loaded.messages, " ") != want || loaded.rng.State != g.rng.State {
		t.Fatal("saving and loading must preserve the introduction and RNG")
	}
}

func TestWelcomeEnterDoesNotConsumeGameplayInput(t *testing.T) {
	for _, raw := range []bool{false, true} {
		t.Run(fmt.Sprintf("raw_%t", raw), func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader("not yet\n\nquit\n"))
			var input *rawInput
			if raw {
				input = &rawInput{Reader: bufio.NewReader(strings.NewReader("not yet\rquit\r"))}
			}
			begin, err := waitForDescent(scanner, input)
			if err != nil || !begin {
				t.Fatalf("welcome Enter: %t %v", begin, err)
			}
			if raw {
				cmd, eof, err := readRawMenuCommand(input, &bytes.Buffer{})
				if err != nil || eof || cmd != "q" {
					t.Fatalf("first gameplay key consumed: %q %v", cmd, err)
				}
			} else if !scanner.Scan() || scanner.Text() != "quit" {
				t.Fatal("welcome consumed the first gameplay command")
			}
		})
	}
	for _, keys := range []string{"", "\x03", "\x04"} {
		begin, err := waitForDescent(nil, &rawInput{Reader: bufio.NewReader(strings.NewReader(keys))})
		if err != nil || begin {
			t.Fatalf("EOF or quit must leave welcome without gameplay: %t %v", begin, err)
		}
	}
}
