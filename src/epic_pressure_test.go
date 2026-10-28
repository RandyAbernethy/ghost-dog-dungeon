package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func rageArena(kind monsterKind) (*game, *monster) {
	g, m := warnedEpicArena(kind)
	m.ActionTurns = 0
	return g, m
}

func TestEpicsAlternatePressureAndSignatures(t *testing.T) {
	for _, kind := range epicMonsterKinds {
		g, m := rageArena(kind)
		g.processCommand(".")
		if g.player.HP >= 200 || m.ActionTurns != 1 {
			t.Fatalf("%s must threaten a reach user with its normal attack", kind)
		}
		g.processCommand(".")
		g.processCommand(".")
		if len(m.StrikeTiles) == 0 {
			t.Fatal("first signature should be warned")
		}
		for _, move := range []struct{ out, back string }{{"a", "d"}, {"w", "s"}, {"q", "x"}, {"s", "w"}} {
			dx, dy, _ := moveDelta(move.out)
			if !g.epicDangerAt(pos{g.player.Pos.X + dx, g.player.Pos.Y + dy}) {
				g.processCommand(move.out)
				g.processCommand(move.back)
				break
			}
		}
		g.processCommand(".")
		if len(m.StrikeTiles) != 0 {
			t.Fatal("next active action must be normal pressure, not another signature")
		}
		hp := g.player.HP
		g.processCommand(".")
		if g.player.HP >= hp || m.ActionTurns != 7 {
			t.Fatal("dodging signatures must not also avoid normal pressure")
		}
	}
}

func TestRageUsesAllDamageSourcesAndTwoActionEscape(t *testing.T) {
	for _, kind := range epicMonsterKinds {
		t.Run(string(kind), func(t *testing.T) {
			g, m := rageArena(kind)
			for _, verb := range []string{"hits", "mauls", "burns", "sears"} {
				g.damageMonster(m, 20, verb)
			}
			if m.Rage != epicRageLimit {
				t.Fatal("every damaging source must build rage")
			}
			if strings.Count(strings.Join(g.messages, "\n"), m.Name+" grows more enraged.") != 4 {
				t.Fatalf("each damaging hit must describe growing anger: %q", g.messages)
			}
			var gauge bytes.Buffer
			g.renderEpicWarnings(&gauge)
			if !strings.Contains(gauge.String(), m.Name+": rage 4/4") {
				t.Fatal("the rage gauge must retain the numeric count")
			}
			g.processCommand(".")
			if m.RetaliationTurns != 2 || m.Rage != 0 || !g.epicDangerAt(g.player.Pos) {
				t.Fatal("full rage must announce a two-action wide attack")
			}
			if g.messages[len(g.messages)-1] != m.Name+" erupts in rage! "+epicStrikeName(kind)+" will strike in two actions. Move clear." {
				t.Fatalf("retaliation must retain timing without explaining map symbols: %q", g.messages)
			}
			for _, next := range g.openNeighbors(g.player.Pos, true) {
				if !g.epicDangerAt(next) {
					t.Fatal("the wide attack should require two moves from its center")
				}
			}
			tiles := append([]pos(nil), m.StrikeTiles...)
			g.processCommand("a")
			if m.RetaliationTurns != 1 || !g.epicDangerAt(g.player.Pos) || !reflect.DeepEqual(tiles, m.StrikeTiles) || g.player.HP != 200 {
				t.Fatal("first step must neither take damage nor retarget the warning")
			}
			g.processCommand("a")
			if g.player.HP != 200 || len(m.StrikeTiles) != 0 || m.RetaliationTurns != 0 {
				t.Fatal("two timely moves must escape without requiring blink")
			}
		})
	}
}

func TestRageCanKillGreedyPlayerAndRespectsDefenses(t *testing.T) {
	for _, kind := range epicMonsterKinds {
		g, m := rageArena(kind)
		g.player.HP = 5
		m.Rage = epicRageLimit
		g.processCommand(".")
		g.processCommand(".")
		g.processCommand(".")
		if g.player.HP != 0 {
			t.Fatalf("ignoring %s's retaliation must risk death", kind)
		}
	}
	g, m := rageArena(monsterStormLeviathan)
	g.player.BodyArmor.SpellWard = 8
	m.Rage = epicRageLimit
	g.processCommand(".")
	g.processCommand(".")
	g.processCommand(".")
	if g.player.HP != 189 {
		t.Fatalf("retaliation must honor spell ward and armor, got HP %d", g.player.HP)
	}
}

