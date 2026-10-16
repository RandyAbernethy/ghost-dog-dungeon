package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExpandedLootKeepsPowerAndOffersOptionalFinds(t *testing.T) {
	// These pool averages are the calibrated ordinary-loot baseline. Compare
	// outcomes independently of catalog length, ordering, or variant pairing.
	weaponDamage := [...]int{31, 33, 44, 51, 55, 63, 75, 76, 81, 95} // Sum for three baseline choices.
	weaponReach := [...]int{4, 5, 4, 4, 5, 5, 4, 5, 5, 4}
	armorTraits := [...][3]int{
		{5, 2, 1}, {4, 2, 1}, {6, 2, 1}, {9, 2, 1}, {10, 3, 2},
		{10, 3, 2}, {13, 3, 2}, {13, 3, 2}, {15, 4, 3}, {17, 4, 3},
	} // Guard, ward, and strike totals for four baseline choices.
	for depth := 0; depth < levelCount; depth++ {
		weapons, armors := weaponPoolForLevel(depth), armorPoolForLevel(depth)
		if len(weapons) < 2 || len(armors) < 2 {
			t.Fatal("ordinary loot should offer multiple choices")
		}
		names := map[string]bool{}
		damage, reach := 0, 0
		for _, w := range weapons {
			if names[w.Name] || w.Name == "" || w.Min < 1 || w.Min > w.Max || w.Magic || w.Rarity == rarityArtifact {
				t.Fatalf("floor %d has invalid or duplicate ordinary weapon %+v", depth+1, w)
			}
			names[w.Name] = true
			damage += w.Min + w.Max
			reach += w.effectiveReach()
		}
		if damage*3 != weaponDamage[depth]*len(weapons) || reach*3 != weaponReach[depth]*len(weapons) {
			t.Fatalf("floor %d changes calibrated average weapon damage or reach", depth+1)
		}
		names = map[string]bool{}
		var totals [3]int
		for _, a := range armors {
			if names[a.Name] || a.Name == "" || a.Defense < 0 || a.SpellWard < 0 || a.StrikeBonus < 0 || a.Rarity == rarityGod ||
				(a.Slot != slotHead && a.Slot != slotBody && a.Slot != slotLeg) {
				t.Fatalf("floor %d has invalid or duplicate ordinary armor %+v", depth+1, a)
			}
			names[a.Name] = true
			totals[0] += a.Defense
			totals[1] += a.SpellWard
			totals[2] += a.StrikeBonus
		}
		for trait, total := range totals {
			if total*4 != armorTraits[depth][trait]*len(armors) {
				t.Fatalf("floor %d changes calibrated average armor trait %d", depth+1, trait)
			}
		}
		g := &game{rng: newSimpleRNG(int64(depth + 10))}
		foundWeapons, foundArmor := map[string]bool{}, map[string]bool{}
		for i := 0; i < 1024; i++ {
			foundWeapons[g.weaponLootForLevel(depth).Name] = true
			foundArmor[g.armorLootForLevel(depth).Name] = true
		}
		if len(foundWeapons) != len(weapons) || len(foundArmor) != len(armors) {
			t.Fatal("every pool entry must be possible")
		}
	}
	for _, name := range []string{additionalWeaponNames[0][0], additionalArmorNames[0][0]} {
		present, absent := false, false
		for seed := int64(1); seed <= 30; seed++ {
			g := newGame(seed)
			found := false
			for _, lvl := range g.levels {
				for _, it := range lvl.Items {
					if it.Name == name {
						found = true
					}
				}
			}
			present = present || found
			absent = absent || !found
		}
		if !present || !absent {
			t.Fatalf("%s should be found in some runs and absent in others", name)
		}
	}
}

func TestAdditionalEquipmentCanBeCarriedEquippedAndSaved(t *testing.T) {
	g := openArena()
	g.saveDir = filepath.Dir(filepath.Join(t.TempDir(), "gear.json"))
	w := makeWeaponLoot(weaponPoolForLevel(0)[3])
	a := makeArmorItem(armorPoolForLevel(0)[4])
	g.collectItem(w)
	g.collectItem(a)
	weaponNumber := len(g.player.Weapons)
	armorNumber := weaponNumber + len(g.player.Armors)
	g.processCommand(fmt.Sprintf("equip %d", weaponNumber))
	g.processCommand(fmt.Sprintf("equip %d", armorNumber))
	if g.stats.Turns != 2 {
		t.Fatal("new equipment should be usable through inventory numbers")
	}
	if g.player.Weapon != w.Weapon || g.player.armorForSlot(a.Armor.Slot) != a.Armor {
		t.Fatal("equipped new gear should keep its generated stats")
	}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.player, loaded.player) {
		t.Fatal("saving must keep carried and equipped new gear")
	}
}
