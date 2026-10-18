package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAshMomentsRespectCompanionStateAndExploration(t *testing.T) {
	for _, freed := range []bool{false, true} {
		for _, alive := range []bool{false, true} {
			g := openArena()
			g.dog.Freed, g.dog.Alive = freed, alive
			g.ashMoment("test", "A quiet moment.")
			g.ashMoment("test", "A repeated moment.")
			want := 0
			if freed && alive {
				want = 1
			}
			if len(g.messages) != want || len(g.adventure.AshMoments) != want {
				t.Fatal("moments require living, freed Ash and occur once per run")
			}
		}
	}
	g := openArena()
	g.dog.Freed, g.dog.Pos = true, pos{4, 5}
	g.current().Theme = themeChapel
	g.stats.Turns = 15
	m := newMonster(monsterRat)
	m.Pos = pos{9, 5}
	g.current().Monsters = []*monster{&m}
	g.finishAdventureTurn()
	if len(g.adventure.AshMoments) != 0 {
		t.Fatal("ambient moments must wait until nearby combat has ended")
	}
	g.current().Monsters = nil
	rng := g.rng.State
	g.processCommand(".")
	if len(g.adventure.AshMoments) != 1 || !strings.Contains(strings.Join(g.messages, "\n"), "bell you cannot hear") || g.rng.State != rng {
		t.Fatal("a quiet chapel turn should trigger one deterministic memory")
	}
	g.player.HP = 1
	g.stats.Turns += 20
	g.processCommand(".")
	if len(g.adventure.AshMoments) != 1 {
		t.Fatal("ambient moments must be limited to one per floor")
	}
	g.currentLevel = 1
	g.processCommand(".")
	if len(g.adventure.AshMoments) != 2 {
		t.Fatal("a later floor may trigger another quiet moment after the cooldown")
	}
	g = saveAndReloadAdventure(t, g)
	before := len(g.adventure.AshMoments)
	g.ashMoment("injured", "This must not repeat.")
	if len(g.adventure.AshMoments) != before {
		t.Fatal("loading must retain the set of moments already seen")
	}
}

func TestAshRescueReturnAndFountainMomentsFollowRealActions(t *testing.T) {
	g := openArena()
	chain := pos{6, 5}
	g.current().DogChain = &chain
	g.processCommand("d")
	if g.adventure.RescueFloor != 1 || len(g.adventure.AshMoments) != 1 || g.stats.Turns != 1 || !g.dog.Freed {
		t.Fatal("rescuing Ash must record the floor and one personality moment on the action's turn")
	}
	g.hurtDog(1000, "The test enemy")
	g.player.GhostRecallScrolls = 1
	g.processCommand("r")
	if !g.dog.Alive || g.stats.GhostDogFalls != 1 || len(g.adventure.AshMoments) != 2 || g.stats.Turns != 2 {
		t.Fatal("an explicit revival must record Ash's return without adding a turn")
	}
	g.dog.HP = 20
	g.current().Fountains = []fountain{{Pos: g.player.Pos}}
	g.processCommand("v")
	if len(g.adventure.AshMoments) != 3 || g.stats.Turns != 3 || !strings.Contains(strings.Join(g.messages, "\n"), "Silver ripples") {
		t.Fatal("drinking with living Ash should add the fountain moment to the same turn")
	}
	saveAndReloadAdventure(t, g)
}

func TestEveryCollectedSideStoryChangesTheEndingWithoutSpoilers(t *testing.T) {
	g := openArena()
	g.dog.Freed = true
	var base bytes.Buffer
	g.renderEnding(&base)
	for i, story := range ghostDogSideStories {
		g.player.SideStories = []int{i + 1}
		var ending bytes.Buffer
		g.renderEnding(&ending)
		if ending.String() == base.String() {
			t.Fatalf("collected story %q should contribute to the ending", story.Title)
		}
	}
	for _, alive := range []bool{false, true} {
		g.dog.Alive = alive
		g.player.SideStories = []int{9, 10}
		beforeRNG, beforeStats := g.rng.State, g.stats
		beforePlayer := g.player
		var ending bytes.Buffer
		g.renderEnding(&ending)
		if !strings.Contains(ending.String(), "Mara's dream") || !strings.Contains(ending.String(), "final line") ||
			(!alive && strings.Contains(ending.String(), "Ash crosses the threshold")) ||
			(alive && !strings.Contains(ending.String(), "Ash crosses the threshold")) {
			t.Fatal("the dream and song must reflect both collected stories and Ash's survival")
		}
		if g.rng.State != beforeRNG || g.stats != beforeStats || !reflect.DeepEqual(beforePlayer, g.player) {
			t.Fatal("the ending must not change gameplay state")
		}
	}
	var fresh bytes.Buffer
	g.player.SideStories = nil
	g.renderEnding(&fresh)
	if strings.Contains(fresh.String(), "Mara") || strings.Contains(fresh.String(), "song") {
		t.Fatal("an uncollected story must not contribute its ending")
	}
}

func TestChronicleUsesRecordedEventsAndOnlyFinishedRuns(t *testing.T) {
	g := openArena()
	g.adventure = adventureState{RescueFloor: 2, EpicsDefeated: []string{"Epic Ashen Wyrm"}, ObjectivesDone: []string{string(objectiveSigils)}}
	g.levels[4].Visited = true
	var out bytes.Buffer
	g.renderChronicle(&out)
	for _, want := range []string{"floor 5", "silver chain on floor 2", "Epic Ashen Wyrm", string(objectiveSigils), "Rusty Knife"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("chronicle is missing a recorded fact: %q", want)
		}
	}
	out.Reset()
	if err := g.loop(strings.NewReader("quit\n"), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "=== Your Adventure ===") {
		t.Fatal("an unfinished run must not show an ending chronicle")
	}
	g = openArena()
	g.player.HP = 0
	g.recordsFile = filepath.Join(t.TempDir(), "records.json")
	out.Reset()
	if err := g.loop(strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "=== Your Adventure ===") || strings.Contains(out.String(), "You freed Ash") || strings.Contains(out.String(), "You defeated Epic") || strings.Contains(out.String(), "You completed The") {
		t.Fatal("death must show the chronicle without inventing missing history")
	}
}

func TestOlderFormatTwoSavesLoadWithoutInventedFeatures(t *testing.T) {
	g := openArena()
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	var old map[string]json.RawMessage
	if err := json.Unmarshal(data, &old); err != nil {
		t.Fatal(err)
	}
	delete(old, "adventure")
	data, err = json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(g.saveDir, "older-format-2.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(path)
	if err != nil {
		t.Fatal(err)
	}
	if saveFormatVersion != 2 || !reflect.DeepEqual(g.levels, loaded.levels) || !reflect.DeepEqual(g.adventure, loaded.adventure) || g.rng.State != loaded.rng.State {
		t.Fatal("older format-2 saves must load without regeneration, RNG draws or invented events")
	}
}
