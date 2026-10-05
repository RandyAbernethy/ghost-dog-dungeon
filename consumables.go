package main

import (
	"fmt"
)

func (g *game) usePotion() bool {
	if g.player.Potions == 0 {
		g.addMessage("You have no healing potions.")
		return false
	}
	g.player.Potions--
	heal := g.randRange(12, 18)
	g.player.HP += heal
	if g.player.HP > g.player.MaxHP {
		g.player.HP = g.player.MaxHP
	}
	g.addMessage(fmt.Sprintf("You drink a potion and recover %d health.", heal))
	return true
}

func (g *game) useFireScroll() bool {
	if g.player.FireScrolls == 0 {
		g.addMessage("You have no fire scrolls.")
		return false
	}
	m := g.nearestMonster(6)
	if m == nil {
		g.addMessage("No monster is close enough for the fire scroll.")
		return false
	}
	g.player.FireScrolls--
	damage := g.randRange(10, 16)
	g.addMessage(fmt.Sprintf("The fire scroll explodes around %s for %d damage.", m.Name, damage))
	g.damageMonster(m, damage, "burns")
	for _, other := range g.current().Monsters {
		if other != m && other.HP > 0 && distance(other.Pos, m.Pos) == 1 {
			splash := g.randRange(3, 6)
			g.damageMonster(other, splash, "scorches")
		}
	}
	return true
}

func (g *game) useWardingCharm() bool {
	if g.player.WardingCharms == 0 {
		g.addMessage("You have no warding charms.")
		return false
	}
	g.player.WardingCharms--
	g.player.ShieldTurns = 3
	if g.dog.Freed && g.dog.Alive {
		g.addMessage("A pale ward circles you and the Ghost Dog for the next 3 enemy turns.")
	} else {
		g.addMessage("A pale ward circles you for the next 3 enemy turns.")
	}
	return true
}

func (g *game) useBlinkStone() bool {
	if g.player.BlinkStones == 0 {
		g.addMessage("You have no blink stones.")
		return false
	}
	landing, ok := g.safestBlinkLanding()
	if !ok {
		g.addMessage("The blink stone finds no room to land.")
		return false
	}
	g.player.BlinkStones--
	g.player.Pos = landing.player
	g.dogFocus = nil
	if g.dog.Freed && g.dog.Alive {
		g.dog.Pos = landing.dog
		g.addMessage("Space folds and carries you and Ghost Dog to the safest open spot on this floor.")
	} else {
		g.addMessage("Space folds and carries you to the safest open spot on this floor.")
	}
	g.collectItems()
	return true
}

func (g *game) useSunOrb() bool {
	if g.player.SunOrbs == 0 {
		g.addMessage("You have no sun orbs.")
		return false
	}
	g.player.SunOrbs--
	hit := 0
	for _, m := range g.current().Monsters {
		if m.HP > 0 && m.Undead {
			dmg := g.randRange(9, 14)
			g.damageMonster(m, dmg, "sears")
			hit++
		}
	}
	if hit == 0 {
		g.addMessage("The sun orb flashes, but no undead are here.")
	} else {
		g.addMessage(fmt.Sprintf("Sunfire crashes through %d undead foe(s).", hit))
	}
	return true
}

func (g *game) useFrostCharm() bool {
	if g.player.FrostCharms == 0 {
		g.addMessage("You have no frost charms.")
		return false
	}
	target := g.nearestMonster(6)
	if target == nil {
		g.addMessage("The frost charm finds no nearby target.")
		return false
	}
	g.player.FrostCharms--
	damage := g.randRange(7, 11)
	target.FrozenTurns = 2
	g.addMessage(fmt.Sprintf("The frost charm locks %s in rime for %d damage.", target.Name, damage))
	g.damageMonster(target, damage, "freezes")
	return true
}

func (g *game) useStarfireOrb() bool {
	if g.player.StarfireOrbs == 0 {
		g.addMessage("You have no starfire orbs.")
		return false
	}
	target := g.nearestMonster(6)
	if target == nil {
		g.addMessage("The starfire orb finds no nearby target.")
		return false
	}
	g.player.StarfireOrbs--
	damage := g.randRange(18, 24)
	g.addMessage(fmt.Sprintf("Starfire crashes into %s for %d damage.", target.Name, damage))
	g.damageMonster(target, damage, "sears")
	return true
}

func (g *game) usePhoenixAsh() bool {
	if g.player.PhoenixAshes == 0 {
		g.addMessage("You have no phoenix ash.")
		return false
	}
	if g.player.HP >= g.player.MaxHP {
		g.addMessage("You are already at full health; save the phoenix ash.")
		return false
	}
	g.player.PhoenixAshes--
	heal := g.randRange(22, 30)
	g.player.HP = min(g.player.MaxHP, g.player.HP+heal)
	g.addMessage(fmt.Sprintf("Phoenix ash restores your health. (%d/%d)", g.player.HP, g.player.MaxHP))
	return true
}

func (g *game) useGhostRecallScroll() bool {
	if g.player.GhostRecallScrolls == 0 {
		g.addMessage("You have no Ghost Dog recall scrolls.")
		return false
	}
	if !g.dog.Freed {
		g.addMessage("The scroll cannot call a Ghost Dog you have not freed.")
		return false
	}
	if g.dog.Alive {
		g.addMessage("The Ghost Dog is already at your side.")
		return false
	}
	g.dog.Alive = true
	if !g.placeDogNearPlayer() {
		g.dog.Alive = false
		g.addMessage("The scroll finds no safe place for the Ghost Dog to return.")
		return false
	}
	g.player.GhostRecallScrolls--
	g.dog.HP = g.dog.MaxHP
	g.addMessage("The recall scroll calls the Ghost Dog back to your side at full hit points.")
	return true
}

func (g *game) drinkFromFountain() bool {
	var nearby *fountain
	for i := range g.current().Fountains {
		f := &g.current().Fountains[i]
		if !f.Used && g.current().Tiles[f.Pos.Y][f.Pos.X] != '#' && distance(g.player.Pos, f.Pos) <= 1 {
			nearby = f
			break
		}
	}
	if nearby == nil {
		g.addMessage("There is no unused fountain close enough to drink from.")
		return false
	}
	if g.player.HP >= g.player.MaxHP && (!g.dog.Freed || !g.dog.Alive || g.dog.HP >= g.dog.MaxHP) {
		g.addMessage("You and Ghost Dog are already at full health; save the fountain.")
		return false
	}
	nearby.Used = true
	playerHeal := g.randRange(4, 8)
	oldHP := g.player.HP
	g.player.HP = min(g.player.MaxHP, g.player.HP+playerHeal)
	actualPlayerHeal := g.player.HP - oldHP
	if g.dog.Freed && g.dog.Alive {
		dogHeal := g.randRange(8, 12)
		oldDogHP := g.dog.HP
		g.dog.HP = min(g.dog.MaxHP, g.dog.HP+dogHeal)
		g.addMessage(fmt.Sprintf("The fountain restores %d health to Ghost Dog. (%d/%d)", g.dog.HP-oldDogHP, g.dog.HP, g.dog.MaxHP))
	}
	g.addMessage(fmt.Sprintf("You drink from the fountain and recover %d health. (%d/%d)", actualPlayerHeal, g.player.HP, g.player.MaxHP))
	return true
}
