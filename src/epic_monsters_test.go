package main

import (
	"strings"
	"testing"
)

func TestEpicsMoveAndAttackSlowlyAndCanBeOutrun(t *testing.T) {
	for _, kind := range epicMonsterKinds {
		g := varietyArena(1)
		m := newMonster(kind)
		m.Pos = pos{8, 5}
		g.current().Monsters = []*monster{&m}
		start := m.Pos
		g.monsterTurn(&m)
		if !m.Awakened || m.Pos == start {
			t.Fatalf("%s should wake and take its first step", kind)
		}
		first := m.Pos
		g.monsterTurn(&m)
		g.monsterTurn(&m)
		if m.Pos != first {
			t.Fatalf("%s should rest for two turns between steps", kind)
		}
		if len(m.StrikeTiles) == 0 {
			t.Fatalf("%s should telegraph its next active attack", kind)
		}
		hpBeforeDodge := g.player.HP
		for _, p := range g.openNeighbors(g.player.Pos, true) {
			if !g.epicDangerAt(p) {
				g.player.Pos = p
				break
			}
		}
		g.monsterTurn(&m)
		if m.Pos != first || len(m.StrikeTiles) != 0 || g.player.HP != hpBeforeDodge || !strings.Contains(strings.Join(g.messages, "\n"), "strikes empty stone") {
			t.Fatalf("%s should spend its fourth turn on the warned tiles and allow a dodge", kind)
		}
		g = varietyArena(1)
		m = newMonster(kind)
		m.Pos = pos{6, 5}
		g.current().Monsters = []*monster{&m}
		g.monsterTurn(&m)
		hp := g.player.HP
		g.monsterTurn(&m)
		g.monsterTurn(&m)
		if g.player.HP != hp {
			t.Fatalf("%s should rest between adjacent attacks", kind)
		}
		g = varietyArena(1)
		m = newMonster(kind)
		m.Pos = pos{6, 5}
		g.current().Monsters = []*monster{&m}
		for i := 0; i < 4; i++ {
			g.processCommand("a")
		}
		if g.player.HP != 46 || distance(g.player.Pos, m.Pos) <= 2 {
			t.Fatalf("%s must allow an immediate retreat", kind)
		}
	}
}

func TestEpicPowersAndDefenses(t *testing.T) {
	g := varietyArena(1)
	m := newMonster(monsterIronColossus)
	m.Pos, m.MinDamage, m.MaxDamage = pos{6, 5}, 10, 10
	g.player.ShieldTurns = 3
	g.attackPlayer(&m)
	if g.player.ShieldTurns != 0 || g.player.HP != 37 {
		t.Fatal("Colossus should shatter the charm before damage")
	}
	g = varietyArena(1)
	m = newMonster(monsterGraveTitan)
	m.Pos, m.HP, m.MinDamage, m.MaxDamage = pos{6, 5}, 100, 10, 10
	g.attackPlayer(&m)
	if m.HP != 108 {
		t.Fatal("Titan should heal after dealing melee damage")
	}
	g.player.BodyArmor = armor{Defense: 100}
	g.attackPlayer(&m)
	if m.HP != 108 {
		t.Fatal("Titan must not heal when its attack deals no damage")
	}
	g = varietyArena(1)
	m = newMonster(monsterAshenWyrm)
	m.Pos, m.MinDamage, m.MaxDamage = pos{6, 5}, 10, 10
	g.dog.Freed, g.dog.Pos = true, pos{6, 6}
	g.attackPlayer(&m)
	if g.player.HP != 25 || g.dog.HP != 22 {
		t.Fatal("Wyrm's damaging strike should flare against adjacent player and dog")
	}
	g = varietyArena(1)
	m = newMonster(monsterHollowReaper)
	m.Pos = pos{6, 5}
	g.player.Weapon = weapon{Min: 10, Max: 10}
	g.attackMonster(&m)
	if g.player.HP != 34 {
		t.Fatal("Reaper should reflect close-range hits")
	}
	g.player.Pos = pos{4, 5}
	g.player.Weapon.Reach = 2
	g.attackMonster(&m)
	if g.player.HP != 34 {
		t.Fatal("reach should avoid Reaper's contact reflection")
	}
}

