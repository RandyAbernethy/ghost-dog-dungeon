package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDropRejectsEquippedMissingAndInvalidItems(t *testing.T) {
	g := openArena()
	before := g.player
	for _, command := range []string{"drop 1", "drop 2", "drop 3", "drop 4", "drop", "drop nope", "drop 0", "drop -1", "drop 14", "drop 9", "drop 1 extra"} {
		g.processCommand(command)
		if !reflect.DeepEqual(g.player, before) || len(g.current().Items) != 0 || g.stats.Turns != 0 {
			t.Fatalf("rejected command %q changed state or spent a turn", command)
		}
	}
}

func TestDropGearSpendsTurnPersistsAndCanBePickedUp(t *testing.T) {
	g := openArena()
	spear := weapon{Name: "Test Spear", Min: 3, Max: 5, Reach: 2}
	g.player.carryWeapon(spear)
	attacker := newMonster(monsterOrc)
	attacker.Pos = pos{6, 5}
	g.current().Monsters = []*monster{&attacker}
	g.processCommand("drop 2")
	if len(g.player.Weapons) != 1 || g.player.Weapon.Name != "Rusty Knife" || g.stats.Turns != 1 || g.player.HP == g.player.MaxHP {
		t.Fatal("dropping unused gear should keep equipped gear and let enemies take a turn")
	}
	if len(g.current().Items) != 1 || g.current().Items[0].Weapon != spear || g.current().Items[0].Pos != g.player.Pos {
		t.Fatal("dropped weapon lost its properties or did not land on the player's tile")
	}
	g.current().Monsters = nil
	g.saveFile, g.timestampedSaves = filepath.Join(t.TempDir(), "drop.json"), false
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.player.Weapons) != 1 {
		t.Fatal("loading should not put dropped gear back in the pack")
	}
	loaded.processCommand("d")
	loaded.processCommand("a")
	if len(loaded.player.Weapons) != 2 || loaded.player.Weapons[1] != spear {
		t.Fatal("walking back onto a dropped weapon should restore it to the pack")
	}
	for _, it := range loaded.current().Items {
		if it.Kind == itemWeapon && it.Weapon == spear {
			t.Fatal("picking gear back up should remove it from the floor")
		}
	}
}

func TestDropArmorProtectsItsEquippedSlot(t *testing.T) {
	g := openArena()
	ward := armor{Name: "Test Ward", Slot: slotHead, SpellWard: 3, Rarity: rarityRare}
	g.player.carryArmor(ward)
	g.processCommand("drop 5")
	if len(g.player.Armors) != 3 || g.current().Items[0].Armor != ward || g.player.HeadArmor.Name != "Padded Hood" {
		t.Fatal("dropping spare armor should preserve its traits and the equipped slot")
	}
	g.collectItems()
	g.processCommand("equip 5")
	turns := g.stats.Turns
	g.processCommand("drop 5")
	if g.player.HeadArmor != ward || len(g.player.Armors) != 4 || g.stats.Turns != turns {
		t.Fatal("newly equipped armor should be protected from dropping")
	}
}

func TestDropEachConsumableReleasesOneAndCanBeRecovered(t *testing.T) {
	for i := 0; i < len(openArena().inventoryLines()); i++ {
		g := openArena()
		line := g.inventoryLines()[i]
		t.Run(line.name, func(t *testing.T) {
			*line.quantity = 2
			n := len(g.player.Weapons) + len(g.player.Armors) + i + 1
			g.processCommand(fmt.Sprintf("drop %d", n))
			if *line.quantity != 1 || len(g.current().Items) != 1 || g.current().Items[0].Kind != line.kind || g.current().Items[0].Glyph != line.glyph || g.stats.Turns != 1 {
				t.Fatal("dropping a consumable should release exactly one of the numbered kind")
			}
			g.processCommand("d")
			g.processCommand("a")
			if *line.quantity != 2 || len(g.current().Items) != 0 {
				t.Fatal("returning to a dropped consumable should restore the stack")
			}
		})
	}
}

func TestDropNumbersFollowInventoryAfterGearRemoval(t *testing.T) {
	g := openArena()
	g.player.carryWeapon(weapon{Name: "Spare Sword", Min: 5, Max: 8})
	g.processCommand("drop 2")
	n := len(g.player.Weapons) + len(g.player.Armors) + 1
	potions := g.player.Potions
	g.processCommand(fmt.Sprintf("drop %d", n))
	if g.player.Potions != potions-1 || !strings.Contains(g.messages[len(g.messages)-1], "Healing Potion") {
		t.Fatal("consumable numbering should follow the current gear list")
	}
}
