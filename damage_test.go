package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestHitReactionsReflectActualDamageAsPercentageOfMaximumHealth(t *testing.T) {
	for _, tc := range []struct {
		name              string
		kind              monsterKind
		hp, maxHP, damage int
		wantHP            int
		reaction          string
	}{
		{"small hit", monsterOrc, 100, 100, 1, 99, "barely flinches"},
		{"below twenty percent", monsterOrc, 100, 100, 19, 81, "barely flinches"},
		{"twenty percent", monsterOrc, 100, 100, 20, 80, "recoils"},
		{"below forty percent", monsterOrc, 100, 100, 39, 61, "recoils"},
		{"forty percent", monsterOrc, 100, 100, 40, 60, "staggers"},
		{"below sixty percent", monsterOrc, 100, 100, 59, 41, "staggers"},
		{"sixty percent", monsterOrc, 100, 100, 60, 40, "reels in agony"},
		{"below eighty percent", monsterOrc, 100, 100, 79, 21, "reels in agony"},
		{"eighty percent", monsterOrc, 100, 100, 80, 20, "nearly collapses"},
		{"almost lethal", monsterOrc, 100, 100, 99, 1, "nearly collapses"},
		{"smaller creature", monsterOrc, 10, 10, 4, 6, "staggers"},
		{"wounded creature", monsterOrc, 25, 100, 20, 5, "recoils"},
		{"armor reduces severity", monsterIronrootBrute, 100, 100, 22, 81, "barely flinches"},
		{"armor-adjusted boundary", monsterIronrootBrute, 100, 100, 23, 80, "recoils"},
		{"fully blocked hit", monsterIronrootBrute, 100, 100, 3, 100, "barely flinches"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.player.Weapon = weapon{Name: "Test Sword", Min: tc.damage, Max: tc.damage}
			m := newMonster(tc.kind)
			m.HP, m.MaxHP, m.Pos = tc.hp, tc.maxHP, pos{6, 5}
			g.attackMonster(&m)
			want := fmt.Sprintf("%s %s from the hit. (%d/%d)", m.Name, tc.reaction, tc.wantHP, tc.maxHP)
			if m.HP != tc.wantHP || g.messages[len(g.messages)-1] != want || g.turnDamage[&m] != tc.hp-tc.wantHP {
				t.Fatalf("hp=%d reaction=%q actual damage=%d; want hp=%d reaction=%q", m.HP, g.messages[len(g.messages)-1], g.turnDamage[&m], tc.wantHP, want)
			}
			wantStun := 0
			if tc.reaction == "nearly collapses" {
				wantStun = 1
			}
			if m.StunnedTurns != wantStun {
				t.Fatalf("reaction %q should stun for %d turn, got %d", tc.reaction, wantStun, m.StunnedTurns)
			}
		})
	}
}

func TestNearlyCollapsingMonsterSkipsOneTurnThenAttacksAgain(t *testing.T) {
	g := openArena()
	g.player.Weapon = weapon{Name: "Test Sword", Min: 80, Max: 80}
	m := newMonster(monsterOrc)
	m.HP, m.MaxHP, m.Pos = 100, 100, pos{6, 5}
	m.MinDamage, m.MaxDamage = 5, 5
	g.current().Monsters = []*monster{&m}
	beforeHP, beforePos := g.player.HP, m.Pos
	g.processCommand("d")
	if m.HP != 20 || g.player.HP != beforeHP || m.Pos != beforePos || m.StunnedTurns != 0 || m.FleeTurns != 0 {
		t.Fatal("an 80% hit should stun the monster through its immediate next action")
	}
	if !strings.Contains(strings.Join(g.messages, "\n"), "nearly collapses from the hit") || g.messages[len(g.messages)-1] != m.Name+" is stunned and cannot act." {
		t.Fatal("a severe hit should report both nearly collapsing and losing an action to stun")
	}
	g.processCommand(".")
	if g.player.HP >= beforeHP || m.StunnedTurns != 0 || g.stats.Turns != 2 {
		t.Fatal("the monster should resume attacking after skipping exactly one action")
	}
}

func TestStunPreventsMovementPowersAndDogAttacks(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind monsterKind
		pos  pos
		flee int
	}{
		{"movement", monsterBoneHound, pos{12, 5}, 0},
		{"ranged spells", monsterStormHerald, pos{8, 5}, 0},
		{"dog attacks", monsterOrc, pos{6, 6}, 0},
		{"fleeing", monsterOrc, pos{6, 5}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.dog.Freed, g.dog.Pos = true, pos{5, 6}
			m := newMonster(tc.kind)
			m.HP, m.MaxHP, m.Pos, m.FleeTurns = 100, 100, tc.pos, tc.flee
			g.damageMonster(&m, 80, "mauls")
			beforePlayerHP, beforeDogHP, beforeRNG := g.player.HP, g.dog.HP, g.rng.State
			g.monsterTurn(&m)
			if g.player.HP != beforePlayerHP || g.dog.HP != beforeDogHP || m.Pos != tc.pos || m.FleeTurns != tc.flee || g.rng.State != beforeRNG || m.StunnedTurns != 0 {
				t.Fatal("a stunned monster should skip its entire next action")
			}
		})
	}
}

func TestPendingStunSurvivesSaveAndExpiresAfterOneAction(t *testing.T) {
	g := openArena()
	g.saveFile = filepath.Join(t.TempDir(), "stun.json")
	g.timestampedSaves = false
	m := newMonster(monsterOrc)
	m.HP, m.MaxHP, m.Pos = 100, 100, pos{6, 5}
	m.MinDamage, m.MaxDamage = 5, 5
	g.current().Monsters = []*monster{&m}
	g.damageMonster(&m, 80, "hits")
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.current().Monsters[0].StunnedTurns != 1 {
		t.Fatal("save/load should retain a pending stun")
	}
	beforeHP := loaded.player.HP
	loaded.processCommand(".")
	if loaded.player.HP != beforeHP || loaded.current().Monsters[0].StunnedTurns != 0 {
		t.Fatal("the saved stun should prevent the next attack and then expire")
	}
	loaded.processCommand(".")
	if loaded.player.HP >= beforeHP {
		t.Fatal("saving must not make the stun permanent")
	}
}

func TestLethalHitsAndElementalReactionsKeepTheirMessages(t *testing.T) {
	g := openArena()
	m := newMonster(monsterOrc)
	g.damageMonster(&m, m.HP+100, "hits")
	if m.HP != 0 || g.stats.MonstersKilled != 1 || g.messages[len(g.messages)-1] != m.Name+" falls." {
		t.Fatal("lethal hits should still report the monster's death")
	}
	for _, verb := range []string{"burns", "scorches", "sears", "freezes"} {
		m = newMonster(monsterOrc)
		g.damageMonster(&m, 1, verb)
		if !strings.Contains(g.messages[len(g.messages)-1], m.Name+" "+verb) {
			t.Fatal("elemental damage should retain its fire or frost feedback")
		}
	}
}
