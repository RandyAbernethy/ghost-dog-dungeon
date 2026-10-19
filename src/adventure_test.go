package main

import (
	"bytes"
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
	memoryTerms := []string{
		"Mara's stolen glove", "chains below", "hundred little moons", "share for a hungry traveler",
		"nest Ash once saved", "weak bell ringing", "wooden soldier", "room for one more",
		"Mara's dream", "Mara's unfinished song", "room behind Mara's pantry",
	}
	for i, story := range ghostDogSideStories {
		g.player.SideStories = []int{i + 1}
		var ending bytes.Buffer
		g.renderEnding(&ending)
		if ending.String() == base.String() || !strings.Contains(ending.String(), memoryTerms[i]) {
			t.Fatalf("collected story %q should contribute to the ending", story.Title)
		}
		if len(strings.Split(strings.TrimSpace(ending.String()), "\n")) != 2 {
			t.Fatalf("collected story %q should add exactly one reminiscence after the bowl", story.Title)
		}
	}
	for _, alive := range []bool{false, true} {
		g.dog.Alive = alive
		beforeRNG, beforeStats := g.rng.State, g.stats
		for _, id := range []int{9, 10} {
			g.player.SideStories = []int{id}
			beforePlayer := g.player
			var ending bytes.Buffer
			g.renderEnding(&ending)
			if id == 9 && (!strings.Contains(ending.String(), "Mara's dream") ||
				strings.Contains(ending.String(), "Ash crosses the threshold") != alive) {
				t.Fatal("the collected dream's memory must reflect Ash's survival")
			}
			if id == 10 && (!strings.Contains(ending.String(), "final line") ||
				strings.Contains(ending.String(), "home is the hand") != alive) {
				t.Fatal("the collected song's memory must reflect Ash's survival")
			}
			if g.rng.State != beforeRNG || g.stats != beforeStats || !reflect.DeepEqual(beforePlayer, g.player) {
				t.Fatal("the ending must not change gameplay state")
			}
		}
	}
	var fresh bytes.Buffer
	g.player.SideStories = nil
	g.renderEnding(&fresh)
	if fresh.String() != "Beside the door, you see a clean bowl filled with fresh rain water.\n" {
		t.Fatal("an uncollected story must not contribute its ending")
	}
}

func TestEndingSelectsOneEligibleMemoryAndPreservesItAcrossSaveLoad(t *testing.T) {
	eligible := map[string]bool{
		"You remember Mara's stolen glove, and hope there will be small, foolish mornings again.":            true,
		"Somewhere beyond the door, you imagine a weak bell ringing, and a dog answering louder than grief.": true,
		"Ash fell in the dark, and you brought him back. This time, he follows you into the light.":          true,
	}
	seen := map[string]bool{}
	for seed := int64(1); seed <= 128; seed++ {
		g := openArena()
		g.rng = newSimpleRNG(seed)
		g.dog.Freed = true
		g.player.SideStories = []int{1, 6}
		g.stats.GhostDogFalls = 1
		rng, stats := g.rng.State, g.stats
		var first, repeated bytes.Buffer
		g.renderEnding(&first)
		g.renderEnding(&repeated)
		lines := strings.Split(strings.TrimSpace(first.String()), "\n")
		if len(lines) != 2 || !eligible[lines[1]] {
			t.Fatalf("the bowl must be followed by exactly one earned memory: %q", first.String())
		}
		seen[lines[1]] = true
		if first.String() != repeated.String() || g.rng.State != rng || g.stats != stats {
			t.Fatal("redrawing the ending must retain its selection without spending RNG or turns")
		}
		if seed == 1 {
			loaded := saveAndReloadAdventure(t, g)
			var restored bytes.Buffer
			loaded.renderEnding(&restored)
			if restored.String() != first.String() || loaded.rng.State != rng {
				t.Fatal("save/load must preserve the selected memory without advancing RNG")
			}
		}
	}
	if len(seen) != len(eligible) {
		t.Fatal("different run RNG states must be able to select every eligible memory")
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
