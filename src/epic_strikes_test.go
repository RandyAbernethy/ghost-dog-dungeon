package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func warnedEpicArena(kind monsterKind) (*game, *monster) {
	g := openArena()
	g.player.HP, g.player.MaxHP = 200, 200
	m := newMonster(kind)
	m.Pos, m.ActionTurns, m.Awakened = pos{7, 5}, 2, true
	m.MinDamage, m.MaxDamage = 20, 20
	g.current().Monsters = []*monster{&m}
	return g, &m
}

func TestEpicWarningsGiveOnePlayerActionAndNeverRetarget(t *testing.T) {
	for _, kind := range epicMonsterKinds {
		t.Run(string(kind), func(t *testing.T) {
			g, m := warnedEpicArena(kind)
			beforeRNG := g.rng.State
			g.processCommand(".")
			if g.player.HP != 200 || m.ActionTurns != 3 || !containsPosition(m.StrikeTiles, g.player.Pos) || g.rng.State != beforeRNG {
				t.Fatal("preparing a signature must warn without damage or randomness")
			}
			var out bytes.Buffer
			g.render(&out)
			g.renderInspect(&out)
			if !strings.Contains(out.String(), "YOU are on a marked tile") || !strings.Contains(out.String(), epicStrikeName(kind)) || !strings.Contains(out.String(), "\x1b[1;31m") {
				t.Fatal("map and inspection must clearly show the warning")
			}
			// Try actual move commands, selecting an exit from the fixed pattern.
			commands := []struct {
				key string
				d   pos
			}{{"a", pos{-1, 0}}, {"w", pos{0, -1}}, {"q", pos{-1, -1}}, {"s", pos{0, 1}}}
			moved := false
			for _, move := range commands {
				next := pos{g.player.Pos.X + move.d.X, g.player.Pos.Y + move.d.Y}
				if !g.epicDangerAt(next) {
					g.processCommand(move.key)
					moved = true
					break
				}
			}
			if !moved || g.player.HP != 200 || g.stats.Turns != 2 || m.ActionTurns != 4 || len(m.StrikeTiles) != 0 || !strings.Contains(strings.Join(g.messages, "\n"), "strikes empty stone") {
				t.Fatal("a move off the marked tiles must dodge the next attack without retargeting")
			}
		})
	}
}

func TestEpicWarningHitDelayCancellationAndPersistence(t *testing.T) {
	for _, kind := range epicMonsterKinds {
		g, m := warnedEpicArena(kind)
		g.processCommand(".")
		tiles := append([]pos(nil), m.StrikeTiles...)
		g.saveDir = t.TempDir()
		if err := g.save(); err != nil {
			t.Fatal(err)
		}
		loaded, err := loadGame(g.saveFile)
		if err != nil {
			t.Fatal(err)
		}
		lm := loaded.current().Monsters[0]
		if !reflect.DeepEqual(lm.StrikeTiles, tiles) || loaded.rng.State != g.rng.State {
			t.Fatal("load must preserve exact warning tiles and RNG")
		}
		var out bytes.Buffer
		if err := loaded.loop(strings.NewReader("m\nbook\ni\nmap\nsave\nload\nquit\n"), &out); err != nil {
			t.Fatal(err)
		}
		lm = loaded.current().Monsters[0]
		if !reflect.DeepEqual(lm.StrikeTiles, tiles) || lm.ActionTurns != 3 || loaded.stats.Turns != 1 || loaded.rng.State != g.rng.State {
			t.Fatal("free screens, saves and loading must not advance a warning")
		}
		loaded.quit = false
		lm.FrozenTurns = 1
		loaded.processCommand(".")
		if loaded.player.HP != 200 || !reflect.DeepEqual(lm.StrikeTiles, tiles) || lm.ActionTurns != 3 {
			t.Fatal("freezing must postpone the prepared strike without rerolling it")
		}
		loaded.processCommand(".")
		if loaded.player.HP >= 200 || len(lm.StrikeTiles) != 0 || lm.ActionTurns != 4 {
			t.Fatalf("%s must hurt a player who remains on the marked tile", kind)
		}
		g, m = warnedEpicArena(kind)
		g.processCommand(".")
		m.FleeTurns = 1
		g.processCommand(".")
		if len(m.StrikeTiles) != 0 || g.player.HP != 200 {
			t.Fatal("fleeing must cancel the warning")
		}
		g, m = warnedEpicArena(kind)
		g.processCommand(".")
		g.damageMonster(m, 10000, "burns")
		if len(m.StrikeTiles) != 0 || len(g.adventure.EpicsDefeated) != 1 {
			t.Fatal("death must cancel the warning and record the Epic defeat once")
		}
		g.damageMonster(m, 10000, "burns")
		if len(g.adventure.EpicsDefeated) != 1 {
			t.Fatal("a corpse must not earn duplicate chronicle entries")
		}
	}
}

