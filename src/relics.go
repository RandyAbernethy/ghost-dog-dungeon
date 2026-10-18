package main

type gearPower string

const (
	powerSanctuary      gearPower = "sanctuary"
	powerClearMind      gearPower = "clear_mind"
	powerReprisal       gearPower = "reprisal"
	powerLifeDrain      gearPower = "life_drain"
	powerFrostStrike    gearPower = "frost_strike"
	powerShatter        gearPower = "shatter"
	powerChainLightning gearPower = "chain_lightning"
)

var godArmorPieces = [...]armor{
	{Name: "Aegis of the Last Dawn", Slot: slotBody, Defense: 24, SpellWard: 6, StrikeBonus: 2, Rarity: rarityGod, Power: powerSanctuary},
	{Name: "Oracle Crown", Slot: slotHead, Defense: 18, SpellWard: 12, StrikeBonus: 4, Rarity: rarityGod, Power: powerClearMind},
	{Name: "Starthorn Greaves", Slot: slotLeg, Defense: 18, SpellWard: 6, StrikeBonus: 3, Rarity: rarityGod, Power: powerReprisal},
}

var artifactWeapons = [...]weapon{
	{Name: "Dawnbreaker", Min: 30, Max: 42, Magic: true, Rarity: rarityArtifact, Power: powerLifeDrain},
	{Name: "Winter's Last Word", Min: 28, Max: 38, Magic: true, Reach: 2, Rarity: rarityArtifact, Power: powerFrostStrike},
	{Name: "Worldsplitter", Min: 38, Max: 52, Magic: true, Rarity: rarityArtifact, Power: powerShatter},
	{Name: "Storm Shepherd", Min: 30, Max: 40, Magic: true, Reach: 2, Rarity: rarityArtifact, Power: powerChainLightning},
}

func relicPool() []item {
	var pool []item
	for _, a := range godArmorPieces {
		pool = append(pool, makeArmorItem(a))
	}
	for _, w := range artifactWeapons {
		pool = append(pool, makeWeaponItem(w))
	}
	return pool
}

func relicItem(it item) bool {
	return it.Kind == itemArmor && it.Armor.Rarity == rarityGod || it.Kind == itemWeapon && it.Weapon.Rarity == rarityArtifact
}

func (lvl *level) hasRelic() bool {
	for _, it := range lvl.Items {
		if relicItem(it) {
			return true
		}
	}
	return false
}

func relicDepth(depth int) bool { return depth >= 6 && depth <= 8 }

// Relics replace ordinary treasure, preserving room space for the scroll and fountain.
// This is generation-only: neither discovery nor loading rolls for another reward.
func (g *game) addLinkedRelic(lvl *level) {
	if !relicDepth(lvl.Index) {
		return
	}
	roots := lvl.Secrets
	var laterRooms []*secretRoom
	for _, root := range roots {
		if root == nil {
			continue
		}
		for inner := root.Next; inner != nil; inner = inner.Next {
			laterRooms = append(laterRooms, inner)
		}
	}
	if len(laterRooms) == 0 {
		return
	}
	for _, floor := range g.levels {
		if floor == nil {
			continue
		}
		for _, it := range floor.Items {
			if relicItem(it) {
				return
			}
		}
	}
	if g.rng.Intn(100) >= 25 {
		return
	}
	var slots []int
	for _, hidden := range laterRooms {
		for _, p := range hidden.Tiles {
			for i, it := range lvl.Items {
				if it.Pos == p && it.Kind != itemSideStoryScroll && it.Kind != itemStoryScroll {
					slots = append(slots, i)
				}
			}
		}
	}
	if len(slots) == 0 {
		return
	}
	pool := g.filterFoundSecretEquipment(relicPool())
	if len(pool) == 0 {
		return
	}
	i := slots[g.rng.Intn(len(slots))]
	relic := pool[g.rng.Intn(len(pool))]
	relic.Pos = lvl.Items[i].Pos
	lvl.Items[i] = relic
}

func gearPowerDescription(power gearPower) string {
	switch power {
	case powerSanctuary:
		return "Sanctuary: living Ghost Dog takes 6 less damage while within two tiles of you."
	case powerClearMind:
		return "Clear mind: immune to curses and entanglement; equipping clears existing hexes and bindings."
	case powerReprisal:
		return "Starthorn reprisal: a surviving melee hit returns 12 damage to its attacker."
	case powerLifeDrain:
		return "Life drain: heals you for one fifth of the damage dealt, up to 8 health per strike."
	case powerFrostStrike:
		return "Winter's grasp: each hit has a 25% chance to freeze a surviving target for its next turn."
	case powerShatter:
		return "Worldshatter: your strikes ignore all monster armor."
	case powerChainLightning:
		return "Chain lightning: a hit also blasts one other monster within two tiles of the target for 12 damage."
	}
	return ""
}

func (p *player) hasArmorPower(power gearPower) bool {
	return p.HeadArmor.Power == power || p.BodyArmor.Power == power || p.LegArmor.Power == power
}

func (p *player) hasGodArmor() bool {
	return p.HeadArmor.Rarity == rarityGod || p.BodyArmor.Rarity == rarityGod || p.LegArmor.Rarity == rarityGod
}
