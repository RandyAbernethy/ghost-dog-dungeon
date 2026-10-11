package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExpandedLootKeepsPowerAndOffersOptionalFinds(t *testing.T) {
	for depth := 0; depth < levelCount; depth++ {
		weapons, armors := weaponPoolForLevel(depth), armorPoolForLevel(depth)
		if len(weapons) != 6 || len(armors) != 8 {
			t.Fatal("floor should have six weapon choices and eight armor choices")
		}
		for i, original := range weapons[:3] {
			variant := weapons[i+3]
			if original.Name == variant.Name || original.Min+original.Max != variant.Min+variant.Max || original.effectiveReach() != variant.effectiveReach() || original.Magic != variant.Magic || variant.Min < 1 || variant.Min > variant.Max {
				t.Fatalf("floor %d changes average weapon power or reach", depth+1)
			}
			if original.Min == variant.Min || original.Max == variant.Max {
				t.Fatal("new weapon should offer a different damage spread")
			}
		}
		var originalTotals, variantTotals [3]int
		for i, original := range armors[:4] {
			variant := armors[i+4]
			if original.Name == variant.Name {
				t.Fatal("new armor should have its own identity")
			}
			if variant.Slot != original.Slot || variant.Rarity != original.Rarity || variant.Defense < 0 || variant.SpellWard < 0 || variant.StrikeBonus < 0 {
				t.Fatal("armor additions must preserve slots and rarities and have nonnegative traits")
			}
			originalTotals[0] += original.Defense
			originalTotals[1] += original.SpellWard
			originalTotals[2] += original.StrikeBonus
			variantTotals[0] += variant.Defense
			variantTotals[1] += variant.SpellWard
			variantTotals[2] += variant.StrikeBonus
		}
		if originalTotals != variantTotals {
			t.Fatal("armor choices should retain the pool's average guard, ward, and strike bonuses")
		}
		if armors[4].Defense >= armors[0].Defense || armors[4].SpellWard <= armors[0].SpellWard || armors[6].Defense <= armors[2].Defense || armors[6].SpellWard >= armors[2].SpellWard {
			t.Fatal("new armor should trade guard for spell protection in both directions")
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
	g.saveFile, g.timestampedSaves = filepath.Join(t.TempDir(), "gear.json"), false
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
