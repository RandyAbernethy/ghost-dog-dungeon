package main

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func blinkArena(open ...pos) *game {
	g := openArena()
	lvl := g.current()
	for y := range lvl.Tiles {
		for x := range lvl.Tiles[y] {
			lvl.Tiles[y][x] = '#'
		}
	}
	g.dog.Freed, g.dog.Pos = true, pos{5, 4}
	g.player.BlinkStones = 1
	for _, p := range append(open, g.player.Pos, g.dog.Pos) {
		lvl.Tiles[p.Y][p.X] = '.'
	}
	return g
}

func TestBlinkStoneAvoidsAllLivingMonstersAndBringsGhostDog(t *testing.T) {
	g := blinkArena(pos{3, 5}, pos{13, 5}, pos{8, 5}, pos{8, 4}, pos{15, 5}, pos{15, 4})
	left, right, dead, otherFloor := newMonster(monsterOrc), newMonster(monsterOrc), newMonster(monsterOrc), newMonster(monsterOrc)
	left.Pos, right.Pos = pos{3, 5}, pos{13, 5}
	left.FrozenTurns, right.FrozenTurns = 1, 1
	dead.Pos, dead.HP, otherFloor.Pos = pos{8, 5}, 0, pos{8, 4}
	g.current().Monsters = []*monster{&left, &right, &dead}
	g.levels[1].Monsters = []*monster{&otherFloor}
	g.dogFocus = &left
	g.processCommand("g")
	if (g.player.Pos != (pos{8, 5}) && g.player.Pos != (pos{8, 4})) || (g.dog.Pos != (pos{8, 5}) && g.dog.Pos != (pos{8, 4})) || g.player.Pos == g.dog.Pos {
		t.Fatalf("expected the pair five tiles from both threats; player=%+v dog=%+v", g.player.Pos, g.dog.Pos)
	}
	if g.currentLevel != 0 || g.player.BlinkStones != 0 || g.stats.Turns != 1 || g.dogFocus != nil {
		t.Fatal("blinking should spend one stone and one turn on this floor and leave the old fight behind")
	}
}

func TestBlinkStoneExcludesSecretRoomsAfterDiscovery(t *testing.T) {
	for _, revealed := range []int{0, 1, 2} {
		g := blinkArena(pos{3, 5}, pos{10, 5}, pos{10, 4})
		foe := newMonster(monsterOrc)
		foe.Pos = pos{3, 5}
		g.current().Monsters = []*monster{&foe}
		hidden := &secretRoom{
			Door:  pos{24, 8},
			Tiles: []pos{{25, 8}, {25, 9}, {26, 8}, {26, 9}, {27, 8}, {27, 9}},
			Inner: &secretRoom{Door: pos{26, 8}, Tiles: []pos{{27, 8}, {27, 9}}},
		}
		g.current().Secret = hidden
		if revealed > 0 {
			revealRoom(g.current(), hidden)
		}
		if revealed > 1 {
			revealRoom(g.current(), hidden.Inner)
		}
		if !g.useBlinkStone() {
			t.Fatal("an ordinary landing area is available")
		}
		if (g.player.Pos != (pos{10, 5}) && g.player.Pos != (pos{10, 4})) || (g.dog.Pos != (pos{10, 5}) && g.dog.Pos != (pos{10, 4})) || g.player.Pos == g.dog.Pos {
			t.Fatalf("secret rooms and their entrances must remain excluded; rooms revealed=%d player=%+v dog=%+v", revealed, g.player.Pos, g.dog.Pos)
		}
	}
}

