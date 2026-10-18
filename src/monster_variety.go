package main

type monsterVariety struct {
	Kind       monsterKind
	FirstFloor int
}

var additionalMonsters = []monsterVariety{
	{monsterTunnelBat, 0}, {monsterCaveCrawler, 1}, {monsterRustGuard, 1},
	{monsterBarrowArcher, 3}, {monsterGraveAcolyte, 4}, {monsterEmberHound, 5},
	{monsterObsidianGuard, 6}, {monsterHollowWarden, 7},
}

func additionalMonsterSpec(kind monsterKind) (monster, bool) {
	var m monster
	switch kind {
	case monsterTunnelBat:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 6, MinDamage: 1, MaxDamage: 3}
	case monsterCaveCrawler:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 7, MinDamage: 2, MaxDamage: 3}
	case monsterRustGuard:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 12, MinDamage: 3, MaxDamage: 6}
	case monsterBarrowArcher:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 18, MinDamage: 3, MaxDamage: 5, Undead: true}
	case monsterGraveAcolyte:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 18, MinDamage: 4, MaxDamage: 6, Undead: true}
	case monsterEmberHound:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 22, MinDamage: 4, MaxDamage: 7}
	case monsterObsidianGuard:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 29, MinDamage: 5, MaxDamage: 8}
	case monsterHollowWarden:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 28, MinDamage: 5, MaxDamage: 8, Undead: true}
	default:
		return monster{}, false
	}
	m.Kind, m.Name, m.HP = kind, string(kind), m.MaxHP
	return m, true
}

func additionalMonsterDescription(kind monsterKind) string {
	switch kind {
	case monsterTunnelBat:
		return "A frail tunnel hunter; pairs are more troublesome than a lone bat."
	case monsterCaveCrawler:
		return "Spits silk from four tiles away and can entangle you with its bite."
	case monsterRustGuard:
		return "Its salvaged iron plates block 1 damage per hit."
	case monsterBarrowArcher:
		return "Undead; fires physical arrows from five tiles away."
	case monsterGraveAcolyte:
		return "Undead; distant chants deal spell damage and weaken your next strike."
	case monsterEmberHound:
		return "A living pack hunter that sometimes charges twice or bites as it closes."
	case monsterObsidianGuard:
		return "A heavy stone guard that blocks 2 damage per hit."
	case monsterHollowWarden:
		return "Undead; restores health when its draining melee strike lands."
	}
	return ""
}
