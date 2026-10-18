package main

type floorTheme string

const (
	themeChapel    floorTheme = "ruined chapel"
	themeBarracks  floorTheme = "abandoned barracks"
	themeOvergrown floorTheme = "overgrown catacombs"
	themeKennel    floorTheme = "forgotten kennels"
)

var floorThemes = []floorTheme{themeChapel, themeBarracks, themeOvergrown, themeKennel}

func themeFavors(theme floorTheme, m monster) bool {
	switch theme {
	case themeChapel:
		return m.Undead
	case themeBarracks:
		return monsterGuard(m.Kind) > 0 || m.Kind == monsterOrc || m.Kind == monsterGoblin || m.Kind == monsterGraveKnight || m.Kind == monsterCrownlessKnight
	case themeOvergrown:
		return m.Kind == monsterSpider || m.Kind == monsterCaveCrawler || m.Kind == monsterSlime || m.Kind == monsterAshBeetle || m.Kind == monsterIronrootBrute
	case themeKennel:
		return m.Kind == monsterRat || m.Kind == monsterTunnelBat || m.Kind == monsterBoneHound || m.Kind == monsterEmberHound || m.Kind == monsterFrostHound
	}
	return false
}

func (g *game) themedItems(lvl *level) []item {
	items := g.itemsForLevel(lvl.Index)
	// Reassign one existing supply instead of adding power to every floor.
	for i, it := range items {
		if it.Kind != itemFireScroll && it.Kind != itemWardingCharm && it.Kind != itemBlinkStone {
			continue
		}
		switch lvl.Theme {
		case themeChapel:
			items[i] = item{Kind: itemWardingCharm, Name: "warding charm", Glyph: itemGlyphs[itemWardingCharm]}
		case themeKennel:
			items[i] = item{Kind: itemPotion, Name: "healing potion", Glyph: itemGlyphs[itemPotion]}
		case themeOvergrown:
			items[i] = item{Kind: itemBlinkStone, Name: "blink stone", Glyph: itemGlyphs[itemBlinkStone]}
		}
		break
	}
	return items
}
