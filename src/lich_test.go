package main

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func lichArena() (*game, *monster) {
	g := openArena()
	g.currentLevel = levelCount - 1
	g.dog.Freed, g.dog.Pos = true, pos{5, 4}
	m := newMonster(monsterBoss)
	m.Pos = pos{9, 5}
	g.current().Monsters = []*monster{&m}
	return g, &m
}

func chainedDogArena() *game {
	g, _ := lichArena()
	g.current().Monsters = nil
	g.dog.Chained, g.dog.ChainFloor = true, g.currentLevel
	return g
}

func TestLichChainHasExactlyATenPercentThreshold(t *testing.T) {
	seen := map[int]bool{}
	for seed := int64(1); seed <= 1000; seed++ {
		g, m := lichArena()
		g.rng = newSimpleRNG(seed)
		expected := newSimpleRNG(seed)
		roll := expected.Intn(100)
		seen[roll] = true
		cast := g.tryLichChain(m)
		if cast != (roll < 10) || g.dog.Chained != cast || g.rng.State != expected.State {
			t.Fatalf("seed %d roll %d: chaining must succeed below 10 and use exactly one roll", seed, roll)
		}
		if g.stats.Turns != 0 || g.dog.HP != g.dog.MaxHP || g.player.HP != g.player.MaxHP {
			t.Fatal("the chain itself must not add turns or deal damage")
		}
	}
	if !seen[9] || !seen[10] {
		t.Fatal("the sweep must cover both sides of the 10% boundary")
	}
}

func TestLichChainRequiresAnEligibleVisibleCompanion(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*game, *monster)
	}{
		{"unfreed", func(g *game, m *monster) { g.dog.Freed = false }},
		{"fallen", func(g *game, m *monster) { g.dog.Alive, g.dog.HP = false, 0 }},
		{"already chained", func(g *game, m *monster) { g.dog.Chained = true }},
		{"distant", func(g *game, m *monster) { m.Pos = pos{11, 5} }},
		{"wall", func(g *game, m *monster) { g.current().Tiles[4][7] = '#' }},
		{"dead Lich", func(g *game, m *monster) { m.HP = 0 }},
		{"other monster", func(g *game, m *monster) { m.Kind = monsterUnderLich }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, m := lichArena()
			g.rng = newSimpleRNG(5)
			tc.change(g, m)
			rng, dog := g.rng.State, g.dog
			if g.tryLichChain(m) || g.rng.State != rng || g.dog != dog {
				t.Fatal("an ineligible chain must not change the companion or draw RNG")
			}
		})
	}
	for _, effect := range []string{"stunned", "frozen", "fleeing"} {
		t.Run(effect, func(t *testing.T) {
			g, m := lichArena()
			g.rng = newSimpleRNG(5)
			switch effect {
			case "stunned":
				m.StunnedTurns = 1
			case "frozen":
				m.FrozenTurns = 1
			case "fleeing":
				m.FleeTurns = 1
			}
			rng := g.rng.State
			g.monsterTurn(m)
			if g.dog.Chained || g.rng.State != rng {
				t.Fatal("a disabled or fleeing Lich must not roll for a chain")
			}
		})
	}
}

func TestLichChainSpendsItsAttackAndPersists(t *testing.T) {
	g, m := lichArena()
	g.rng = newSimpleRNG(5)
	g.processCommand(".")
	if !g.dog.Chained || g.dog.ChainFloor != 9 || g.stats.Turns != 1 ||
		g.player.HP != g.player.MaxHP || g.dog.HP != g.dog.MaxHP || m.Pos != (pos{9, 5}) {
		t.Fatal("a successful chain must replace the Lich's attack and bind Ash on his current floor")
	}
	rng, dog := g.rng.State, g.dog
	var out bytes.Buffer
	g.render(&out)
	if symbolMapRows(t, out.String())[dog.Pos.Y][dog.Pos.X] != 'd' || !strings.Contains(strings.Join(strings.Fields(out.String()), " "), "Move onto his d tile") {
		t.Fatal("the map and chain message must show d and explain how to free Ash")
	}
	loaded := saveAndReloadAdventure(t, g)
	if loaded.dog != dog || loaded.rng.State != rng || loaded.stats != g.stats {
		t.Fatal("save/load must preserve the chain, its anchor floor, health, turns and RNG")
	}
}

