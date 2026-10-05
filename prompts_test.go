package main

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestObjectPromptsRepeatOnlyAfterLeavingAndReturning(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind eventKind
		key  string
		info string
	}{
		{"friendly ghost", eventGhost, "Press y", "choice of three gifts"},
		{"healing shrine", eventShrine, "press y", "heals you and a living Ghost Dog"},
		{"keepsake", eventKeepsake, "press y", "warding charm and restore his health"},
		{"lever", eventLever, "Press y", "open a passage"},
		{"fountain", "", "Press v", "refills when you leave this floor"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			if tc.kind == "" {
				g.current().Fountains = []fountain{{Pos: pos{8, 5}}}
			} else {
				g.current().Events = []floorEvent{{Kind: tc.kind, Pos: pos{8, 5}}}
			}
			move := func(cmd string, wantPrompt bool) {
				t.Helper()
				g.messages = nil
				g.processCommand(cmd)
				count := 0
				for _, message := range g.messages {
					if strings.Contains(message, tc.info) {
						count++
						if !strings.Contains(message, tc.key) {
							t.Fatalf("prompt lacks the interaction key: %s", message)
						}
					}
				}
				if count != 0 && !wantPrompt || count != 1 && wantPrompt {
					t.Fatalf("command %q produced %d prompts, want prompt=%t: %v", cmd, count, wantPrompt, g.messages)
				}
			}
			move("d", false) // Two tiles away.
			rngBefore := g.rng.State
			move("d", true) // Adjacent.
			if g.stats.Turns != 2 || g.rng.State != rngBefore {
				t.Fatal("the arrival prompt must not spend another turn or use randomness")
			}
			move(".", false)
			move("d", false) // On the object.
			move("w", false)
			move("a", false) // Diagonally adjacent.
			move("s", false)
			g.current().Tiles[6][7] = '#'
			turns := g.stats.Turns
			move("s", false) // A blocked move is not another arrival.
			if g.stats.Turns != turns {
				t.Fatal("a blocked move should remain free")
			}
			g.player.WebbedTurns = 1
			move("d", false) // Entanglement does not move the player.
			move("a", false) // Outside the object's immediate area.
			move("e", true)  // Return diagonally.
			if tc.kind == "" {
				g.current().Fountains[0].Used = true
			} else {
				g.current().Events[0].Used = true
			}
			move("a", false)
			move("d", false) // Spent objects have no available interaction.
		})
	}
}

func TestNearbyObjectsPromptIndependently(t *testing.T) {
	g := openArena()
	g.current().Events = []floorEvent{{Kind: eventGhost, Pos: pos{8, 4}}}
	g.current().Fountains = []fountain{{Pos: pos{8, 6}}}
	g.processCommand("d")
	g.messages = nil
	g.processCommand("d")
	if len(g.messages) != 2 || !strings.Contains(g.messages[0], "friendly ghost") || !strings.Contains(g.messages[1], "healing fountain") {
		t.Fatalf("each newly nearby object should prompt: %v", g.messages)
	}
	g.messages = nil
	g.processCommand("w") // Leave only the fountain's area.
	g.processCommand("s")
	if len(g.messages) != 1 || !strings.Contains(g.messages[0], "healing fountain") {
		t.Fatalf("returning to one object should not repeat another's prompt: %v", g.messages)
	}
}

func TestHiddenObjectsDoNotPromptUntilApproachedAfterDiscovery(t *testing.T) {
	g := openArena()
	lvl := g.current()
	lvl.Secret = &secretRoom{Door: pos{7, 5}, Tiles: []pos{{8, 5}, {8, 6}, {9, 5}, {9, 6}}}
	lvl.Fountains = []fountain{{Pos: pos{8, 5}}}
	for _, p := range append([]pos{lvl.Secret.Door}, lvl.Secret.Tiles...) {
		lvl.Tiles[p.Y][p.X] = '#'
	}
	g.processCommand("d")
	g.messages = nil
	g.processCommand("e") // Diagonally beside the concealed fountain.
	if len(g.messages) != 0 {
		t.Fatalf("concealed objects should not be announced: %v", g.messages)
	}
	g.processCommand("a")
	g.processCommand("search")
	g.messages = nil
	g.processCommand("d")
	if len(g.messages) != 1 || !strings.Contains(g.messages[0], "healing fountain") {
		t.Fatalf("approaching a revealed fountain should prompt: %v", g.messages)
	}
}