func TestNewEpicPowersAndDefenses(t *testing.T) {
	t.Run("Frost Monarch movement cost", func(t *testing.T) {
		g := varietyArena(1)
		m := newMonster(monsterFrostMonarch)
		m.Pos, m.MinDamage, m.MaxDamage = pos{6, 5}, 10, 10
		g.attackPlayer(&m)
		if g.player.BoundTurns != 1 || g.player.Binding != bindingIce {
			t.Fatal("the Frost Monarch's damaging melee strike must freeze the player's feet")
		}
		start := g.player.Pos
		g.processCommand("a")
		if g.player.Pos != start || g.stats.Turns != 1 || g.player.BoundTurns != 0 || !strings.Contains(strings.Join(g.messages, "\n"), "Ice locks your feet") {
			t.Fatal("ice must name its cause and cost exactly one attempted movement turn")
		}
		g.processCommand("a")
		if g.player.Pos == start || g.stats.Turns != 2 {
			t.Fatal("the player must move normally after breaking the ice")
		}
	})
	t.Run("Dusk Tyrant next strike", func(t *testing.T) {
		g := varietyArena(1)
		m := newMonster(monsterDuskTyrant)
		m.Pos, m.MinDamage, m.MaxDamage = pos{6, 5}, 10, 10
		g.player.Weapon = weapon{Min: 20, Max: 20}
		g.attackPlayer(&m)
		if g.player.HexedTurns != 1 {
			t.Fatal("the Dusk Tyrant's damaging melee strike must hex the player")
		}
		before := m.HP
		g.attackMonster(&m)
		if before-m.HP != 11 || g.player.HexedTurns != 0 {
			t.Fatal("the hex must weaken exactly the next strike, with the Tyrant's armor still applying")
		}
		before = m.HP
		g.attackMonster(&m)
		if before-m.HP != 14 {
			t.Fatal("the following strike must regain its normal damage")
		}
	})
	t.Run("Storm Leviathan range and wards", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			freed bool
			alive bool
			dogX  int
			ward  int
			dogHP int
		}{{"adjacent", true, true, 7, 0, 25}, {"two tiles", true, true, 8, 0, 25},
			{"out of range", true, true, 9, 0, 34}, {"not freed", false, true, 7, 0, 34},
			{"fallen", true, false, 7, 0, 0}, {"spell ward", true, true, 8, 9, 25}} {
			t.Run(tc.name, func(t *testing.T) {
				g := varietyArena(1)
				m := newMonster(monsterStormLeviathan)
				m.Pos, m.MinDamage, m.MaxDamage = pos{6, 5}, 10, 10
				g.dog.Freed, g.dog.Alive, g.dog.Pos = tc.freed, tc.alive, pos{tc.dogX, 5}
				if !tc.alive {
					g.dog.HP = 0
				}
				g.player.BodyArmor.SpellWard = tc.ward
				g.attackPlayer(&m)
				if g.player.HP != 28+tc.ward || g.dog.HP != tc.dogHP || g.dog.Alive != tc.alive {
					t.Fatalf("lightning must respect range, companion state, and spell wards: player=%d dog=%+v", g.player.HP, g.dog)
				}
			})
		}
		g := varietyArena(1)
		m := newMonster(monsterStormLeviathan)
		m.Pos, m.MinDamage, m.MaxDamage = pos{6, 5}, 10, 10
		g.player.Pos, g.dog.Pos, g.dog.Freed = pos{4, 5}, pos{7, 5}, true
		g.attackDog(&m)
		if g.player.HP != 37 || g.dog.HP != 16 {
			t.Fatal("a damaging strike on Ghost Dog must also arc lightning to nearby foes")
		}
	})
	t.Run("Obsidian Golem armor", func(t *testing.T) {
		g := varietyArena(1)
		m := newMonster(monsterObsidianGolem)
		m.Pos = pos{6, 5}
		before := m.HP
		g.damageMonster(&m, 20, "hits")
		if before-m.HP != 8 {
			t.Fatal("the Golem's shell must block 12 damage")
		}
		g.player.Weapon = artifactWeapons[2]
		g.player.Weapon.Min, g.player.Weapon.Max = 20, 20
		before = m.HP
		g.attackMonster(&m)
		if before-m.HP != 20 {
			t.Fatal("Worldsplitter must pierce the Golem's entire shell")
		}
	})
}

