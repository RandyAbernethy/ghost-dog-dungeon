package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func trappedArena(kind trapKind) *game {
	g := cryptArena()
	g.inCrypt, g.crypt.Floor.Visited = true, true
	g.player.Pos = pos{5, 5}
	g.crypt.Traps = []cryptTrap{{Kind: kind, Pos: pos{6, 5}, Damage: 10}}
	return g
}

func TestAllEightCryptTrapsTriggerOnceThroughMovement(t *testing.T) {
	for _, kind := range cryptTrapKinds {
		t.Run(string(kind), func(t *testing.T) {
			g := trappedArena(kind)
			rng := g.rng.State
			g.processCommand("d")
			trap := g.crypt.Traps[0]
			if !trap.Spent || !trap.Discovered || g.player.HP >= 200 || g.stats.Turns != 1 || g.rng.State != rng {
				t.Fatal("stepping onto a trap must reveal it, spend it and apply its saved effect once")
			}
			switch kind {
			case trapPit:
				if g.player.BoundTurns != 1 || g.player.Binding != bindingPit {
					t.Fatal("a pit must hold the player until a movement action climbs out")
				}
				if !strings.Contains(strings.Join(g.messages, "\n"), "You struggle for a handhold on the pit's crumbling edge.") {
					t.Fatal("falling into a pit must describe the struggle to climb out")
				}
			case trapFrost:
				if g.player.Binding != bindingIce || g.player.BoundTurns != 1 {
					t.Fatal("frost must use the existing ice hold")
				}
			case trapBell:
				if g.player.HexedTurns != 1 {
					t.Fatal("a grave bell must weaken the next strike")
				}
			case trapGas:
				if g.player.PoisonTurns != 2 || g.player.HP != 198 {
					t.Fatal("gas must start three action turns of poison, including this action")
				}
				if !strings.Contains(strings.Join(g.messages, "\n"), "Choking gas surrounds you. Your lungs begin to burn.") {
					t.Fatal("gas must describe the inhaled poison")
				}
			case trapRepulsion:
				if g.player.Pos != (pos{5, 5}) {
					t.Fatal("repulsion must push the player back to the previous free tile")
				}
			}
			g = saveCryptAndReload(t, g)
			// Clear a movement hold, then revisit the spent trap.
			if g.player.BoundTurns > 0 {
				p := g.player.Pos
				g.processCommand("a")
				if g.player.Pos != p || g.player.BoundTurns != 0 || g.player.Binding != "" {
					t.Fatal("escaping the hold must cost exactly one movement turn")
				}
				if kind == trapPit && g.messages[0] != "You haul yourself out of the collapsed pit." {
					t.Fatal("escaping a pit must describe climbing out")
				}
			}
			g.player.PoisonTurns = 0
			g.player.Pos = pos{5, 5}
			before := g.player.HP
			g.processCommand("d")
			if g.player.HP != before || g.player.Pos != (pos{6, 5}) {
				t.Fatal("a spent trap cannot fire again after saving or revisiting")
			}
		})
	}
}

func TestCryptSearchDiscoveryChanceRangeAndSafeDisarming(t *testing.T) {
	found := 0
	for seed := int64(1); seed <= 1000; seed++ {
		g := trappedArena(trapBlock)
		g.rng = newSimpleRNG(seed)
		g.processCommand("k")
		if g.crypt.Traps[0].Discovered {
			found++
		}
		if g.crypt.Traps[0].Spent || g.stats.Turns != 1 || g.player.HP != 200 {
			t.Fatal("searching must cost one turn and cannot trigger or disarm a trap")
		}
	}
	if found < 475 || found > 525 {
		t.Fatalf("discovery should be a 50%% roll: %d/1000", found)
	}
	g := trappedArena(trapBlock)
	g.player.Pos = pos{20, 10}
	rng := g.rng.State
	g.processCommand("k")
	if g.crypt.Traps[0].Discovered || g.rng.State != rng {
		t.Fatal("searching farther than one tile must not roll discovery")
	}
	g.player.Pos = pos{5, 5}
	g.processCommand("y")
	if g.stats.Turns != 1 {
		t.Fatal("an undiscovered trap cannot be disarmed by a free interaction probe")
	}
	for attempts := 0; attempts < 30 && !g.crypt.Traps[0].Discovered; attempts++ {
		g.processCommand("k")
	}
	if !g.crypt.Traps[0].Discovered {
		t.Fatal("repeated adjacent searches should be able to reveal the trap")
	}
	g = saveCryptAndReload(t, g)
	rng = g.rng.State
	g.processCommand("k")
	if g.rng.State != rng {
		t.Fatal("a known trap must never reroll discovery")
	}
	before := g.stats.Turns
	g.processCommand("y")
	if !g.crypt.Traps[0].Spent || g.stats.Turns != before+1 || g.player.HP != 200 {
		t.Fatal("disarming a discovered trap must safely spend one turn")
	}
	g = saveCryptAndReload(t, g)
	g.processCommand("d")
	if g.player.HP != 200 {
		t.Fatal("disarmed traps must stay safe after loading")
	}
}

func TestCryptPoisonOnlyTicksOnActionsAndSurvivesTheExit(t *testing.T) {
	g := trappedArena(trapGas)
	g.processCommand("d")
	g = saveCryptAndReload(t, g)
	var out bytes.Buffer
	if err := g.loop(strings.NewReader("m\nbook\ni\nmap\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	g.quit = false
	if g.player.HP != 198 || g.player.PoisonTurns != 2 || g.stats.Turns != 1 {
		t.Fatal("reading screens must not advance poison")
	}
	g.crypt.ExitRevealed = true
	g.crypt.Floor.Tiles[g.crypt.Exit.Y][g.crypt.Exit.X] = '.'
	g.player.Pos = pos{35, 5}
	g.processCommand("d")
	if g.inCrypt || g.player.HP != 196 || g.player.PoisonTurns != 1 {
		t.Fatal("already inhaled poison must persist through the crypt exit")
	}
	g = saveCryptAndReload(t, g)
	g.processCommand(".")
	g.processCommand(".")
	if g.player.HP != 194 || g.player.PoisonTurns != 0 || g.stats.Turns != 4 {
		t.Fatal("the third and final poison tick must expire without renewal")
	}
}

func TestOracleCrownStopsMagicalTrapsButNotAPhysicalPit(t *testing.T) {
	for _, kind := range []trapKind{trapPit, trapFrost, trapBell} {
		g := trappedArena(kind)
		g.player.HeadArmor = godArmorPieces[1]
		g.processCommand("d")
		if kind == trapPit {
			if g.player.Binding != bindingPit {
				t.Fatal("an Oracle Crown cannot levitate the player out of a physical pit")
			}
			g.player.BodyArmor = armor{}
			g.player.carryArmor(armor{Name: "Replacement coat", Slot: slotBody, Defense: 1})
			g.processCommand(fmt.Sprintf("equip %d", len(g.player.Weapons)+len(g.player.Armors)))
			if g.player.Binding != bindingPit {
				t.Fatal("swapping armor while wearing a crown must not clear a physical pit hold")
			}
		} else if g.player.BoundTurns != 0 || g.player.HexedTurns != 0 {
			t.Fatal("Oracle Crown should prevent magical binding and hexes")
		}
	}
}
