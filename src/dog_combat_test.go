package main

import "testing"

func TestGhostDogHelpsAgainstAnAttackerAcrossThePlayer(t *testing.T) {
	for _, tc := range []struct {
		name string
		dog  pos
		foe  pos
	}{
		{"attacker south", pos{5, 4}, pos{5, 6}},
		{"attacker north", pos{5, 6}, pos{5, 4}},
		{"attacker east", pos{4, 5}, pos{6, 5}},
		{"attacker west", pos{6, 5}, pos{4, 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.dog.Freed, g.dog.Pos = true, tc.dog
			attacker := newMonster(monsterOrc)
			attacker.Pos, attacker.HP, attacker.MaxHP = tc.foe, 60, 60
			g.current().Monsters = []*monster{&attacker}
			g.attackPlayer(&attacker)
			if g.dogFocus != &attacker {
				t.Fatal("an unengaged Ghost Dog should focus the monster attacking the player")
			}
			g.processCommand(".")
			if attacker.HP >= 60 || distance(g.dog.Pos, attacker.Pos) != 1 {
				t.Fatalf("Ghost Dog did not come around to help: dog=%+v attacker=%+v hp=%d", g.dog.Pos, attacker.Pos, attacker.HP)
			}
			if g.dog.Pos == g.player.Pos || distance(g.dog.Pos, tc.dog) != 1 || distance(g.dog.Pos, g.player.Pos) > 2 || g.stats.Turns != 1 {
				t.Fatal("coming to help should use one nearby movement step during the normal companion turn")
			}
		})
	}
}

func TestGhostDogTakesADetourAroundBlockedSides(t *testing.T) {
	g := openArena()
	g.dog.Freed, g.dog.Pos = true, pos{5, 4}
	lvl := g.current()
	lvl.Tiles[5][4], lvl.Tiles[5][6] = '#', '#'
	attacker := newMonster(monsterOrc)
	attacker.Pos, attacker.HP, attacker.MaxHP = pos{5, 6}, 60, 60
	attacker.MinDamage, attacker.MaxDamage = 1, 1
	lvl.Monsters = []*monster{&attacker}
	g.attackPlayer(&attacker)
	for turn := 0; turn < 3 && attacker.HP == 60; turn++ {
		before := g.dog.Pos
		g.processCommand(".")
		if g.dog.Pos == g.player.Pos || lvl.Tiles[g.dog.Pos.Y][g.dog.Pos.X] == '#' || distance(g.dog.Pos, before) != 1 || distance(g.dog.Pos, g.player.Pos) > 2 {
			t.Fatal("Ghost Dog should take a valid nearby step around the obstruction")
		}
	}
	if attacker.HP >= 60 {
		t.Fatal("Ghost Dog should finish the detour and help attack")
	}
}

func TestGhostDogKeepsHisFightWhenAnotherMonsterAttacksThePlayer(t *testing.T) {
	for _, remembered := range []bool{false, true} {
		g := openArena()
		g.dog.Freed, g.dog.Pos = true, pos{5, 4}
		attacker, opponent := newMonster(monsterOrc), newMonster(monsterOrc)
		attacker.Pos, attacker.HP, attacker.MaxHP = pos{5, 6}, 60, 60
		opponent.Pos, opponent.HP, opponent.MaxHP = pos{5, 3}, 60, 60
		g.current().Monsters = []*monster{&attacker, &opponent}
		if remembered {
			g.dogFocus = &opponent
		}
		g.attackPlayer(&attacker)
		g.processCommand(".")
		if attacker.HP != 60 || opponent.HP >= 60 || g.dog.Pos != (pos{5, 4}) || g.dogFocus != &opponent {
			t.Fatalf("Ghost Dog should keep fighting his current opponent; remembered=%t, attacker hp=%d, opponent hp=%d, dog=%+v", remembered, attacker.HP, opponent.HP, g.dog.Pos)
		}
	}
}

func TestGhostDogCannotCrossWallsToHelp(t *testing.T) {
	g := openArena()
	g.dog.Freed, g.dog.Pos = true, pos{5, 4}
	for y := g.dog.Pos.Y - 1; y <= g.dog.Pos.Y+1; y++ {
		for x := g.dog.Pos.X - 1; x <= g.dog.Pos.X+1; x++ {
			p := pos{x, y}
			if p != g.dog.Pos && p != g.player.Pos {
				g.current().Tiles[y][x] = '#'
			}
		}
	}
	attacker := newMonster(monsterOrc)
	attacker.Pos, attacker.HP, attacker.MaxHP = pos{5, 6}, 60, 60
	g.current().Monsters = []*monster{&attacker}
	g.attackPlayer(&attacker)
	g.processCommand(".")
	if g.dog.Pos != (pos{5, 4}) || attacker.HP != 60 {
		t.Fatal("Ghost Dog must wait if walls and the player block every route")
	}
}