func TestChainedGhostDogCannotAct(t *testing.T) {
	g := chainedDogArena()
	m := newMonster(monsterIronColossus)
	m.Pos, m.StrikeTiles = pos{6, 4}, []pos{g.dog.Pos}
	g.current().Monsters = []*monster{&m}
	g.dogFocus = &m
	rng, dog, hp := g.rng.State, g.dog, m.HP
	g.dogTurn()
	g.dogAttack(&m)
	if g.dodgeEpicWarning() || g.placeDogNearPlayer() || g.dog != dog || m.HP != hp || g.rng.State != rng {
		t.Fatal("chained Ghost Dog must neither attack, move, dodge nor spend RNG")
	}
	g.player.Pos = pos{3, 5}
	attacker := newMonster(monsterOrc)
	attacker.Pos, attacker.MinDamage, attacker.MaxDamage = pos{6, 4}, 2, 2
	g.current().Monsters = []*monster{&attacker}
	g.monsterTurn(&attacker)
	if !g.dog.Chained || g.dog.Pos != dog.Pos || g.dog.HP != dog.HP-1 {
		t.Fatal("a chained companion must remain vulnerable without moving or retaliating")
	}
}

func TestWalkingOntoChainedDogFreesHimBeforeAReachAttack(t *testing.T) {
	g := chainedDogArena()
	g.player.Weapon.Reach = 2
	g.dog.HP = 12
	g.adventure.RescueFloor = 2
	m := newMonster(monsterOrc)
	m.Pos = pos{5, 3}
	g.current().Monsters = []*monster{&m}
	if !g.tryMove("w") || g.dog.Chained || g.dog.ChainFloor != 0 || g.player.Pos != (pos{5, 4}) ||
		g.dog.Pos != (pos{5, 5}) || m.HP != m.MaxHP || g.dog.HP != 12 || g.adventure.RescueFloor != 2 {
		t.Fatal("walking onto d must free Ash without a reach attack, healing, or repeating the original rescue")
	}
	g = chainedDogArena()
	g.processCommand("w")
	if g.dog.Chained || g.stats.Turns != 1 || g.dog.Pos == g.player.Pos ||
		!strings.HasPrefix(g.messages[0], "You move north and break the silver chain") {
		t.Fatal("freeing Ash must spend one movement turn and lead the turn's log")
	}
}

func TestChainedDogCannotFollowStairsBlinkOrRecall(t *testing.T) {
	g := chainedDogArena()
	dog := g.dog
	if !g.tryAscend() || g.currentLevel != 8 || g.dogPresent() || g.dog != dog || !g.isWalkable(dog.Pos, false) {
		t.Fatal("Ash must stay anchored when the player leaves his floor")
	}
	if !g.tryMove("w") || g.dog != dog {
		t.Fatal("walking through matching coordinates on another floor must not free Ash")
	}
	g.player.Pos = g.current().Stairs
	if !g.tryDescend() || !g.dogPresent() || g.dog != dog {
		t.Fatal("returning to the anchor floor must find Ash at the same chained position and health")
	}
	g.current().Monsters = nil
	g.player.BlinkStones = 1
	g.processCommand("g")
	if g.dog != dog || g.player.Pos == dog.Pos || g.player.BlinkStones != 0 || g.stats.Turns != 1 {
		t.Fatal("blink must move only the player and cannot land on or free chained Ash")
	}
	g.player.GhostRecallScrolls = 1
	rng := g.rng.State
	if g.useGhostRecallScroll() || g.player.GhostRecallScrolls != 1 || g.dog != dog || g.rng.State != rng {
		t.Fatal("recall cannot release a living chained companion")
	}
}

func TestLichDeathDoesNotReleaseALivingChainedDog(t *testing.T) {
	g, m := lichArena()
	g.dog.Chained, g.dog.ChainFloor = true, 9
	dog := g.dog
	g.damageMonster(m, m.HP, "hits")
	if g.dog != dog {
		t.Fatal("living Ash must remain chained after the Lich dies until the player reaches him")
	}
	g.player.GhostRecallScrolls = 1
	g.hurtDog(g.dog.HP, "A test blow")
	if g.dog.Alive || g.dog.Chained || g.dog.ChainFloor != 0 || g.stats.GhostDogFalls != 1 {
		t.Fatal("dispersion must clear the physical chain and count one fall")
	}
	if !g.useGhostRecallScroll() || !g.dog.active() || g.dog.HP != g.dog.MaxHP {
		t.Fatal("explicit revival must preserve the existing living-companion rules")
	}
}

