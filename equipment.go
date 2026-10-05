package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func weaponDescription(w weapon) string {
	if w.effectiveReach() > 1 {
		return fmt.Sprintf("Reach %d: strike along a clear movement direction without stepping closer. Trades close-range damage for distance.", w.effectiveReach())
	}
	switch w.Name {
	case "Rusty Knife":
		return "A chipped backup blade, quick but weak."
	case "Short Sword":
		return "A balanced edge that makes early fights much safer."
	case "Iron Spear":
		return "A sturdy reach weapon that hits a little harder every swing."
	case "Chapel Mace":
		return "Blessed iron that caves in skulls better than your starting knife."
	case "Hooked Glaive":
		return "A polearm with a wicked bite, good for steady deep-floor damage."
	case "Battle Axe":
		return "Heavy steel that can end weaker monsters in a few brutal chops."
	case "Moonblade":
		return "A silvered sword that feels made for cursed halls and undead flesh."
	case "Dawnstar Flail":
		return "A brutal chain-weapon that thrives in tight rooms and ugly melees."
	case "Rune Saber":
		return "Its etched edge holds a steady killing line against elite monsters."
	case "Gravecleaver":
		return "A heavy cleaver made for the armored dead below."
	case "Hollowfang Spear":
		return "A barbed spear that keeps deep-floor horrors at a careful distance."
	case "Dragontooth Pike":
		return "A savage relic with enough force to carry you into the middle depths."
	case "Starforged Hammer":
		return "A dense hammer whose star-forged head breaks through mid-dungeon foes."
	case "Emberbrand":
		return "A fire-rune blade whose magic burns brighter against undead foes."
	case "Tempest Spear":
		return "A storm-charged spear that strikes with magical force against undead."
	case "Voidglass Dagger":
		return "A black-glass blade that cuts through undead essence as well as flesh."
	case "Lichbane Greatsword":
		return "An ancient enchanted greatsword made to bring down undead tyrants."
	case "Frostbite Axe":
		return "A deep-floor axe forged to bite through thick hides and armor."
	case "Gloomsteel Saber":
		return "Dark steel with a keen edge, tempered beneath the oldest halls."
	case "Graveglass Halberd":
		return "A long, heavy polearm built to keep deep-floor horrors at bay."
	case "Stormcaller Blade":
		return "A balanced blade that carries a sharp crack through every strike."
	case "Wyrmheart Maul":
		return "A dense war-maul made for creatures that shrug off ordinary blows."
	case "Gloaming Pike":
		return "A dark-tipped pike with enough force to pierce the lich's guard."
	case "Dawnforged Greatsword":
		return "A deep-floor greatsword with a bright, punishing edge."
	case "Kingsbane Axe":
		return "A heavy axe whose broad head can split even a monster's guard."
	case "Gravetide Maul":
		return "A secret relic that releases a burst of force against undead foes."
	case "Star-Eater Blade":
		return "A mythic secret blade that tears through undead and mortal alike."
	default:
		return fmt.Sprintf("Reliable steel dealing %d-%d damage.", w.Min, w.Max)
	}
}

func armorDescription(a armor) string {
	if a.SpellWard > 0 || a.StrikeBonus > 0 {
		return armorTraits(a) + " Trades physical protection for its special benefit."
	}
	switch a.Name {
	case "Padded Hood":
		return "Thin padding for your skull; barely enough to stop a lucky scrape."
	case "Worn Coat":
		return "An old coat that still turns knives and claws a little."
	case "Frayed Boots":
		return "Almost no protection, but better than bare feet on dungeon stone."
	case "Leather Jerkin":
		return "Tough hide stitched for dungeon delvers who expect real hits."
	case "Scout Cap":
		return "Light headgear with just enough structure to turn away a nasty cut."
	case "Iron Helm":
		return "A dented helm that makes curses and clubs feel less final."
	case "Wolfhide Boots":
		return "Fur-lined boots that keep your footing when the halls turn murderous."
	case "Reinforced Greaves":
		return "Plated shins and boots that steady you against lunging beasts."
	case "Scale Coat":
		return "Linked scales spread hard blows across your chest and shoulders."
	case "Knight Mail":
		return "Layered steel rings that turn many brutal blows into survivable ones."
	case "Warden Crown":
		return "An ancient circlet that hardens your brow and your will together."
	case "Nightstride Boots":
		return "Quiet, heavy boots built for long marches through murderous halls."
	case "Sunplate Cuirass":
		return "Radiant plate that makes the last floor feel much less final."
	case "Dreadmark Mantle":
		return "A reinforced mantle that keeps deep-floor blows from landing cleanly."
	case "Gloamrunner Boots":
		return "Quiet boots stitched for long runs through hostile halls."
	case "Aetherweave Hood":
		return "A tightly woven hood that guards against steel and strange magic."
	case "Ashguard Coat":
		return "A soot-black coat that turns aside the worst blows from below."
	case "Stormscale Greaves":
		return "Layered greaves that brace your legs against heavy strikes."
	case "Graveward Helm":
		return "A sealed helm that holds firm against both blades and curses."
	case "Ruinplate Vest":
		return "A late-floor cuirass built from plates salvaged from a fallen citadel."
	case "Mourner's Helm":
		return "A deep-floor helm that guards your head through the worst encounters."
	case "Starforged Cuirass":
		return "Rare star-metal plate with exceptional protection for the deepest floor."
	case "Voidwalker Boots":
		return "Rare boots that steady every step through the dungeon's oldest reaches."
	case "Aegis of Echoes":
		return "Secret-forged armor that turns aside attacks with a lingering ward."
	case "Crown of the Hollow Star":
		return "A secret circlet that shelters its wearer from brutal blows."
	case "Wraithstep Greaves":
		return "Secret greaves that protect without slowing a careful retreat."
	case "Voidheart Plate":
		return "The dungeon's rarest armor, hidden where the oldest powers sleep."
	default:
		return fmt.Sprintf("Protective gear for your %s slot worth %d guard.", slotLabel(a.Slot), a.Defense)
	}
}

