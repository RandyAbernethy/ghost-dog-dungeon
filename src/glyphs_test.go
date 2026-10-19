package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestMapSymbolsAreUniqueAcrossFeaturesMonstersItemsAndTraps(t *testing.T) {
	seen := map[rune]string{}
	claim := func(ch rune, name string) {
		t.Helper()
		if ch < '!' || ch > '~' {
			t.Fatalf("%s has a missing or non-ASCII symbol %q", name, ch)
		}
		if previous, ok := seen[ch]; ok {
			t.Fatalf("symbol %q represents both %s and %s", ch, previous, name)
		}
		seen[ch] = name
	}
	for _, feature := range []struct {
		ch   rune
		name string
	}{
		{glyphWall, "wall"}, {glyphFloor, "floor"}, {glyphPlayer, "player"},
		{glyphDog, "Ghost Dog"}, {glyphChainedDog, "chained Ghost Dog"},
		{glyphStairsDown, "stairs down"}, {glyphStairsUp, "stairs up/out"},
		{glyphFountain, "fountain"}, {glyphSpentFountain, "spent fountain"},
		{glyphEpicWarning, "Epic warning"}, {glyphCryptEntrance, "crypt entrance"},
		{glyphCryptExit, "crypt exit"}, {glyphSealedCrypt, "sealed crypt entrance"},
	} {
		claim(feature.ch, feature.name)
	}
	for _, kind := range []eventKind{eventGhost, eventKeepsake, eventLever, eventShrine} {
		e := floorEvent{Kind: kind}
		claim(e.glyph(), string(kind))
	}
	for _, kind := range []objectiveKind{objectiveSpirit, objectiveSigils} {
		o := objectiveRoom{Kind: kind, Pos: pos{5, 5}, Sigils: []ritualSigil{{Pos: pos{5, 5}}}}
		claim(o.marker(o.Pos), string(kind))
	}
	if len(monsterGlyphs) != len(allMonsterKinds) {
		t.Fatal("the symbol catalog must cover the complete monster roster")
	}
	for _, kind := range allMonsterKinds {
		m := newMonster(kind)
		claim(m.glyph(), string(kind))
	}
	if len(itemGlyphs) != int(itemSideStoryScroll)+1 || len(trapGlyphs) != len(cryptTrapKinds) {
		t.Fatal("the symbol catalog must cover all item and trap kinds")
	}
	for kind := itemWeapon; kind <= itemSideStoryScroll; kind++ {
		claim((item{Kind: kind}).glyph(), fmt.Sprintf("item kind %d", kind))
	}
	for _, kind := range cryptTrapKinds {
		claim((cryptTrap{Kind: kind}).glyph(), "trap: "+string(kind))
	}
	if seen['F'] != "fountain" || seen['f'] != "spent fountain" {
		t.Fatal("F and f must remain reserved for fountains")
	}
	t.Logf("audited %d distinct map symbols", len(seen))
}

func symbolMapRows(t *testing.T, output string) []string {
	t.Helper()
	var rows []string
	for _, line := range strings.Split(stripANSI(output), "\n") {
		if len(line) == mapWidth && (line[0] == '#' || line[0] == ' ') {
			rows = append(rows, line)
			if len(rows) == mapHeight {
				return rows
			}
		}
	}
	t.Fatalf("missing map rows in output:\n%s", output)
	return nil
}

func TestEveryMonsterAndItemDisplaysItsCurrentSymbol(t *testing.T) {
	g := openArena()
	for i, kind := range allMonsterKinds {
		m := newMonster(kind)
		m.Pos = pos{2 + i%30, 2 + i/30}
		g.current().Monsters = append(g.current().Monsters, &m)
	}
	for kind := itemWeapon; kind <= itemSideStoryScroll; kind++ {
		g.current().Items = append(g.current().Items, item{Kind: kind, Pos: pos{2 + int(kind), 7}})
	}
	rng, turns := g.rng.State, g.stats.Turns
	var output bytes.Buffer
	g.render(&output)
	rows := symbolMapRows(t, output.String())
	for _, m := range g.current().Monsters {
		if rune(rows[m.Pos.Y][m.Pos.X]) != monsterGlyphs[m.Kind] {
			t.Fatalf("%s should display the symbol for its kind", m.Kind)
		}
	}
	for _, it := range g.current().Items {
		if rune(rows[it.Pos.Y][it.Pos.X]) != itemGlyphs[it.Kind] {
			t.Fatalf("item kind %d should display the symbol for its kind", it.Kind)
		}
	}
	if g.rng.State != rng || g.stats.Turns != turns {
		t.Fatal("displaying the current symbols must not draw randomness or spend turns")
	}
}

func TestDiscoveredTrapsDisplayDistinctSymbolsAndSpentTrapsDisappear(t *testing.T) {
	for _, kind := range cryptTrapKinds {
		t.Run(string(kind), func(t *testing.T) {
			g := trappedArena(kind)
			g.crypt.Traps[0].Discovered = true
			p := g.crypt.Traps[0].Pos
			var output bytes.Buffer
			g.render(&output)
			rows := symbolMapRows(t, output.String())
			if rune(rows[p.Y][p.X]) != trapGlyphs[kind] || !strings.Contains(output.String(), fmt.Sprintf("Trap: %s (%c)", kind, trapGlyphs[kind])) {
				t.Fatal("the map and trap description must identify the same discovered hazard")
			}
			g.crypt.Traps[0].Spent = true
			output.Reset()
			g.render(&output)
			if symbolMapRows(t, output.String())[p.Y][p.X] != '.' {
				t.Fatal("a spent trap must no longer appear as a hazard")
			}
		})
	}
}

func TestMapSymbolsThroughExecutableAndSaveLoad(t *testing.T) {
	g := openArena()
	m := newMonster(monsterFrostMonarch)
	m.Pos = pos{8, 5}
	g.current().Monsters = []*monster{&m}
	g.current().Fountains = []fountain{{Pos: pos{6, 5}}, {Pos: pos{6, 6}, Used: true}}
	g.current().Items = []item{{Kind: itemWardingCharm, Name: "warding charm", Pos: pos{7, 5}}}
	g.player.WardingCharms = 1
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader("m\nc\ni\nsave\nload\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executable: %v\n%s", err, output)
	}
	rows := symbolMapRows(t, string(output))
	if rows[5][6] != 'F' || rows[6][6] != 'f' || rows[5][7] != ']' || rows[5][8] != '5' {
		t.Fatal("the fountain, spent fountain, warding charm and Frost Monarch must be visually distinct after loading")
	}
	for _, want := range []string{"Epic Frost Monarch (5)", "- 5 Epic Frost Monarch", "Warding Charm x1 (map ])", "F/f fountain/spent"} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("map, inspection, inventory and codex must use consistent symbols: missing %q", want)
		}
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.levels, g.levels) || !reflect.DeepEqual(loaded.player, g.player) || loaded.dog != g.dog || loaded.stats != g.stats || loaded.rng.State != g.rng.State {
		t.Fatal("displaying and saving a run must preserve gameplay state")
	}
}