func TestLeavingChainedDogBehindIsNotASafeEscape(t *testing.T) {
	g := chainedDogArena()
	g.current().HasEscapeStair, g.current().EscapeStairs = true, g.player.Pos
	g.current().Secrets = []*secretRoom{{Revealed: true, Tiles: []pos{g.player.Pos}}}
	g.recordsFile = filepath.Join(t.TempDir(), "records.json")
	var out bytes.Buffer
	if err := g.loop(strings.NewReader("<\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !g.won || !strings.Contains(out.String(), "You escaped the dungeon. Victory!") ||
		strings.Contains(out.String(), "with your ghost dog. Victory!") || strings.Contains(out.String(), "Together, you and Ash") {
		t.Fatal("the ending must not claim chained Ash escaped with the player")
	}
	records, err := readChallengeRecords(g.recordsFile)
	if err != nil || records.SafeEscape != nil || records.FewestTurns == nil {
		t.Fatal("an escape leaving Ash chained must still submit records but cannot earn a safe companion escape")
	}
}

func TestLichChainThroughExecutableAndSaveLoad(t *testing.T) {
	g, _ := lichArena()
	g.rng = newSimpleRNG(5)
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader(".\nsave\nload\nm\nc\ni\nbook\nmap\nsave\nw\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executable: %v\n%s", err, output)
	}
	paths := executableSavePathsForChain(t, string(output))
	first, err := loadGame(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	read, err := loadGame(paths[1])
	if err != nil {
		t.Fatal(err)
	}
	freed, err := loadGame(paths[2])
	if err != nil {
		t.Fatal(err)
	}
	if !first.dog.Chained || first.dog.ChainFloor != 9 || first.stats.Turns != 1 ||
		first.dog != read.dog || first.stats != read.stats || first.rng.State != read.rng.State ||
		!reflect.DeepEqual(first.levels, read.levels) || freed.dog.Chained || freed.stats.Turns != 2 {
		t.Fatal("the executable must retain chains across free screens and save/load, then release Ash on one movement turn")
	}
	for _, want := range []string{"binds Ghost Dog with a silver chain", "chained on floor 10", "Ghost Dog is chained (d)", "break the silver chain holding Ghost Dog"} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("the executable must explain chaining and rescue: missing %q", want)
		}
	}
}

func executableSavePathsForChain(t *testing.T, output string) []string {
	t.Helper()
	var paths []string
	for _, line := range strings.Split(output, "\n") {
		if _, path, ok := strings.Cut(line, "Game saved to "); ok {
			paths = append(paths, path)
		}
	}
	if len(paths) != 3 {
		t.Fatalf("expected three chain checkpoints, got %d: %s", len(paths), output)
	}
	return paths
}

func TestChainedDogOnAnotherFloorCannotReceiveHealing(t *testing.T) {
	g := chainedDogArena()
	g.dog.HP = 10
	g.currentLevel--
	g.current().Fountains = []fountain{{Pos: g.player.Pos}}
	g.current().Events = []floorEvent{{Kind: eventShrine, Pos: pos{6, 5}}}
	rng, dog := g.rng.State, g.dog
	for _, cmd := range []string{"v", "y"} {
		g.processCommand(cmd)
	}
	if g.dog != dog || g.stats.Turns != 0 || g.rng.State != rng || g.current().Fountains[0].Used || g.current().Events[0].Used {
		t.Fatal("healing on another floor must not affect chained Ash or consume an unusable blessing")
	}
	var out bytes.Buffer
	g.render(&out)
	if symbolMapRows(t, out.String())[dog.Pos.Y][dog.Pos.X] == 'd' {
		t.Fatal("chained Ash must not appear at matching coordinates on another floor")
	}
}

func TestChainedDogSaveRejectsAnInvalidAnchor(t *testing.T) {
	g := chainedDogArena()
	g.dog.ChainFloor = levelCount
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	if _, err := loadGame(g.saveFile); err == nil || !strings.Contains(err.Error(), "anchor floor") {
		t.Fatalf("invalid anchor should produce a useful load error, got %v", err)
	}
}

func TestLichChainDoesNotRollOnFailedOrFreeActions(t *testing.T) {
	g, _ := lichArena()
	g.rng = newSimpleRNG(5)
	g.player.Potions = 0
	g.current().Tiles[5][4] = '#'
	rng, dog := g.rng.State, g.dog
	for _, cmd := range []string{"p", "a", "h"} {
		g.processCommand(cmd)
	}
	if g.dog != dog || g.rng.State != rng || g.stats.Turns != 0 {
		t.Fatal("failed moves, empty consumables and free help must not give the Lich a chain attempt")
	}
}
