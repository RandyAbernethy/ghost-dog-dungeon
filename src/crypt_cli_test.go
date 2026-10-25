package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestGeneratedCryptThroughExecutableAndSaveLoad(t *testing.T) {
	gameBinary(t)
	g := newGame(6)
	g.saveDir = t.TempDir()
	c := g.crypt
	g.currentLevel = c.ParentFloor
	g.current().Monsters = nil
	for _, d := range cardinalDirections {
		p := pos{c.Entrance.X + d.X, c.Entrance.Y + d.Y}
		if g.current().Tiles[p.Y][p.X] != '#' {
			g.player.Pos = p
			break
		}
	}
	// This arena isolates travel, fog, traps and persistence from balance tuning.
	g.player.HP, g.player.MaxHP = 2000, 2000
	g.player.Weapon = weapon{Name: "Test exorcist blade", Min: 1000, Max: 1000, Magic: true}
	g.player.ensureCarriedEquipment()
	g.player.BlinkStones, g.player.WardingCharms = 1, 1
	g.dog.Freed, g.dog.HP, g.dog.MaxHP = true, 2000, 2000
	g.placeDogNearPlayer()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	initial := g.saveFile
	var commands []string
	act := func(cmd string) {
		commands = append(commands, cmd)
		g.processCommand(cmd)
		g.noteVisibleMonsters()
	}
	freeCheckpoint := func() { commands = append(commands, "save", "load", "m", "book", "map") }
	act("k")
	freeCheckpoint()
	act(moveKeyToward(g.player.Pos, c.Entrance))
	if !g.inCrypt {
		t.Fatal("fixture failed to enter the crypt")
	}
	act("g")
	act("b")
	freeCheckpoint()
	target := pos{c.Exit.X - 1, c.Exit.Y}
	for steps := 0; steps < 1000 && g.player.Pos != target && g.player.HP > 0; steps++ {
		cmd := cryptPathCommand(g, target)
		if cmd == "" {
			t.Fatal("generated maze has no path to the far door")
		}
		act(cmd)
		if steps == 10 {
			freeCheckpoint()
		}
	}
	if g.player.Pos != target {
		t.Fatal("the generated-maze replay did not reach the far side")
	}
	act("k")
	freeCheckpoint()
	act("d")
	freeCheckpoint()
	commands = append(commands, "save", "quit")
	if g.inCrypt || !c.Completed || !g.bossAlive(g.levels[9]) {
		t.Fatal("the crypt must return to the campaign without finishing it")
	}
	cmd := gameCommand(t, "--load-file", initial, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader(strings.Join(commands, "\n") + "\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("crypt executable: %v\n%s", err, output)
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.player, loaded.player) || g.dog != loaded.dog || g.stats != loaded.stats || g.rng.State != loaded.rng.State || !reflect.DeepEqual(g.levels, loaded.levels) || !reflect.DeepEqual(g.crypt, loaded.crypt) || loaded.inCrypt {
		t.Fatal("compiled save/load traversal must match uninterrupted play through the entire generated crypt")
	}
	prose := strings.Join(strings.Fields(stripANSI(string(output))), " ")
	for _, want := range []string{cryptName, "entrance seals behind you", "no ward forms", "space refuses to fold", "Under Lich falls.", "return to floor"} {
		if !strings.Contains(prose, want) {
			t.Fatalf("compiled crypt did not show %q", want)
		}
	}
	if strings.Contains(string(output), "Victory!") || strings.Contains(string(output), "NEW RECORD!") {
		t.Fatal("leaving the optional crypt must not finish the run or submit records")
	}
}

func TestCryptTrapDiscoveryDisarmingAndPoisonThroughExecutable(t *testing.T) {
	for _, disarm := range []bool{false, true} {
		g := trappedArena(trapGas)
		g.rng = newSimpleRNG(2) // The first adjacent search succeeds.
		g.saveDir = t.TempDir()
		if err := g.save(); err != nil {
			t.Fatal(err)
		}
		actions := "d\nsave\nload\nm\n.\n.\n"
		if disarm {
			actions = "k\nsave\nload\nm\ny\nd\n"
		}
		cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
		cmd.Stdin = strings.NewReader(actions + "save\nquit\n")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("trap executable: %v\n%s", err, output)
		}
		loaded, err := loadGame(lastSavePath(t, string(output)))
		if err != nil {
			t.Fatal(err)
		}
		wantHP := 194
		if disarm {
			wantHP = 200
		}
		if loaded.player.HP != wantHP || loaded.player.PoisonTurns != 0 || loaded.stats.Turns != 3 || !loaded.crypt.Traps[0].Spent {
			t.Fatalf("disarm=%v: saved trap state and poison turns must follow command-loop actions", disarm)
		}
	}
}

func moveKeyToward(from, to pos) string {
	dx, dy := sign(to.X-from.X), sign(to.Y-from.Y)
	for _, key := range []string{"w", "a", "s", "d", "q", "e", "z", "x"} {
		x, y, _ := moveDelta(key)
		if dx == x && dy == y {
			return key
		}
	}
	return ""
}

func cryptPathCommand(g *game, target pos) string {
	type node struct {
		p     pos
		first string
	}
	queue := []node{{g.player.Pos, ""}}
	seen := map[pos]bool{g.player.Pos: true}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.p == target {
			return cur.first
		}
		for _, key := range []string{"w", "a", "s", "d", "q", "e", "z", "x"} {
			dx, dy, _ := moveDelta(key)
			next := pos{cur.p.X + dx, cur.p.Y + dy}
			if !g.inBounds(next) || g.current().Tiles[next.Y][next.X] == '#' || seen[next] {
				continue
			}
			first := cur.first
			if first == "" {
				first = key
			}
			seen[next] = true
			queue = append(queue, node{next, first})
		}
	}
	return ""
}
