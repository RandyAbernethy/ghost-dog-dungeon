package main

import "testing"

func varietyArena(seed int64) *game {
	lvl := &level{Tiles: blankFloor()}
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			lvl.Tiles[y][x] = '.'
		}
	}
	return &game{rng: newSimpleRNG(seed), levels: []*level{lvl}, knownMonsters: map[monsterKind]bool{},
		player: player{Pos: pos{5, 5}, HP: 46, MaxHP: 46}, dog: ghostDog{Pos: pos{4, 5}, HP: 34, MaxHP: 34, Alive: true}}
}

func TestAdditionalMonstersPerformTheirAdvertisedPowers(t *testing.T) {
	for _, pair := range [][2]monsterKind{
		{monsterCaveCrawler, monsterSpider}, {monsterBarrowArcher, monsterGloomArcher},
		{monsterGraveAcolyte, monsterBellRevenant}, {monsterEmberHound, monsterBoneHound},
		{monsterHollowWarden, monsterWraith},
	} {
		observed := false
		for seed := int64(1); seed <= 80; seed++ {
			a, b := varietyArena(seed), varietyArena(seed)
			ma, mb := newMonster(pair[0]), newMonster(pair[1])
			ma.Pos = pos{8, 5}
			if pair[0] == monsterHollowWarden {
				ma.Pos = pos{6, 5}
			}
			ma.HP, ma.MaxHP, ma.MinDamage, ma.MaxDamage = 20, 40, 4, 8
			mb.Pos, mb.HP, mb.MaxHP, mb.MinDamage, mb.MaxDamage = ma.Pos, ma.HP, ma.MaxHP, ma.MinDamage, ma.MaxDamage
			a.current().Monsters, b.current().Monsters = []*monster{&ma}, []*monster{&mb}
			a.monsterTurn(&ma)
			b.monsterTurn(&mb)
			if a.player.HP != b.player.HP || a.player.BoundTurns != b.player.BoundTurns || a.player.HexedTurns != b.player.HexedTurns || ma.Pos != mb.Pos || ma.HP != mb.HP || a.rng.State != b.rng.State {
				t.Fatalf("%s should use its advertised %s power", pair[0], pair[1])
			}
			if a.player.HP < 46 || a.player.BoundTurns > 0 || a.player.HexedTurns > 0 || ma.HP > 20 || distance(ma.Pos, a.player.Pos) == 1 {
				observed = true
			}
		}
		if !observed {
			t.Fatalf("never observed %s's power", pair[0])
		}
	}
	for _, kind := range []monsterKind{monsterRustGuard, monsterObsidianGuard} {
		g := varietyArena(1)
		m := newMonster(kind)
		g.damageMonster(&m, 5, "hits")
		if m.MaxHP-m.HP != 5-monsterGuard(kind) {
			t.Fatalf("%s's armor should block incoming damage", kind)
		}
	}
}