func TestAshDodgesEpicWarningsAndBlinkAvoidsThem(t *testing.T) {
	g, m := warnedEpicArena(monsterStormLeviathan)
	g.dog.Freed, g.dog.Pos = true, pos{6, 6}
	g.processCommand(".")
	if !containsPosition(m.StrikeTiles, g.dog.Pos) {
		t.Fatal("the storm must mark both companions' positions")
	}
	before := g.dog.HP
	g.processCommand("a")
	if g.dog.HP != before || !strings.Contains(strings.Join(g.messages, "\n"), "Ash springs clear") {
		t.Fatal("Ash must spend his action dodging instead of attacking")
	}
	for _, freed := range []bool{false, true} {
		for _, alive := range []bool{false, true} {
			g, m = warnedEpicArena(monsterStormLeviathan)
			g.dog.Freed, g.dog.Alive = freed, alive
			m.StrikeTiles = []pos{g.dog.Pos}
			if (!freed || !alive) && g.dodgeEpicWarning() {
				t.Fatal("an inactive companion must not dodge or revive")
			}
		}
	}
	g, m = warnedEpicArena(monsterStormLeviathan)
	g.dog.Freed = true
	m.StrikeTiles = []pos{{1, 1}, {1, 2}, {2, 1}}
	g.player.BlinkStones = 1
	if !g.useBlinkStone() || g.epicDangerAt(g.player.Pos) || g.epicDangerAt(g.dog.Pos) {
		t.Fatal("blink must land both companions outside warned tiles")
	}
}

func TestEpicWarningsRespectWallsAndDodgeSpace(t *testing.T) {
	g, m := warnedEpicArena(monsterAshenWyrm)
	g.current().Tiles[5][6] = '#'
	g.processCommand(".")
	if len(m.StrikeTiles) != 0 {
		t.Fatal("warnings must not reach through a wall")
	}
	g, m = warnedEpicArena(monsterFrostMonarch)
	for _, next := range g.openNeighbors(g.player.Pos, true) {
		g.current().Tiles[next.Y][next.X] = '#'
	}
	g.prepareEpicStrike(m)
	if len(m.StrikeTiles) != 0 {
		t.Fatal("do not telegraph an attack when every dodge tile is blocked")
	}
}

func TestEpicSignaturesRespectSpellWardsAndConditionImmunity(t *testing.T) {
	for _, kind := range []monsterKind{monsterGraveTitan, monsterAshenWyrm, monsterHollowReaper, monsterFrostMonarch, monsterStormLeviathan, monsterDuskTyrant} {
		plain, _ := warnedEpicArena(kind)
		warded, _ := warnedEpicArena(kind)
		warded.player.BodyArmor.SpellWard = 8
		plain.processCommand(".")
		warded.processCommand(".")
		plain.processCommand(".")
		warded.processCommand(".")
		if warded.player.HP-plain.player.HP != 8 {
			t.Fatalf("spell ward must block eight points of %s's signature", kind)
		}
	}
	for _, kind := range []monsterKind{monsterFrostMonarch, monsterDuskTyrant} {
		for _, protected := range []bool{false, true} {
			g, _ := warnedEpicArena(kind)
			if protected {
				g.player.HeadArmor = godArmorPieces[1]
			}
			g.processCommand(".")
			g.processCommand(".")
			affected := g.player.BoundTurns > 0 || g.player.HexedTurns > 0
			if g.player.HP >= 200 || affected == protected {
				t.Fatal("a damaging signature must apply its condition unless Oracle Crown prevents it")
			}
		}
	}
}
