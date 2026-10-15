package main

import "fmt"

func (g *game) applyWeaponPower(target *monster, dealt int) {
	if dealt <= 0 {
		return
	}
	switch g.player.Weapon.Power {
	case powerLifeDrain:
		healed := min(min(8, dealt/5), g.player.MaxHP-g.player.HP)
		if healed > 0 {
			g.player.HP += healed
			g.addMessage(fmt.Sprintf("Dawnbreaker restores %d health through stolen vitality.", healed))
		}
	case powerFrostStrike:
		if target.HP > 0 && g.rng.Intn(4) == 0 {
			target.FrozenTurns = max(target.FrozenTurns, 1)
			g.addMessage("Winter's Last Word seals the foe in ice.")
		}
	case powerChainLightning:
		var next *monster
		for _, m := range g.current().Monsters {
			if m == target || m.HP <= 0 || g.current().Tiles[m.Pos.Y][m.Pos.X] == '#' || distance(m.Pos, target.Pos) > 2 {
				continue
			}
			if next == nil || distance(m.Pos, target.Pos) < distance(next.Pos, target.Pos) {
				next = m
			}
		}
		if next != nil {
			g.addMessage("Storm Shepherd arcs lightning into " + next.Name + ".")
			g.damageMonster(next, 12, "scorches")
		}
	}
}

func (g *game) hexPlayer() {
	if g.player.hasArmorPower(powerClearMind) {
		g.addMessage("Oracle Crown scatters the curse.")
		return
	}
	g.player.HexedTurns = 1
}

func (g *game) godArmorReprisal(attacker *monster, beforeHP int) {
	if g.player.HP > 0 && g.player.HP < beforeHP && attacker.HP > 0 && g.player.hasArmorPower(powerReprisal) {
		g.addMessage("Starthorn Greaves lash out at " + attacker.Name + ".")
		g.damageMonster(attacker, 12, "hits")
	}
}