func TestRagePersistenceFreeReadsInterruptsAndCancellation(t *testing.T) {
	g, m := rageArena(monsterStormLeviathan)
	m.Rage = epicRageLimit
	g.processCommand(".")
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	lm := loaded.current().Monsters[0]
	var out bytes.Buffer
	if err := loaded.loop(strings.NewReader("i\n1\nmap\nm\nsave\nload\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	lm = loaded.current().Monsters[0]
	if lm.Rage != m.Rage || lm.RetaliationTurns != 2 || !reflect.DeepEqual(lm.StrikeTiles, m.StrikeTiles) || loaded.rng.State != g.rng.State {
		t.Fatal("menus, reads and saving must preserve retaliation exactly")
	}
	loaded.quit = false
	lm.FrozenTurns = 1
	loaded.processCommand(".")
	if lm.RetaliationTurns != 2 {
		t.Fatal("freeze must delay the retaliation countdown")
	}
	lm.FleeTurns = 1
	loaded.processCommand(".")
	if lm.RetaliationTurns != 0 || len(lm.StrikeTiles) != 0 {
		t.Fatal("fear must cancel retaliation")
	}
	g, m = rageArena(monsterStormLeviathan)
	m.Rage = epicRageLimit
	g.processCommand(".")
	g.damageMonster(m, 10000, "burns")
	if m.Rage != 0 || m.RetaliationTurns != 0 || len(m.StrikeTiles) != 0 {
		t.Fatal("death must clear all pending combat state")
	}
}

func TestAshEscapesWideRetaliationAndInactiveAshCannotDodge(t *testing.T) {
	g, m := rageArena(monsterStormLeviathan)
	g.dog.Freed = true
	g.dog.Pos = pos{5, 7}
	m.Rage = epicRageLimit
	g.processCommand(".")
	before := g.dog.HP
	g.processCommand("a")
	g.processCommand("a")
	if g.dog.HP != before || g.epicDangerAt(g.dog.Pos) {
		t.Fatal("Ash must start escaping wide attacks immediately")
	}
	for _, chained := range []bool{false, true} {
		g, m = rageArena(monsterStormLeviathan)
		g.dog.Freed, g.dog.Alive, g.dog.Chained, g.dog.ChainFloor = true, chained, chained, g.currentLevel
		m.StrikeTiles = []pos{g.dog.Pos}
		m.RetaliationTurns = 2
		if g.dodgeEpicWarning() {
			t.Fatal("fallen or chained Ash must not dodge")
		}
	}
}

func TestRetaliationRequiresARealEscapeRoute(t *testing.T) {
	g, m := rageArena(monsterFrostMonarch)
	for _, p := range g.openNeighbors(g.player.Pos, true) {
		g.current().Tiles[p.Y][p.X] = '#'
	}
	if g.prepareEpicRetaliation(m) {
		t.Fatal("a wide pattern must not trap an actor without a two-step escape")
	}
}

func TestRetaliationCannotDemandMovesFromHeldPlayerOrChainedAsh(t *testing.T) {
	g, m := rageArena(monsterStormLeviathan)
	g.player.BoundTurns, g.player.Binding = 1, bindingIce
	if g.prepareEpicRetaliation(m) {
		t.Fatal("a movement hold prevents a two-move escape")
	}
	g, m = rageArena(monsterStormLeviathan)
	g.dog.Freed, g.dog.Alive, g.dog.Chained, g.dog.ChainFloor = true, true, true, g.currentLevel
	g.dog.Pos = pos{6, 6}
	if g.prepareEpicRetaliation(m) {
		t.Fatal("chained Ash cannot be promised an escape")
	}
}

func TestInvalidEpicCombatSavesAreRejected(t *testing.T) {
	for _, mutate := range []func(*monster){
		func(m *monster) { m.Rage = -1 }, func(m *monster) { m.Rage = 5 },
		func(m *monster) { m.RetaliationTurns = 3 }, func(m *monster) { m.RetaliationTurns = 1; m.StrikeTiles = nil },
		func(m *monster) { m.Kind = monsterRat; m.Rage = 1 },
	} {
		g, m := rageArena(monsterStormLeviathan)
		g.saveDir = t.TempDir()
		mutate(m)
		if err := g.save(); err != nil {
			t.Fatal(err)
		}
		if _, err := loadGame(g.saveFile); err == nil || !strings.Contains(err.Error(), "invalid Epic combat state") {
			t.Fatalf("invalid combat state must fail clearly: %v", err)
		}
	}
}

func TestRetaliationThroughExecutableSaveLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("compiled retaliation replay")
	}
	g, m := rageArena(monsterStormLeviathan)
	m.Rage = epicRageLimit
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader(".\nsave\nload\na\nsave\nload\na\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("retaliation replay: %v\n%s", err, output)
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	g.processCommand(".")
	g.processCommand("a")
	g.processCommand("a")
	if !reflect.DeepEqual(g.levels, loaded.levels) || !reflect.DeepEqual(g.player, loaded.player) || g.stats != loaded.stats || g.rng.State != loaded.rng.State {
		t.Fatal("loaded retaliation must match uninterrupted combat")
	}
	if !strings.Contains(string(output), "RETALIATION in 1 action") {
		t.Fatal("countdown must remain visible after saving and loading")
	}
	prose := strings.Join(strings.Fields(stripANSI(string(output))), " ")
	if !strings.Contains(prose, m.Name+" erupts in rage! "+epicStrikeName(m.Kind)+" will strike in two actions. Move clear.") {
		t.Fatal("the executable must show the concise retaliation warning")
	}
}
