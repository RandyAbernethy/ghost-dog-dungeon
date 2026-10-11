package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func crowdedLogArena() *game {
	g := openArena()
	g.rng = newSimpleRNG(1)
	g.stats.Turns = 80
	g.messages = []string{"An event from the previous turn."}
	g.player.HP, g.player.MaxHP = 1000, 1000
	g.player.Weapon = weapon{Name: "Silver test blade", Min: 3, Max: 3, Magic: true}
	g.dog.Freed, g.dog.Pos = true, pos{5, 4}
	g.dog.HP, g.dog.MaxHP = 1000, 1000
	foes := []struct {
		kind monsterKind
		spot pos
	}{
		{monsterSkeleton, pos{6, 5}},
		{monsterOrc, pos{4, 4}}, {monsterOrc, pos{4, 5}}, {monsterOrc, pos{4, 6}},
		{monsterOrc, pos{5, 6}}, {monsterOrc, pos{6, 6}}, {monsterOrc, pos{6, 4}},
		{monsterVoidSeer, pos{8, 3}}, {monsterVoidSeer, pos{9, 4}}, {monsterVoidSeer, pos{10, 5}},
		{monsterGloomArcher, pos{8, 7}}, {monsterGloomArcher, pos{9, 8}}, {monsterGloomArcher, pos{10, 9}},
	}
	for i, foe := range foes {
		m := newMonster(foe.kind)
		m.Name, m.Pos = fmt.Sprintf("%s %d", m.Name, i+1), foe.spot
		m.HP, m.MaxHP, m.MinDamage, m.MaxDamage = 200, 200, 2, 2
		g.current().Monsters = append(g.current().Monsters, &m)
	}
	return g
}

func assertWholeTurnRendered(t *testing.T, g *game, output string) {
	t.Helper()
	header := fmt.Sprintf("Turn %d events:\n", g.stats.Turns)
	_, remaining, ok := strings.Cut(output, header)
	if !ok {
		t.Fatalf("missing current turn heading %q", header)
	}
	for _, message := range g.messages {
		_, rest, found := strings.Cut(remaining, "- "+message+"\n")
		if !found {
			t.Fatalf("missing or reordered event %q", message)
		}
		remaining = rest
	}
	if strings.Contains(output, "An event from the previous turn.") {
		t.Fatal("current turn must not display an earlier turn")
	}
}

func TestCrowdedCombatLogKeepsPlayerDogRangedAndMeleeEvents(t *testing.T) {
	g := crowdedLogArena()
	g.processCommand("d")
	if g.stats.Turns != 81 || len(g.messages) <= 8 || !strings.HasPrefix(g.messages[0], "You strike Skeleton 1") {
		t.Fatalf("combat should start a complete new turn with your strike: %v", g.messages)
	}
	joined := strings.Join(g.messages, "\n")
	for _, event := range []string{"flares against the undead", "Ghost dog tears into", "hits you", "tears at your thoughts", "fires a barbed arrow"} {
		if !strings.Contains(joined, event) {
			t.Fatalf("crowded turn lost %q: %s", event, joined)
		}
	}
	beforePlayer, beforeDog, beforeStats, beforeRNG := g.player, g.dog, g.stats, g.rng.State
	beforeLog := append([]string(nil), g.messages...)
	var out bytes.Buffer
	g.render(&out)
	assertWholeTurnRendered(t, g, out.String())
	if !reflect.DeepEqual(beforePlayer, g.player) || beforeDog != g.dog || beforeStats != g.stats || beforeRNG != g.rng.State || !reflect.DeepEqual(beforeLog, g.messages) {
		t.Fatal("rendering a complete turn must not change gameplay or its log")
	}
}

func TestEachSuccessfulActionReplacesThePreviousTurnLog(t *testing.T) {
	g := openArena()
	g.messages = []string{"An event from the previous turn."}
	g.processCommand("d")
	if !reflect.DeepEqual(g.messages, []string{"You move east."}) || g.stats.Turns != 1 {
		t.Fatal("quiet movement should begin its own turn")
	}
	potion := item{Pos: pos{6, 4}, Kind: itemPotion, Name: "healing potion", Glyph: '!'}
	g.current().Items = []item{potion}
	g.processCommand("w")
	if len(g.messages) != 2 || g.messages[0] != "You move north." || !strings.Contains(g.messages[1], "stash a healing potion") || g.stats.Turns != 2 {
		t.Fatalf("movement should precede its pickup: %v", g.messages)
	}
	g.processCommand(".")
	if !reflect.DeepEqual(g.messages, []string{"You wait and let the dungeon move around you."}) || g.stats.Turns != 3 {
		t.Fatal("a quiet wait must not retain the last move or pickup")
	}
	g.player.WebbedTurns = 1
	beforePos := g.player.Pos
	g.processCommand("d")
	if !reflect.DeepEqual(g.messages, []string{"Sticky webs hold you in place for a turn."}) || g.stats.Turns != 4 || g.player.Pos != beforePos {
		t.Fatal("a turn spent entangled should replace the previous log too")
	}
}

