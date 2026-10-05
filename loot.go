package main

import (
	"fmt"
)

func makeWeaponItem(w weapon) item {
	return item{Kind: itemWeapon, Name: w.Name, Glyph: ')', Weapon: w}
}

func makeWeaponLoot(w weapon) item {
	w.Max++
	return makeWeaponItem(w)
}

func makeArmorItem(a armor) item {
	return item{Kind: itemArmor, Name: a.Name, Glyph: '[', Armor: a}
}

func (g *game) weaponLootForLevel(depth int) item {
	weaponPools := [][]weapon{{
		{Name: "Short Sword", Min: 4, Max: 7},
		{Name: "Chapel Mace", Min: 4, Max: 8},
	}, {
		{Name: "Iron Spear", Min: 3, Max: 6, Reach: 2},
		{Name: "Hooked Glaive", Min: 5, Max: 9},
	}, {
		{Name: "Battle Axe", Min: 6, Max: 10},
		{Name: "Dawnstar Flail", Min: 6, Max: 10},
	}, {
		{Name: "Moonblade", Min: 7, Max: 11},
		{Name: "Rune Saber", Min: 7, Max: 12},
	}, {
		{Name: "Dragontooth Pike", Min: 6, Max: 10, Reach: 2},
		{Name: "Starforged Hammer", Min: 9, Max: 14},
	}, {
		{Name: "Gravecleaver", Min: 10, Max: 15},
		{Name: "Hollowfang Spear", Min: 8, Max: 12, Reach: 2},
	}, {
		{Name: "Frostbite Axe", Min: 11, Max: 16},
		{Name: "Gloomsteel Saber", Min: 12, Max: 16},
	}, {
		{Name: "Graveglass Halberd", Min: 10, Max: 14, Reach: 2},
		{Name: "Stormcaller Blade", Min: 13, Max: 17},
	}, {
		{Name: "Wyrmheart Maul", Min: 13, Max: 18},
		{Name: "Gloaming Pike", Min: 11, Max: 15, Reach: 2},
	}, {
		{Name: "Dawnforged Greatsword", Min: 15, Max: 19},
		{Name: "Kingsbane Axe", Min: 16, Max: 19},
	}}
	weapons := weaponPools[depth]
	weapons = append(weapons, weapon{Name: reachWeaponNames[depth], Min: 3 + depth, Max: 5 + depth, Reach: 2})
	return makeWeaponLoot(weapons[g.rng.Intn(len(weapons))])
}

func (g *game) armorLootForLevel(depth int) item {
	armorPools := [][]armor{{
		{Name: "Leather Jerkin", Slot: slotBody, Defense: 2, Rarity: rarityUncommon},
		{Name: "Scout Cap", Slot: slotHead, Defense: 2, Rarity: rarityUncommon},
	}, {
		{Name: "Iron Helm", Slot: slotHead, Defense: 2, Rarity: rarityUncommon},
		{Name: "Wolfhide Boots", Slot: slotLeg, Defense: 1, Rarity: rarityUncommon},
	}, {
		{Name: "Reinforced Greaves", Slot: slotLeg, Defense: 2, Rarity: rarityRare},
		{Name: "Scale Coat", Slot: slotBody, Defense: 3, Rarity: rarityRare},
	}, {
		{Name: "Knight Mail", Slot: slotBody, Defense: 3, Rarity: rarityEpic},
		{Name: "Nightstride Boots", Slot: slotLeg, Defense: 3, Rarity: rarityEpic},
	}, {
		{Name: "Warden Crown", Slot: slotHead, Defense: 3, Rarity: rarityLegendary},
		{Name: "Sunplate Cuirass", Slot: slotBody, Defense: 4, Rarity: rarityLegendary},
	}, {
		{Name: "Dreadmark Mantle", Slot: slotBody, Defense: 4, Rarity: rarityEpic},
		{Name: "Gloamrunner Boots", Slot: slotLeg, Defense: 3, Rarity: rarityEpic},
	}, {
		{Name: "Aetherweave Hood", Slot: slotHead, Defense: 4, Rarity: rarityEpic},
		{Name: "Ashguard Coat", Slot: slotBody, Defense: 4, Rarity: rarityEpic},
	}, {
		{Name: "Stormscale Greaves", Slot: slotLeg, Defense: 4, Rarity: rarityEpic},
		{Name: "Graveward Helm", Slot: slotHead, Defense: 4, Rarity: rarityEpic},
	}, {
		{Name: "Ruinplate Vest", Slot: slotBody, Defense: 5, Rarity: rarityEpic},
		{Name: "Mourner's Helm", Slot: slotHead, Defense: 5, Rarity: rarityEpic},
	}, {
		{Name: "Starforged Cuirass", Slot: slotBody, Defense: 5, Rarity: rarityLegendary},
		{Name: "Voidwalker Boots", Slot: slotLeg, Defense: 5, Rarity: rarityLegendary},
	}}
	armors := armorPools[depth]
	slot := []armorSlot{slotHead, slotBody, slotLeg}[depth%3]
	armors = append(armors,
		armor{Name: "Runespun " + slotLabel(slot) + " armor", Slot: slot, Defense: 1 + depth/3, SpellWard: 2 + depth/4, Rarity: rarityRare},
		armor{Name: "Hunter's " + slotLabel(slot) + " armor", Slot: slot, Defense: depth / 3, StrikeBonus: 1 + depth/4, Rarity: rarityRare},
	)
	return makeArmorItem(armors[g.rng.Intn(len(armors))])
}

