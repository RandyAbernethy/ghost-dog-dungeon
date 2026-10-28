package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestBarrowArcherShotsRequireClearFloorLines(t *testing.T) {
	for _, direction := range []struct {
		name   string
		offset pos
	}{
		{"east", pos{4, 0}}, {"west", pos{-4, 0}},
		{"south", pos{0, 4}}, {"north", pos{0, -4}},
		{"southeast", pos{4, 4}}, {"southwest", pos{-4, 4}},
		{"northeast", pos{4, -4}}, {"northwest", pos{-4, -4}},
		{"shallow angle", pos{4, 2}}, {"steep angle", pos{2, 4}},
	} {
		for _, blocked := range []bool{false, true} {
			name := direction.name + "/clear"
			if blocked {
				name = direction.name + "/wall"
			}
			t.Run(name, func(t *testing.T) {
				shots := 0
				for seed := int64(1); seed <= 32; seed++ {
					g := openArena()
					g.rng = newSimpleRNG(seed)
					m := newMonster(monsterBarrowArcher)
					m.Pos = pos{g.player.Pos.X + direction.offset.X, g.player.Pos.Y + direction.offset.Y}
					g.current().Monsters = []*monster{&m}
					if blocked {
						wall := pos{g.player.Pos.X + direction.offset.X/2, g.player.Pos.Y + direction.offset.Y/2}
						g.current().Tiles[wall.Y][wall.X] = '#'
					}
					playerHP, dogHP, rng := g.player.HP, g.dog.HP, g.rng.State
					g.monsterTurn(&m)
					fired := strings.Contains(strings.Join(g.messages, "\n"), "fires a barbed arrow")
					if fired {
						shots++
						if g.player.HP >= playerHP {
							t.Fatalf("seed %d: a clear shot must retain physical arrow damage", seed)
						}
					}
					if blocked && (fired || g.player.HP != playerHP || g.dog.HP != dogHP || g.rng.State != rng) {
						t.Fatalf("seed %d: a blocked archer must not fire, damage either companion or roll an attack", seed)
					}
				}
				if !blocked && shots == 0 {
					t.Fatal("clear floor lines must still permit arrows")
				}
			})
		}
	}
}

func TestBarrowArcherRangeAndOffLineTerrain(t *testing.T) {
	for _, tc := range []struct {
		name    string
		archer  pos
		terrain pos
		tile    rune
		fires   bool
	}{
		{"five-tile range", pos{10, 5}, pos{7, 4}, '#', true},
		{"outside range", pos{11, 5}, pos{7, 4}, '#', false},
		{"wall beside shot", pos{9, 5}, pos{7, 4}, '#', true},
		{"non-floor in shot", pos{9, 5}, pos{7, 5}, ' ', false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.rng = newSimpleRNG(2) // The first roll fires when the target is eligible.
			m := newMonster(monsterBarrowArcher)
			m.Pos = tc.archer
			g.current().Monsters = []*monster{&m}
			g.current().Tiles[tc.terrain.Y][tc.terrain.X] = tc.tile
			hp := g.player.HP
			g.monsterTurn(&m)
			fired := strings.Contains(strings.Join(g.messages, "\n"), "fires a barbed arrow")
			if fired != tc.fires || (g.player.HP < hp) != tc.fires {
				t.Fatalf("fired=%t HP=%d, want arrow damage=%t", fired, g.player.HP, tc.fires)
			}
		})
	}
}

func TestBarrowArcherCannotHurtAshWithoutSeeingPlayer(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		g := openArena()
		g.dog.Freed, g.dog.Pos = true, pos{8, 5}
		m := newMonster(monsterBarrowArcher)
		m.Pos = pos{9, 5}
		g.current().Monsters = []*monster{&m}
		if blocked {
			g.current().Tiles[5][7] = '#'
		}
		playerHP, dogHP, rng := g.player.HP, g.dog.HP, g.rng.State
		g.monsterTurn(&m)
		if blocked {
			if g.player.HP != playerHP || g.dog.HP != dogHP || g.rng.State != rng {
				t.Fatal("an archer behind a wall must not attack nearby Ash or roll damage")
			}
		} else if g.dog.HP >= dogHP || g.player.HP != playerHP {
			t.Fatal("seeing the player must retain ordinary adjacent-companion targeting")
		}
	}
}

func TestBarrowArcherUsesActiveCryptTerrain(t *testing.T) {
	g := cryptArena()
	g.inCrypt = true
	g.rng = newSimpleRNG(2)
	m := newMonster(monsterBarrowArcher)
	m.Pos = pos{7, 5}
	g.current().Monsters = []*monster{&m}
	// The parent floor has a wall at (6,5); the active crypt has clear floor.
	hp := g.player.HP
	g.monsterTurn(&m)
	if g.player.HP >= hp || !strings.Contains(strings.Join(g.messages, "\n"), "fires a barbed arrow") {
		t.Fatal("archer sight must use the active crypt instead of the entrance floor")
	}
}

func TestBlockedBarrowArcherThroughExecutableAndSaveLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("compiled archer continuation")
	}
	g := openArena()
	g.saveDir = t.TempDir()
	g.dog.Freed, g.dog.Chained, g.dog.Pos = true, true, pos{8, 5}
	m := newMonster(monsterBarrowArcher)
	m.Pos = pos{9, 5}
	g.current().Monsters = []*monster{&m}
	for y := 1; y < mapHeight-1; y++ {
		g.current().Tiles[y][7] = '#'
	}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader(".\nsave\nload\n.\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("archer executable: %v\n%s", err, output)
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.player.HP != g.player.HP || loaded.dog.HP != g.dog.HP || loaded.stats.Turns != 2 || loaded.rng.State != g.rng.State || strings.Contains(string(output), "fires a barbed arrow") || strings.Contains(string(output), "Barrow Archer claws") {
		t.Fatal("walls must protect both companions across command-loop turns and save/load")
	}
	g.processCommand(".")
	g.processCommand(".")
	if !reflect.DeepEqual(loaded.levels, g.levels) || !reflect.DeepEqual(loaded.player, g.player) || loaded.dog != g.dog {
		t.Fatal("blocked archer movement and companion state must persist exactly")
	}
}
