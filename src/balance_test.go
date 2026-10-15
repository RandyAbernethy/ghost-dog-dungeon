package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestFountainsIncludeOrdinaryFloorsAndEverySecretRoom(t *testing.T) {
	for seed := int64(1); seed <= 30; seed++ {
		g := newGame(seed)
		ordinary := 0
		for _, lvl := range g.levels {
			secret := map[pos]bool{}
			for _, hidden := range lvl.secretRooms() {
				for _, p := range hidden.Tiles {
					secret[p] = true
				}
			}
			secretCount := 0
			seen := map[pos]bool{}
			for _, f := range lvl.Fountains {
				if seen[f.Pos] || g.itemAt(lvl, f.Pos) != nil || (lvl.HasEscapeStair && f.Pos == lvl.EscapeStairs) {
					t.Fatalf("seed %d floor %d fountain overlaps another feature", seed, lvl.Index+1)
				}
				seen[f.Pos] = true
				if secret[f.Pos] {
					secretCount++
				} else {
					ordinary++
					if lvl.Tiles[f.Pos.Y][f.Pos.X] == '#' {
						t.Fatal("ordinary fountains should be accessible")
					}
				}
			}
			if secretCount != len(lvl.secretRooms()) {
				t.Fatalf("seed %d floor %d has %d secret fountains", seed, lvl.Index+1, secretCount)
			}
			for _, hidden := range lvl.secretRooms() {
				count := 0
				for _, p := range hidden.Tiles {
					if g.fountainAt(lvl, p) != nil {
						count++
					}
				}
				if count != 1 {
					t.Fatalf("seed %d floor %d has %d fountains in one room", seed, lvl.Index+1, count)
				}
			}
		}
		if ordinary < 3 || ordinary > 5 {
			t.Fatalf("seed %d has %d ordinary fountains, want 3-5", seed, ordinary)
		}
	}
}

func TestSecretFountainStaysHiddenThenHealsAndRenews(t *testing.T) {
	g := openArena()
	lvl := g.current()
	lvl.HasStair, lvl.HasUpStair = false, false
	lvl.Secrets = []*secretRoom{{Door: pos{6, 5}, Tiles: []pos{{7, 5}, {8, 5}, {7, 6}, {8, 6}}}}
	lvl.Tiles[5][6] = '#'
	for _, p := range lvl.primarySecret().Tiles {
		lvl.Tiles[p.Y][p.X] = '#'
	}
	g.player.Pos, g.player.HP = pos{6, 4}, 20
	g.dog.Freed, g.dog.HP, g.dog.Pos = true, 1, pos{5, 4}
	g.addSecretFountain(lvl)
	g.addSecretFountain(lvl)
	if len(lvl.Fountains) != 1 {
		t.Fatal("secret fountains should not duplicate")
	}
	mapGlyph := func() byte {
		var out bytes.Buffer
		g.render(&out)
		var rows []string
		for _, line := range strings.Split(out.String(), "\n") {
			if len(line) == mapWidth && strings.HasPrefix(line, "#") {
				rows = append(rows, line)
			}
		}
		return rows[5][7]
	}
	var inspect bytes.Buffer
	g.renderInspect(&inspect)
	if mapGlyph() != '#' || strings.Contains(inspect.String(), "Fountain (") || strings.Contains(g.contextHint(), "healing fountain") || g.drinkFromFountain() {
		t.Fatal("an undiscovered secret fountain must not be visible or usable through the wall")
	}
	if !g.search() || mapGlyph() != 'F' || !g.drinkFromFountain() {
		t.Fatal("searching should reveal a usable fountain")
	}
	if g.dog.HP < 9 || g.dog.HP > 13 || g.player.HP < 24 || g.player.HP > 28 {
		t.Fatalf("wrong healing: player=%d dog=%d", g.player.HP, g.dog.HP)
	}
	if !lvl.Fountains[0].Used || g.drinkFromFountain() {
		t.Fatal("the fountain should be spent for the current visit")
	}
	lvl.HasStair = true
	g.player.Pos = lvl.Stairs
	if !g.tryDescend() || lvl.Fountains[0].Used || !g.tryAscend() {
		t.Fatal("leaving and returning should renew the secret fountain")
	}
	g.player.Pos = pos{6, 4}
	oldHP := g.dog.HP
	if !g.drinkFromFountain() || g.dog.HP < oldHP+8 || g.dog.HP > oldHP+12 {
		t.Fatal("the renewed fountain should heal Ghost Dog again")
	}
	lvl.Fountains[0].Used = false
	g.player.HP, g.dog.HP = g.player.MaxHP, g.dog.MaxHP-1
	if !g.drinkFromFountain() || g.dog.HP != g.dog.MaxHP {
		t.Fatal("extra Ghost Dog healing should still cap at his maximum health")
	}
}

func TestSavedFountainsKeepSpentStateWithoutRenewal(t *testing.T) {
	g := newGameWithSaveDir(15, t.TempDir())
	for _, lvl := range g.levels {
		for i := range lvl.Fountains {
			lvl.Fountains[i].Used = true
		}
	}
	beforeRNG := g.rng.State
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := loaded.save(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadGame(loaded.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.levels, loaded.levels) || !reflect.DeepEqual(g.levels, reloaded.levels) || reloaded.rng.State != beforeRNG {
		t.Fatal("saving and loading must preserve fountains and dungeon state without renewing or adding anything")
	}
}

func TestWeaponLootGetsSmallBoostWithoutGrowingOnDrop(t *testing.T) {
	g := openArena()
	if g.player.Weapon.Max != 7 {
		t.Fatal("the starting knife should gain only one maximum damage")
	}
	var found item
	for _, it := range g.itemsForLevel(4) {
		if it.Kind == itemWeapon {
			found = it
		}
	}
	wantMax := map[string]int{"Dragontooth Pike": 11, "Starforged Hammer": 15, "Ember Lance": 10,
		"Dragonbone Partisan": 10, "Bellringer Hammer": 16, "Cinder Pike": 9}
	if found.Weapon.Max != wantMax[found.Weapon.Name] {
		t.Fatalf("weapon boost should be +1 maximum damage: %+v", found.Weapon)
	}
	found.Pos = g.player.Pos
	g.current().Items = []item{found}
	g.collectItems()
	for i := 0; i < 3; i++ {
		g.processCommand("drop 2")
		g.processCommand("d")
		g.processCommand("a")
		if len(g.player.Weapons) != 2 || g.player.Weapons[1] != found.Weapon {
			t.Fatal("dropping and recovering a weapon must not apply its boost again")
		}
	}
}
