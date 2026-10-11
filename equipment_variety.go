package main

var additionalWeaponNames = [...][3]string{
	{"Watchman's Saber", "Stonebreaker Club", "Pilgrim's Staff"},
	{"Tombwatch Spear", "Butcher's Billhook", "Lantern Pike"},
	{"Blacksmith's Axe", "Chainbreaker Flail", "Sentinel's Lance"},
	{"Crescent Falchion", "Runebound Cutlass", "Chapel Glaive"},
	{"Dragonbone Partisan", "Bellringer Hammer", "Cinder Pike"},
	{"Executioner's Cleaver", "Thornhook Spear", "Mourning Lance"},
	{"Rimefang Hatchet", "Nightwarden Saber", "Winterwatch Pike"},
	{"Cryptglass Voulge", "Thunderbrand Sword", "Tempest Glaive"},
	{"Dragonwake Maul", "Duskwatch Partisan", "Eclipse Lance"},
	{"Sunrise Claymore", "Crownsplitter Axe", "Lastwatch Pike"},
}

var additionalArmorNames = [...][4]string{
	{"Quilted Surcoat", "Pilgrim's Cap", "Candleweave Hood", "Tracker's Hood"},
	{"Riveted Sallet", "Ranger's Greaves", "Chantwoven Vest", "Skirmisher's Vest"},
	{"Tombwatch Greaves", "Ringmail Coat", "Sanctuary Greaves", "Trapper's Greaves"},
	{"Lamellar Hauberk", "Chapelguard Boots", "Psalmwoven Hood", "Falconer's Hood"},
	{"Dawnwatch Crown", "Gilded Breastplate", "Cinderweave Robe", "Outrider's Coat"},
	{"Cryptwarden Mantle", "Gravestep Boots", "Spiritwoven Greaves", "Houndkeeper's Greaves"},
	{"Winterveil Hood", "Rimeguard Coat", "Frostscript Hood", "Pathfinder's Hood"},
	{"Thunderstep Greaves", "Stormwatch Helm", "Stormscript Vest", "Ravenguard Vest"},
	{"Dragonward Vest", "Eclipse Helm", "Moonweave Greaves", "Shadehunter Greaves"},
	{"Dawnguard Plate", "Kingsguard Greaves", "Lastlight Hood", "Lichhunter Hood"},
}

func expandedWeapons(depth int, base []weapon) []weapon {
	// Preserve each original option's average damage and reach frequency.
	for i, original := range append([]weapon(nil), base...) {
		variant := original
		variant.Name = additionalWeaponNames[depth][i]
		if i == 1 {
			variant.Min--
			variant.Max++
		} else {
			variant.Min++
			variant.Max--
		}
		base = append(base, variant)
	}
	return base
}

func expandedArmor(depth int, base []armor) []armor {
	// Paired shifts keep average guard, wards, and strike bonuses unchanged.
	for i, original := range append([]armor(nil), base...) {
		variant := original
		variant.Name = additionalArmorNames[depth][i]
		switch i {
		case 0:
			variant.Defense--
			variant.SpellWard++
		case 2:
			variant.Defense++
			variant.SpellWard--
		case 1:
			if base[3].StrikeBonus > 1 {
				variant.Defense--
				variant.StrikeBonus++
			}
		case 3:
			if original.StrikeBonus > 1 {
				variant.Defense++
				variant.StrikeBonus--
			}
		}
		base = append(base, variant)
	}
	return base
}