func TestBlinkStoneReservesSpaceForBothCompanions(t *testing.T) {
	g := blinkArena(pos{3, 5}, pos{10, 10}, pos{11, 10}, pos{20, 10})
	foe := newMonster(monsterOrc)
	foe.Pos, foe.FrozenTurns = pos{3, 5}, 1
	g.current().Monsters = []*monster{&foe}
	g.current().Items = []item{{Pos: pos{11, 10}, Kind: itemPotion}}
	potions := g.player.Potions
	g.processCommand("g")
	if g.player.Pos != (pos{11, 10}) || g.dog.Pos != (pos{10, 10}) {
		t.Fatalf("the farthest isolated tile cannot fit both companions; player=%+v dog=%+v", g.player.Pos, g.dog.Pos)
	}
	if g.player.Potions != potions+1 || len(g.current().Items) != 0 || g.stats.Turns != 1 {
		t.Fatal("a successful blink should still collect landing items and spend one turn")
	}
}

func TestBlinkStoneDoesNotReviveOrFreeGhostDog(t *testing.T) {
	for _, fallen := range []bool{false, true} {
		g := blinkArena(pos{3, 5}, pos{10, 10}, pos{11, 10}, pos{20, 10})
		foe := newMonster(monsterOrc)
		foe.Pos = pos{3, 5}
		g.current().Monsters = []*monster{&foe}
		if fallen {
			g.dog.Alive, g.dog.HP = false, 0
		} else {
			g.dog.Freed = false
			chain := g.dog.Pos
			g.current().DogChain = &chain
		}
		before := g.dog
		if !g.useBlinkStone() || g.player.Pos != (pos{20, 10}) {
			t.Fatalf("a player without an active companion should use the farthest solo landing; fallen=%t player=%+v", fallen, g.player.Pos)
		}
		if g.dog != before {
			t.Fatal("a blink stone should not move, free, or revive an unavailable Ghost Dog")
		}
	}
}

func TestBlinkStoneWorksOnAClearedFloor(t *testing.T) {
	g := blinkArena(pos{10, 10}, pos{11, 10})
	before := g.player.Pos
	g.processCommand("g")
	if g.player.Pos == before || distance(g.player.Pos, g.dog.Pos) != 1 || g.player.BlinkStones != 0 || g.stats.Turns != 1 {
		t.Fatal("even without monsters, a blink should move the companions together and spend one turn")
	}
}

func TestBlinkStoneFailurePreservesTheStoneAndTurn(t *testing.T) {
	for _, noStone := range []bool{false, true} {
		g := blinkArena(pos{10, 10})
		g.current().Secret = &secretRoom{Door: pos{5, 3}, Tiles: []pos{g.dog.Pos}, Revealed: true}
		if noStone {
			g.player.BlinkStones = 0
		}
		playerBefore, dogBefore := g.player, g.dog
		g.processCommand("g")
		if !reflect.DeepEqual(g.player, playerBefore) || g.dog != dogBefore || g.stats.Turns != 0 {
			t.Fatal("an unavailable stone or landing must leave positions, supplies, and turns unchanged")
		}
	}
}

func TestRunBlinkStone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blink.json")
	g := blinkArena(pos{3, 5}, pos{10, 10}, pos{11, 10})
	g.saveFile, g.timestampedSaves = path, false
	foe := newMonster(monsterOrc)
	foe.Pos, foe.FrozenTurns = pos{3, 5}, 1
	g.current().Monsters = []*monster{&foe}
	g.current().Secret = &secretRoom{Door: pos{30, 10}, Tiles: []pos{{31, 10}, {31, 11}}}
	revealRoom(g.current(), g.current().Secret)
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := []string{"--load-file", path, "--records-file", filepath.Join(dir, "records.json")}
	if err := run(args, strings.NewReader("g\nsave\nquit\n"), &stdout, &stderr); err != nil {
		t.Fatalf("run blink command: %v\n%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Space folds and carries you and Ghost Dog to the safest open spot on this floor.") {
		t.Fatal("the game should report the companions teleporting together")
	}
	loaded, err := loadGame(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.player.Pos != (pos{11, 10}) || loaded.dog.Pos != (pos{10, 10}) || loaded.player.BlinkStones != 0 || loaded.stats.Turns != 1 {
		t.Fatal("the blink command should move both companions to the safest ordinary tiles and persist the result")
	}
}
