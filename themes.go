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
		return m.Kind == monsterSpider || m.Kind == monsterSlime || m.Kind == monsterAshBeetle || m.Kind == monsterIronrootBrute
	case themeKennel:
		return m.Kind == monsterRat || m.Kind == monsterBoneHound || m.Kind == monsterFrostHound
	}
	return false
}

func monsterThreat(m monster) int {
	return m.MaxHP + 2*(m.MinDamage+m.MaxDamage) + 3*monsterGuard(m.Kind)
}

// Themes change only a few picks, with a capped aggregate threat budget.
func (g *game) themedMonsters(lvl *level) []monster {
	monsters := g.monstersForLevel(lvl.Index)
	baseline := 0
	for _, m := range monsters {
		baseline += monsterThreat(m)
	}
	total := baseline
	changed := 0
	for i, m := range monsters {
		if m.Boss || themeFavors(lvl.Theme, m) || changed >= max(1, len(monsters)/3) {
			continue
		}
		choices := []monster{}
		for _, candidate := range monsterPoolForLevel(lvl.Index) {
			candidate = scaleMonsterForDepth(candidate, lvl.Index)
			newTotal := total - monsterThreat(m) + monsterThreat(candidate)
			if !candidate.Boss && themeFavors(lvl.Theme, candidate) && newTotal <= baseline*110/100 && newTotal >= baseline*90/100 {
				choices = append(choices, candidate)
			}
		}
		if len(choices) > 0 {
			monsters[i] = choices[g.rng.Intn(len(choices))]
			total += monsterThreat(monsters[i]) - monsterThreat(m)
			changed++
		}
	}
	return monsters
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
			items[i] = item{Kind: itemWardingCharm, Name: "warding charm", Glyph: '*'}
		case themeKennel:
			items[i] = item{Kind: itemPotion, Name: "healing potion", Glyph: '!'}
		case themeOvergrown:
			items[i] = item{Kind: itemBlinkStone, Name: "blink stone", Glyph: '*'}
		}
		break
	}
	return items
}