func compareWeapon(candidate, current weapon) string {
	return fmt.Sprintf("Compare to %s: %+d min, %+d max damage, %+d reach, %+d undead bonus.", current.Name, candidate.Min-current.Min, candidate.Max-current.Max, candidate.effectiveReach()-current.effectiveReach(), weaponUndeadBonus(candidate)-weaponUndeadBonus(current))
}

func compareArmor(candidate, current armor) string {
	return fmt.Sprintf("Compare to %s: %+d guard, %+d spell ward, %+d strike damage in the %s slot.", current.Name, candidate.Defense-current.Defense, candidate.SpellWard-current.SpellWard, candidate.StrikeBonus-current.StrikeBonus, slotLabel(candidate.Slot))
}

func rarityLabel(r rarity) string {
	switch r {
	case rarityCommon:
		return "common"
	case rarityUncommon:
		return "uncommon"
	case rarityRare:
		return "rare"
	case rarityEpic:
		return "epic"
	case rarityLegendary:
		return "legendary"
	default:
		return string(r)
	}
}

func rarityColor(r rarity) string {
	switch r {
	case rarityCommon:
		return "\x1b[37m"
	case rarityUncommon:
		return "\x1b[32m"
	case rarityRare:
		return "\x1b[34m"
	case rarityEpic:
		return "\x1b[35m"
	case rarityLegendary:
		return "\x1b[33m"
	default:
		return ""
	}
}

func coloredArmorName(a armor) string {
	prefix := strings.Title(rarityLabel(a.Rarity)) + " " + a.Name
	if color := rarityColor(a.Rarity); color != "" {
		return color + prefix + ansiReset
	}
	return prefix
}

func slotLabel(slot armorSlot) string {
	switch slot {
	case slotHead:
		return "head"
	case slotBody:
		return "body"
	case slotLeg:
		return "leg"
	default:
		return string(slot)
	}
}

func (p *player) armorForSlot(slot armorSlot) armor {
	switch slot {
	case slotHead:
		return p.HeadArmor
	case slotBody:
		return p.BodyArmor
	case slotLeg:
		return p.LegArmor
	default:
		return armor{}
	}
}

func (p *player) setArmor(a armor) {
	switch a.Slot {
	case slotHead:
		p.HeadArmor = a
	case slotBody:
		p.BodyArmor = a
	case slotLeg:
		p.LegArmor = a
	}
}

func (p *player) armorDefense() int {
	return p.HeadArmor.Defense + p.BodyArmor.Defense + p.LegArmor.Defense
}

func (p *player) armorBlock() int { return p.armorDefense() / 2 }

var reachWeaponNames = [...]string{
	"Ashwood Spear", "Chapel Lance", "Bone-Tipped Pike", "Silver Glaive", "Ember Lance",
	"Gravewood Pike", "Frostglass Spear", "Stormreach Glaive", "Voidsteel Lance", "Dawnwatch Pike",
}

func (w weapon) effectiveReach() int { return max(1, w.Reach) }

func weaponUndeadBonus(w weapon) int {
	if w.Magic {
		return 2
	}
	return 0
}

func (p *player) spellWard() int {
	return p.HeadArmor.SpellWard + p.BodyArmor.SpellWard + p.LegArmor.SpellWard
}

func (p *player) strikeBonus() int {
	return p.HeadArmor.StrikeBonus + p.BodyArmor.StrikeBonus + p.LegArmor.StrikeBonus
}

func (p *player) carryWeapon(w weapon) {
	for _, carried := range p.Weapons {
		if carried == w {
			return
		}
	}
	p.Weapons = append(p.Weapons, w)
}