func TestFreeAndFailedActionsKeepCurrentTurnAvailable(t *testing.T) {
	g := openArena()
	g.processCommand(".")
	current := append([]string(nil), g.messages...)
	g.player.Potions = 0
	g.current().Tiles[5][4] = '#'
	g.current().Events = []floorEvent{testFriendlyGhost(pos{6, 5})}
	beforePlayer, beforeStats, beforeRNG := g.player, g.stats, g.rng.State
	for _, cmd := range []string{"a", "p", "equip 999", "drop 1", "unknown", "y", "h"} {
		g.processCommand(cmd)
		if !reflect.DeepEqual(current, g.messages[:len(current)]) || !reflect.DeepEqual(g.player, beforePlayer) || g.stats != beforeStats || g.rng.State != beforeRNG {
			t.Fatalf("free command %q must keep the current log and leave game time unchanged", cmd)
		}
	}
	beforeLog := append([]string(nil), g.messages...)
	var out bytes.Buffer
	if err := g.loop(strings.NewReader("i\nc\nm\nbook\nequipment\nchallenges\n\x1b\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeLog, g.messages) || !reflect.DeepEqual(g.player, beforePlayer) || g.stats != beforeStats || g.rng.State != beforeRNG {
		t.Fatal("reading screens and Esc should preserve the same complete turn")
	}
	g.processCommand(".")
	if len(g.messages) != 1 || g.messages[0] != current[0] || g.stats.Turns != beforeStats.Turns+1 {
		t.Fatal("the next real turn should also clear free-command feedback")
	}
}

func TestCurrentTurnLogSurvivesSaveLoadWithoutMergingOtherTurns(t *testing.T) {
	g := crowdedLogArena()
	g.saveFile, g.timestampedSaves = filepath.Join(t.TempDir(), "turn.json"), false
	g.processCommand("d")
	want := append([]string(nil), g.messages...)
	beforeStats, beforeRNG := g.stats, g.rng.State
	g.processCommand("save")
	if !reflect.DeepEqual(want, g.messages[:len(want)]) || g.stats != beforeStats || g.rng.State != beforeRNG {
		t.Fatal("saving should retain the combat log and spend no turn")
	}
	g.processCommand(".")
	g.processCommand("load")
	if len(g.messages) != len(want)+1 || !reflect.DeepEqual(want, g.messages[:len(want)]) || !strings.HasPrefix(g.messages[len(want)], "Game loaded from ") || g.stats != beforeStats || g.rng.State != beforeRNG {
		t.Fatal("loading must restore the saved turn, without merging another turn's events")
	}
	var out bytes.Buffer
	g.render(&out)
	assertWholeTurnRendered(t, g, out.String())
}

func TestPlayerActionPrecedesMagicFountainAndFloorTransition(t *testing.T) {
	t.Run("hexed magic strike", func(t *testing.T) {
		g := crowdedLogArena()
		g.player.HexedTurns = 1
		g.processCommand("d")
		if !strings.HasPrefix(g.messages[0], "You strike ") || !strings.Contains(strings.Join(g.messages, "\n"), "The hex weakens your strike.") {
			t.Fatal("the attack must precede its magic and curse details")
		}
	})
	t.Run("sun orb", func(t *testing.T) {
		g := crowdedLogArena()
		g.player.SunOrbs = 1
		g.processCommand("u")
		if g.messages[0] != "You unleash the sun orb's fire." {
			t.Fatal("using the orb should precede its victims' reactions")
		}
	})
	t.Run("fountain", func(t *testing.T) {
		g := openArena()
		g.dog.Freed, g.dog.Pos, g.dog.HP = true, pos{5, 4}, 10
		g.player.HP = 10
		g.current().Fountains = []fountain{{Pos: g.player.Pos}}
		g.processCommand("v")
		if !strings.HasPrefix(g.messages[0], "You drink from the fountain") || !strings.Contains(g.messages[1], "restores") {
			t.Fatal("your drink should precede Ghost Dog's healing")
		}
	})
	t.Run("stairs", func(t *testing.T) {
		g := openArena()
		g.current().Stairs, g.current().HasStair = g.player.Pos, true
		g.levels[1].Monsters = nil
		g.dog.Freed, g.dog.HP = true, 10
		g.processCommand(">")
		if g.messages[0] != "You descend to level 2." || !strings.Contains(g.messages[1], "recovers") {
			t.Fatal("descending should precede companion recovery")
		}
	})
}

func TestCompleteTurnLogThroughRealExecutableAndSavedGame(t *testing.T) {
	dir := t.TempDir()
	g := crowdedLogArena()
	g.saveFile, g.timestampedSaves = filepath.Join(dir, "combat.json"), false
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "ghostdog")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	cmd := exec.Command(binary, "--load-file", g.saveFile)
	cmd.Env = append(os.Environ(), "HOME="+dir)
	cmd.Stdin = strings.NewReader("d\ni\n\x1b\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, output)
	}
	g.processCommand("d")
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.messages, loaded.messages) || loaded.stats.Turns != 81 || len(loaded.messages) <= 8 {
		t.Fatal("the executable must preserve the entire crowded turn through reading screens and saving")
	}
	for _, screen := range strings.Split(string(output), "=== Ghost Dog Dungeon:")[1:] {
		if strings.Contains(screen, "Turn 81 events:") {
			assertWholeTurnRendered(t, loaded, screen)
		}
	}
}
