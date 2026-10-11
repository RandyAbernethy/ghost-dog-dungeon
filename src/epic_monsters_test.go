package main

import "testing"

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
		g.monsterTurn(&m)
		if m.Pos == first {
			t.Fatalf("%s should take its next step on turn four", kind)
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