func (p *player) carryArmor(a armor) {
	for _, carried := range p.Armors {
		if carried == a {
			return
		}
	}
	p.Armors = append(p.Armors, a)
}

func (p *player) ensureCarriedEquipment() {
	p.carryWeapon(p.Weapon)
	for _, a := range []armor{p.HeadArmor, p.BodyArmor, p.LegArmor} {
		p.carryArmor(a)
	}
}

func (g *game) equipCommand(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) != 2 {
		g.addMessage("Use equip NUMBER from the inventory; in immediate-key mode, type :equip NUMBER and Enter.")
		return false
	}
	n, err := strconv.Atoi(fields[1])
	g.player.ensureCarriedEquipment()
	if err != nil || n < 1 || n > len(g.player.Weapons)+len(g.player.Armors) {
		g.addMessage("Choose an equipment number shown in your inventory.")
		return false
	}
	if n <= len(g.player.Weapons) {
		w := g.player.Weapons[n-1]
		if g.player.Weapon == w {
			g.addMessage("You already have that weapon equipped.")
			return false
		}
		g.player.Weapon = w
		g.addMessage(fmt.Sprintf("You equip %s: %d-%d damage, reach %d. Swapping equipment takes a turn.", w.Name, w.Min, w.Max, w.effectiveReach()))
	} else {
		a := g.player.Armors[n-len(g.player.Weapons)-1]
		if g.player.armorForSlot(a.Slot) == a {
			g.addMessage("You already have that armor equipped.")
			return false
		}
		g.player.setArmor(a)
		g.addMessage(fmt.Sprintf("You equip %s. %s Swapping equipment takes a turn.", a.Name, armorTraits(a)))
	}
	return true
}

func (g *game) dropCommand(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) != 2 {
		g.addMessage("Use :drop NUMBER + Enter to drop an item numbered in your inventory.")
		return false
	}
	n, err := strconv.Atoi(fields[1])
	g.player.ensureCarriedEquipment()
	lines := g.inventoryLines()
	gearCount := len(g.player.Weapons) + len(g.player.Armors)
	if err != nil || n < 1 || n > gearCount+len(lines) {
		g.addMessage("Choose an item number shown in your inventory.")
		return false
	}
	var dropped item
	switch {
	case n <= len(g.player.Weapons):
		i := n - 1
		w := g.player.Weapons[i]
		if w == g.player.Weapon {
			g.addMessage("You cannot drop an equipped item. Equip something else first.")
			return false
		}
		dropped = makeWeaponItem(w)
		g.player.Weapons = append(g.player.Weapons[:i], g.player.Weapons[i+1:]...)
	case n <= gearCount:
		i := n - len(g.player.Weapons) - 1
		a := g.player.Armors[i]
		if a == g.player.armorForSlot(a.Slot) {
			g.addMessage("You cannot drop an equipped item. Equip something else first.")
			return false
		}
		dropped = makeArmorItem(a)
		g.player.Armors = append(g.player.Armors[:i], g.player.Armors[i+1:]...)
	default:
		line := lines[n-gearCount-1]
		if *line.quantity <= 0 {
			g.addMessage("You have none of that item to drop.")
			return false
		}
		*line.quantity--
		dropped = item{Kind: line.kind, Name: line.name, Glyph: line.glyph}
	}
	dropped.Pos = g.player.Pos
	g.current().Items = append(g.current().Items, dropped)
	g.addMessage(fmt.Sprintf("You drop %s on the floor. Dropping takes a turn.", dropped.Name))
	return true
}

func (g *game) renderEquipment(w io.Writer) {
	g.player.ensureCarriedEquipment()
	fmt.Fprintln(w, "\nCarried equipment (equip NUMBER or drop NUMBER; in immediate-key mode prefix with : and press Enter):")
	for i, carried := range g.player.Weapons {
		active := ""
		if carried == g.player.Weapon {
			active = " [equipped]"
		}
		magic := ""
		if carried.Magic {
			magic = ", +2 damage against undead"
		}
		fmt.Fprintf(w, "%d. %s: %d-%d damage, reach %d%s%s\n", i+1, carried.Name, carried.Min, carried.Max, carried.effectiveReach(), magic, active)
	}
	for i, carried := range g.player.Armors {
		active := ""
		if carried == g.player.armorForSlot(carried.Slot) {
			active = " [equipped]"
		}
		fmt.Fprintf(w, "%d. %s (%s): %s%s\n", len(g.player.Weapons)+i+1, coloredArmorName(carried), slotLabel(carried.Slot), armorTraits(carried), active)
	}
	fmt.Fprintln(w, "Reach 2: press a movement direction to strike a foe two tiles away along a clear line. Equipping takes one turn.")
}

func armorTraits(a armor) string {
	return fmt.Sprintf("%d guard, %d spell ward, %+d strike damage.", a.Defense, a.SpellWard, a.StrikeBonus)
}