func TestNewEpicEffectsRespectBlockedHitsAndOracleCrown(t *testing.T) {
	for _, kind := range []monsterKind{monsterFrostMonarch, monsterStormLeviathan, monsterDuskTyrant} {
		g := varietyArena(1)
		m := newMonster(kind)
		m.Pos, m.MinDamage, m.MaxDamage = pos{6, 5}, 10, 10
		g.player.BodyArmor = armor{Defense: 300}
		g.dog.Freed, g.dog.Pos = true, pos{8, 5}
		g.attackPlayer(&m)
		if g.player.HP != 46 || g.dog.HP != 34 || g.player.BoundTurns != 0 || g.player.HexedTurns != 0 {
			t.Fatalf("%s must not trigger a power when its melee attack deals no damage", kind)
		}
	}
	for _, kind := range []monsterKind{monsterFrostMonarch, monsterDuskTyrant} {
		g := varietyArena(1)
		m := newMonster(kind)
		m.Pos, m.MinDamage, m.MaxDamage = pos{6, 5}, 10, 10
		g.player.HeadArmor = godArmorPieces[1]
		g.attackPlayer(&m)
		if g.player.HP >= 46 || g.player.BoundTurns != 0 || g.player.HexedTurns != 0 || !strings.Contains(strings.Join(g.messages, "\n"), "Oracle Crown") {
			t.Fatalf("Oracle Crown must prevent %s's effect despite taking melee damage", kind)
		}
	}
}

func epicFight(kind monsterKind, gear *item, seed int64) bool {
	g := varietyArena(seed)
	g.player.Weapon = weapon{Name: "Deep blade", Min: 12, Max: 17}
	g.player.HeadArmor = armor{Defense: 4}
	g.player.BodyArmor = armor{Defense: 4}
	g.player.LegArmor = armor{Defense: 3}
	g.player.Potions, g.player.PhoenixAshes = 3, 1
	g.dog.Freed = true
	if gear != nil {
		if gear.Kind == itemWeapon {
			g.player.Weapon = gear.Weapon
		} else {
			g.player.setArmor(gear.Armor)
		}
	}
	m := scaleMonsterForDepth(newMonster(kind), 6)
	m.Pos = pos{6, 5}
	g.current().Monsters = []*monster{&m}
	for turn := 0; turn < 300 && g.player.HP > 0 && m.HP > 0; turn++ {
		command := "d"
		if g.player.HP <= 20 && g.player.PhoenixAshes > 0 {
			command = "n"
		} else if g.player.HP <= 30 && g.player.Potions > 0 {
			command = "p"
		}
		g.processCommand(command)
	}
	return g.player.HP > 0 && m.HP == 0
}

func TestEpicFightDifficultyAndSingleRelicAdvantage(t *testing.T) {
	ordinary, relicWins := 0, 0
	pool := relicPool()
	for _, kind := range epicMonsterKinds {
		without := 0
		with := make([]int, len(pool))
		for seed := int64(1); seed <= 50; seed++ {
			if epicFight(kind, nil, seed) {
				without++
				ordinary++
			}
			for i, it := range pool {
				if epicFight(kind, &it, seed) {
					with[i]++
					relicWins++
				}
			}
		}
		t.Logf("%s: ordinary wins %d/50; single relic wins in catalog order %v", kind, without, with)
		if without > 10 {
			t.Fatalf("%s should be very hard to defeat with ordinary gear", kind)
		}
	}
	if relicWins <= ordinary*len(pool) || relicWins == 0 {
		t.Fatal("a single relic must materially improve Epic fights")
	}
	if relicWins == 50*len(pool)*len(epicMonsterKinds) {
		t.Fatal("relics must not guarantee survival")
	}
}

func TestEpicKillAwardsDifficultyOnceIncludingGodReprisal(t *testing.T) {
	g := varietyArena(1)
	g.player.LegArmor = godArmorPieces[2]
	m := newMonster(monsterHollowReaper)
	m.Pos, m.HP, m.MinDamage, m.MaxDamage = pos{6, 5}, 8, 10, 10
	g.current().Monsters = []*monster{&m}
	want := monsterDifficulty(m)
	g.attackPlayer(&m)
	g.damageMonster(&m, 99, "hits")
	if m.HP != 0 || g.stats.MonstersKilled != 1 || g.stats.DifficultyDefeated != want {
		t.Fatal("Epic killed by God armor must award its full difficulty once")
	}
}