func TestArrivalPromptsOnStairsAndBlink(t *testing.T) {
	t.Run("stairs", func(t *testing.T) {
		g := openArena()
		first, second := g.current(), g.levels[1]
		for y := 1; y < mapHeight-1; y++ {
			for x := 1; x < mapWidth-1; x++ {
				second.Tiles[y][x] = '.'
			}
		}
		second.Monsters, second.Items, second.Fountains = nil, nil, nil
		second.Secret, second.ExtraSecrets, second.Special, second.DogChain = nil, nil, nil, nil
		first.Stairs, second.UpStairs = g.player.Pos, g.player.Pos
		first.HasStair, second.HasUpStair = true, true
		second.Events = []floorEvent{{Kind: eventGhost, Pos: pos{6, 5}}}
		first.Fountains = []fountain{{Pos: pos{6, 5}, Used: true}}
		g.messages = nil
		g.processCommand(">")
		if !strings.Contains(strings.Join(g.messages, " "), "friendly ghost offers a choice") || g.stats.Turns != 1 {
			t.Fatal("descending should announce nearby objects on the destination floor")
		}
		g.messages = nil
		g.processCommand("<")
		if !strings.Contains(strings.Join(g.messages, " "), "healing fountain flows nearby") || first.Fountains[0].Used || g.stats.Turns != 2 {
			t.Fatal("returning should announce the renewed fountain even at matching coordinates")
		}
	})
	t.Run("blink", func(t *testing.T) {
		g := blinkArena(pos{3, 5}, pos{10, 10}, pos{11, 10})
		foe := newMonster(monsterOrc)
		foe.Pos, foe.FrozenTurns = pos{3, 5}, 1
		g.current().Monsters = []*monster{&foe}
		g.current().Events = []floorEvent{{Kind: eventKeepsake, Pos: pos{10, 10}}}
		g.messages = nil
		g.processCommand("g")
		if !strings.Contains(strings.Join(g.messages, " "), "Ash's keepsake, a faded collar") || g.stats.Turns != 1 || g.current().Events[0].Used {
			t.Fatal("blinking beside an object should describe it without activating it")
		}
	})
}

func TestObjectPromptsStayQuietAcrossScreensAndSaveLoad(t *testing.T) {
	g := openArena()
	g.current().Events = []floorEvent{{Kind: eventGhost, Pos: pos{8, 5}}}
	g.processCommand("d")
	g.processCommand("d")
	g.saveFile, g.timestampedSaves = filepath.Join(t.TempDir(), "prompts.json"), false
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	before := append([]string(nil), loaded.messages...)
	var out bytes.Buffer
	loaded.render(&out)
	loaded.renderInventory(&out)
	loaded.renderInspect(&out)
	loaded.renderBook(&out)
	loaded.render(&out)
	if !reflect.DeepEqual(before, loaded.messages) || strings.Contains(loaded.contextHint(), "interact") {
		t.Fatal("reading screens must not add or repeat object prompts")
	}
	loaded.messages = nil
	loaded.processCommand("d")
	loaded.processCommand("w")
	if len(loaded.messages) != 0 || loaded.stats.Turns != 4 {
		t.Fatal("loading and moving within the area should preserve prompt suppression")
	}
	loaded.processCommand("a")
	loaded.processCommand("a")
	loaded.processCommand("d")
	if len(loaded.messages) != 1 || !strings.Contains(loaded.messages[0], "friendly ghost offers a choice") {
		t.Fatal("leaving and returning after loading should prompt again")
	}
}