func (g *game) itemsForLevel(depth int) []item {
	items := []item{}
	if depth < levelCount-1 {
		items = append(items, item{Kind: itemStoryScroll, Name: fmt.Sprintf("Ghost Dog story scroll %d", depth+1), Glyph: '?', StoryChapter: depth + 1})
	}
	if depth%2 == 0 {
		items = append(items, item{Kind: itemPotion, Name: "healing potion", Glyph: '!'})
	}
	items = append(items, g.weaponLootForLevel(depth), g.armorLootForLevel(depth))
	switch depth {
	case 0:
		items = append(items, item{Kind: itemFireScroll, Name: "fire scroll", Glyph: '?'})
	case 1:
		items = append(items, item{Kind: itemBlinkStone, Name: "blink stone", Glyph: '*'})
	case 2:
		items = append(items, item{Kind: itemFrostCharm, Name: "frost charm", Glyph: '*'})
	case 3:
		items = append(items, item{Kind: itemWardingCharm, Name: "warding charm", Glyph: '*'})
	case 4:
		items = append(items, item{Kind: itemSunOrb, Name: "sun orb", Glyph: '*'})
	case 5:
		items = append(items, item{Kind: itemFrostCharm, Name: "frost charm", Glyph: '*'})
	case 6:
		items = append(items, item{Kind: itemWardingCharm, Name: "warding charm", Glyph: '*'})
	case 7:
		items = append(items, item{Kind: itemFireScroll, Name: "fire scroll", Glyph: '?'})
	case 8:
		items = append(items, item{Kind: itemBlinkStone, Name: "blink stone", Glyph: '*'})
	case 9:
		items = append(items, item{Kind: itemSunOrb, Name: "sun orb", Glyph: '*'})
	}
	return items
}

func (g *game) collectItems() {
	g.describeSpecialRoom()
	lvl := g.current()
	kept := lvl.Items[:0]
	for _, it := range lvl.Items {
		if it.Pos != g.player.Pos {
			kept = append(kept, it)
			continue
		}
		g.collectItem(it)
	}
	lvl.Items = kept
}

func (g *game) collectItem(it item) {
	switch it.Kind {
	case itemWeapon:
		cmp := compareWeapon(it.Weapon, g.player.Weapon)
		g.player.ensureCarriedEquipment()
		g.player.carryWeapon(it.Weapon)
		g.addMessage(fmt.Sprintf("You pack %s (%d-%d, reach %d). %s Press i to choose equipment.", it.Weapon.Name, it.Weapon.Min, it.Weapon.Max, it.Weapon.effectiveReach(), cmp))
	case itemArmor:
		equipped := g.player.armorForSlot(it.Armor.Slot)
		cmp := compareArmor(it.Armor, equipped)
		candidateName := coloredArmorName(it.Armor)
		g.player.ensureCarriedEquipment()
		g.player.carryArmor(it.Armor)
		g.addMessage(fmt.Sprintf("You pack %s for your %s slot. %s %s Press i to choose equipment.", candidateName, slotLabel(it.Armor.Slot), cmp, armorTraits(it.Armor)))
	case itemPotion:
		g.player.Potions++
		g.addMessage("You stash a healing potion. Restores 12-18 health when used with p.")
	case itemFireScroll:
		g.player.FireScrolls++
		g.addMessage("You pocket a fire scroll. It detonates near the closest foe when used with f.")
	case itemBlinkStone:
		g.player.BlinkStones++
		g.addMessage("You take a blink stone. Use g to teleport to the safest open spot on this floor.")
	case itemWardingCharm:
		g.player.WardingCharms++
		g.addMessage("You take a warding charm. Use b to protect you and your Ghost Dog for 3 enemy turns.")
	case itemSunOrb:
		g.player.SunOrbs++
		g.addMessage("You cradle a sun orb. Use u to burn every undead monster on this floor.")
	case itemFrostCharm:
		g.player.FrostCharms++
		g.addMessage("You take a frost charm. Use t to freeze and wound the nearest foe.")
	case itemStarfireOrb:
		g.player.StarfireOrbs++
		g.addMessage("You take a starfire orb. Use o to blast the nearest foe for heavy damage.")
	case itemPhoenixAsh:
		g.player.PhoenixAshes++
		g.addMessage("You gather phoenix ash. Use n to restore a large amount of health.")
	case itemGhostRecallScroll:
		g.player.GhostRecallScrolls++
		g.addMessage("You find a Ghost Dog recall scroll. Use r to call your fallen companion back at full health.")
	case itemStoryScroll:
		if it.StoryChapter >= 1 && it.StoryChapter <= len(ghostDogStory) {
			seen := false
			for _, chapter := range g.player.StoryChapters {
				if chapter == it.StoryChapter {
					seen = true
					break
				}
			}
			if !seen {
				g.player.StoryChapters = append(g.player.StoryChapters, it.StoryChapter)
			}
			g.addMessage(fmt.Sprintf("Ghost Dog story scroll %d/10: %s", it.StoryChapter, ghostDogStory[it.StoryChapter-1]))
			g.addMessage("The chapter is added to your Book of Ghost Dog. Press B to read it.")
		}
	case itemSideStoryScroll:
		if id := it.SideStoryID; id >= 1 && id <= len(ghostDogSideStories) {
			if !hasStoryID(g.player.SideStories, id) {
				g.player.SideStories = append(g.player.SideStories, id)
			}
			story := ghostDogSideStories[id-1]
			g.addMessage(fmt.Sprintf("Side story: %s. %s", story.Title, story.Text))
			g.addMessage("The side story is added to your Book of Ghost Dog. Press B to read it.")
		}
	}
}
