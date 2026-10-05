package main

import "fmt"

func (g *game) rollGhostGifts(depth int) []item {
	pool := []item{
		g.armorLootForLevel(depth),
		g.weaponLootForLevel(depth),
		{Kind: itemPotion, Name: "healing potion", Glyph: '!'},
		{Kind: itemBlinkStone, Name: "blink stone", Glyph: '*'},
		{Kind: itemFireScroll, Name: "fire scroll", Glyph: '?'},
		{Kind: itemWardingCharm, Name: "warding charm", Glyph: '*'},
		{Kind: itemSunOrb, Name: "sun orb", Glyph: '*'},
		{Kind: itemFrostCharm, Name: "frost charm", Glyph: '*'},
		{Kind: itemStarfireOrb, Name: "starfire orb", Glyph: '*'},
		{Kind: itemPhoenixAsh, Name: "phoenix ash", Glyph: '*'},
		{Kind: itemGhostRecallScroll, Name: "Ghost Dog recall scroll", Glyph: '?'},
	}
	// Sample distinct categories so the player always has three different options.
	for i := 0; i < 3; i++ {
		j := i + g.rng.Intn(len(pool)-i)
		pool[i], pool[j] = pool[j], pool[i]
	}
	return pool[:3]
}

func (g *game) prepareGhostGifts(lvl *level) {
	for i := range lvl.Events {
		e := &lvl.Events[i]
		if e.Kind == eventGhost && !e.Used && len(e.Gifts) == 0 {
			e.Gifts = g.rollGhostGifts(lvl.Index)
		}
	}
}

func (g *game) describeGhostGifts(e *floorEvent) {
	g.addMessage("The friendly ghost offers one gift. Choose from:")
	for i, gift := range e.Gifts {
		g.addMessage(fmt.Sprintf("%d. %s", i+1, inspectItemText(gift)))
	}
	g.addMessage("Type :choose 1, :choose 2, or :choose 3 and Enter (choose 1, choose 2, or choose 3 in line-input mode).")
}

func (g *game) chooseGhostGift(choice byte) bool {
	e := g.nearbyEvent()
	if e == nil || e.Kind != eventGhost || choice < 1 || choice > 3 || int(choice) > len(e.Gifts) {
		g.addMessage("A nearby friendly ghost must offer that gift. Choose 1, 2, or 3.")
		return false
	}
	gift := e.Gifts[choice-1]
	g.collectItem(gift)
	g.addMessage(fmt.Sprintf("The friendly ghost gives you one %s, then fades.", gift.Name))
	e.Used = true
	return true
}
