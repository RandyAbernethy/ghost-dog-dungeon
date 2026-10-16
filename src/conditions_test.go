package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestMonsterBindingsNameTheirCauseAndCostOneMovementTurn(t *testing.T) {
	for _, tc := range []struct {
		name    string
		kind    monsterKind
		spot    pos
		binding bindingKind
		message string
	}{
		{"spider silk", monsterSpider, pos{8, 5}, bindingWebs, "Sticky webs hold you in place for a turn."},
		{"crawler silk", monsterCaveCrawler, pos{8, 5}, bindingWebs, "Sticky webs hold you in place for a turn."},
		{"spider bite", monsterSpider, pos{6, 5}, bindingWebs, "Sticky webs hold you in place for a turn."},
		{"crawler bite", monsterCaveCrawler, pos{6, 5}, bindingWebs, "Sticky webs hold you in place for a turn."},
		{"frost hound", monsterFrostHound, pos{6, 5}, bindingIce, "Ice locks your feet in place for a turn."},
		{"chain imp", monsterChainImp, pos{8, 5}, bindingChains, "Silver chains pin your ankles in place for a turn."},
		{"rift weaver", monsterRiftWeaver, pos{8, 5}, bindingRift, "Rift magic binds your feet in place for a turn."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for seed := int64(1); seed <= 100; seed++ {
				g := varietyArena(seed)
				m := newMonster(tc.kind)
				m.Pos = tc.spot
				g.current().Monsters = []*monster{&m}
				g.monsterTurn(&m)
				if g.player.BoundTurns == 0 {
					continue
				}
				if g.player.Binding != tc.binding || g.player.BoundTurns != 1 {
					t.Fatalf("%s applied the wrong movement effect: %+v", tc.name, g.player)
				}
				beforePlayer, beforeStats, beforeRNG := g.player, g.stats, g.rng.State
				var out bytes.Buffer
				g.render(&out)
				if !strings.Contains(out.String(), tc.binding.label()+" 1") || !reflect.DeepEqual(g.player, beforePlayer) || g.stats != beforeStats || g.rng.State != beforeRNG {
					t.Fatal("rendering must name the movement effect without changing gameplay")
				}
				// Isolate the hold's movement cost from another attack or reapplication.
				m.FrozenTurns = 2
				command, destination := "a", (pos{4, 5})
				if m.Pos == destination {
					command, destination = "w", (pos{5, 4})
				}
				g.processCommand(command)
				if g.player.Pos != beforePlayer.Pos || g.stats.Turns != 1 || g.player.BoundTurns != 0 || g.player.Binding != "" || g.messages[0] != tc.message {
					t.Fatalf("blocked movement should name its cause and consume exactly one turn: %v", g.messages)
				}
				g.processCommand(command)
				if g.player.Pos != destination || g.stats.Turns != 2 {
					t.Fatal("the following movement should succeed after the hold expires")
				}
				return
			}
			t.Fatal("never observed the advertised movement effect")
		})
	}
}

func TestOracleCrownPreventsAndClearsEveryBinding(t *testing.T) {
	for _, kind := range []bindingKind{bindingWebs, bindingIce, bindingChains, bindingRift} {
		t.Run(string(kind), func(t *testing.T) {
			g := openArena()
			g.player.HeadArmor = godArmorPieces[1]
			g.bindPlayer(kind)
			if g.player.BoundTurns != 0 || g.player.Binding != "" {
				t.Fatal("an equipped Oracle Crown should prevent every movement binding")
			}
			g.player.HeadArmor = armor{}
			g.bindPlayer(kind)
			if g.player.BoundTurns != 1 || g.player.Binding != kind {
				t.Fatal("removing the crown should restore the usual binding behavior")
			}
			g.hexPlayer()
			g.player.carryArmor(godArmorPieces[1])
			g.processCommand(fmt.Sprintf("equip %d", len(g.player.Weapons)+len(g.player.Armors)))
			if g.player.BoundTurns != 0 || g.player.Binding != "" || g.player.HexedTurns != 0 || g.stats.Turns != 1 {
				t.Fatal("equipping the crown must clear the binding and curse without an extra turn")
			}
		})
	}
}

func TestBindingCauseSurvivesSaveLoadAndFreeCommands(t *testing.T) {
	for _, kind := range []bindingKind{bindingWebs, bindingIce, bindingChains, bindingRift} {
		t.Run(string(kind), func(t *testing.T) {
			g := openArena()
			g.saveDir = t.TempDir()
			g.bindPlayer(kind)
			g.processCommand("save")
			loaded, err := loadGame(g.saveFile)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.player.Binding != kind || loaded.player.BoundTurns != 1 || loaded.rng.State != g.rng.State {
				t.Fatal("save/load must preserve the binding's cause and remaining movement penalty")
			}
			beforePlayer, beforeStats, beforeRNG := loaded.player, loaded.stats, loaded.rng.State
			var out bytes.Buffer
			if err := loaded.loop(strings.NewReader("i\n\x1b\nh\nnot-a-command\nquit\n"), &out); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(loaded.player, beforePlayer) || loaded.stats != beforeStats || loaded.rng.State != beforeRNG {
				t.Fatal("reading, Esc, help, and invalid commands must not consume a binding")
			}
		})
	}
}

func TestFrozenMovementThroughRealExecutableAndSavedGame(t *testing.T) {
	dir := t.TempDir()
	g := openArena()
	g.saveDir = dir
	g.bindPlayer(bindingIce)
	beforePos := g.player.Pos
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", dir)
	cmd.Env = testEnvironment(dir)
	cmd.Stdin = strings.NewReader("i\n\x1b\nd\nsave\nload\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "frozen 1") || !strings.Contains(string(output), "Ice locks your feet in place for a turn.") || strings.Contains(string(output), "Sticky webs hold") {
		t.Fatal("the frozen player's HUD and attempted move must describe ice rather than webs")
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.player.Pos != beforePos || loaded.stats.Turns != 1 || loaded.player.BoundTurns != 0 || loaded.player.Binding != "" {
		t.Fatal("the actual command loop must spend exactly one turn on the frozen move and persist its expiry")
	}
}
