package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGhostGiftsVaryAcrossAllCategoriesWithinFloorLootRanges(t *testing.T) {
	allowed := map[itemKind]bool{
		itemArmor: true, itemWeapon: true, itemPotion: true, itemBlinkStone: true,
		itemFireScroll: true, itemWardingCharm: true, itemSunOrb: true, itemFrostCharm: true,
		itemStarfireOrb: true, itemPhoenixAsh: true, itemGhostRecallScroll: true,
	}
	for depth := 0; depth < levelCount; depth++ {
		weapons, armors := map[weapon]bool{}, map[armor]bool{}
		for seed := int64(1); seed <= 80; seed++ {
			g := &game{rng: newSimpleRNG(seed)}
			for _, loot := range g.itemsForLevel(depth) {
				if loot.Kind == itemWeapon {
					weapons[loot.Weapon] = true
				}
				if loot.Kind == itemArmor {
					armors[loot.Armor] = true
				}
			}
		}
		seen := map[itemKind]bool{}
		for seed := int64(1); seed <= 120; seed++ {
			g := &game{rng: newSimpleRNG(seed)}
			offer := g.rollGhostGifts(depth)
			if len(offer) != 3 {
				t.Fatalf("floor %d ghost offered %d gifts", depth+1, len(offer))
			}
			kinds := map[itemKind]bool{}
			for _, gift := range offer {
				if !allowed[gift.Kind] || kinds[gift.Kind] || gift.Name == "" || gift.Glyph == 0 {
					t.Fatalf("floor %d seed %d has an invalid or duplicate gift: %+v", depth+1, seed, gift)
				}
				if gift.Kind == itemWeapon && !weapons[gift.Weapon] || gift.Kind == itemArmor && !armors[gift.Armor] {
					t.Fatalf("floor %d offers equipment outside its normal loot range: %+v", depth+1, gift)
				}
				kinds[gift.Kind], seen[gift.Kind] = true, true
			}
			repeat := (&game{rng: newSimpleRNG(seed)}).rollGhostGifts(depth)
			if !reflect.DeepEqual(offer, repeat) {
				t.Fatal("a seed must reproduce the three exact gifts and their order")
			}
		}
		if len(seen) != len(allowed) {
			t.Fatalf("floor %d did not offer all requested categories across seeds: %v", depth+1, seen)
		}
	}
}

func TestEveryGhostGiftCanBeClaimedIntoThePackExactlyOnce(t *testing.T) {
	for _, gift := range []item{
		makeArmorItem(armor{Name: "Test Ward", Slot: slotBody, Defense: 1, SpellWard: 3, Rarity: rarityRare}),
		makeWeaponItem(weapon{Name: "Test Spear", Min: 3, Max: 7, Reach: 2}),
		{Kind: itemPotion, Name: "healing potion"},
		{Kind: itemBlinkStone, Name: "blink stone"},
		{Kind: itemFireScroll, Name: "fire scroll"},
		{Kind: itemWardingCharm, Name: "warding charm"},
		{Kind: itemSunOrb, Name: "sun orb"},
		{Kind: itemFrostCharm, Name: "frost charm"},
		{Kind: itemStarfireOrb, Name: "starfire orb"},
		{Kind: itemPhoenixAsh, Name: "phoenix ash"},
		{Kind: itemGhostRecallScroll, Name: "Ghost Dog recall scroll"},
	} {
		t.Run(gift.Name, func(t *testing.T) {
			g := openArena()
			e := testFriendlyGhost(pos{6, 5})
			index := int(gift.Kind) % 3
			e.Gifts[index] = gift
			g.current().Events = []floorEvent{e}
			// Claiming a gift must not collect unrelated ground loot or use the gift.
			g.current().Items = []item{{Kind: itemStoryScroll, StoryChapter: 1, Pos: g.player.Pos}}
			g.player.HP--
			g.dog.Freed, g.dog.Alive, g.dog.HP = true, false, 0
			before := g.player
			counts := map[itemKind]int{}
			for _, line := range g.inventoryLines() {
				counts[line.kind] = *line.quantity
			}
			g.processCommand(fmt.Sprintf("choose %d", index+1))
			if g.stats.Turns != 1 || !g.current().Events[0].Used || g.player.HP != before.HP || g.dog.Alive {
				t.Fatal("claiming should spend one turn, end the offer, and keep healing/revival for item use")
			}
			if g.player.Weapon != before.Weapon || g.player.HeadArmor != before.HeadArmor || g.player.BodyArmor != before.BodyArmor || g.player.LegArmor != before.LegArmor {
				t.Fatal("gift equipment should enter the pack without replacing equipped gear")
			}
			weaponCount, armorCount := len(before.Weapons), len(before.Armors)
			if gift.Kind == itemWeapon {
				weaponCount++
				if g.player.Weapons[weaponCount-1] != gift.Weapon {
					t.Fatal("the packed weapon differs from the offer")
				}
			}
			if gift.Kind == itemArmor {
				armorCount++
				if g.player.Armors[armorCount-1] != gift.Armor {
					t.Fatal("the packed armor differs from the offer")
				}
			}
			if len(g.player.Weapons) != weaponCount || len(g.player.Armors) != armorCount {
				t.Fatal("the ghost should grant only the selected equipment")
			}
			for _, line := range g.inventoryLines() {
				want := counts[line.kind]
				if line.kind == gift.Kind {
					want++
				}
				if *line.quantity != want {
					t.Fatalf("%s quantity=%d, want %d", line.name, *line.quantity, want)
				}
			}
			if len(g.player.StoryChapters) != 0 || len(g.current().Items) != 1 {
				t.Fatal("a ghost gift should not consume ground loot")
			}
			claimed := g.player
			for _, cmd := range []string{"choose 1", "choose 2", "choose 3", "y"} {
				g.processCommand(cmd)
			}
			if !reflect.DeepEqual(claimed, g.player) || g.stats.Turns != 1 {
				t.Fatal("the ghost should give exactly one gift, regardless of subsequent choices")
			}
		})
	}
}

func TestReadingGhostChoicesAndInvalidChoicesDoNotChangeTheOffer(t *testing.T) {
	g := openArena()
	g.current().Events = []floorEvent{{Kind: eventGhost, Pos: pos{6, 5}, Gifts: g.rollGhostGifts(0)}}
	before, rngBefore := g.player, g.rng.State
	offer := append([]item(nil), g.current().Events[0].Gifts...)
	for i := 0; i < 2; i++ {
		g.messages = nil
		g.processCommand("y")
		for index, gift := range offer {
			if !strings.Contains(strings.Join(g.messages, " "), fmt.Sprintf("%d. %s", index+1, inspectItemText(gift))) {
				t.Fatalf("the offer must show numbered item descriptions: %v", g.messages)
			}
		}
		if !strings.Contains(strings.Join(g.messages, " "), ":choose 3") {
			t.Fatal("the instructions must explain the third choice")
		}
	}
	for _, cmd := range []string{"choose 0", "choose 4", "choose nope"} {
		g.processCommand(cmd)
	}
	if !reflect.DeepEqual(before, g.player) || !reflect.DeepEqual(offer, g.current().Events[0].Gifts) || g.rng.State != rngBefore || g.stats.Turns != 0 || g.current().Events[0].Used {
		t.Fatal("reading and invalid choices must not grant gifts, reroll, or spend turns")
	}
	if g.chooseGhostGift(0) || g.chooseGhostGift(4) {
		t.Fatal("out-of-range selections must be rejected")
	}
	g.processCommand("a")
	g.processCommand("a")
	before = g.player
	turns := g.stats.Turns
	g.processCommand("choose 3")
	if !reflect.DeepEqual(before, g.player) || g.stats.Turns != turns || g.current().Events[0].Used {
		t.Fatal("a distant ghost cannot give a gift")
	}
}

func TestGhostOffersSurviveRevisitingAndSaveLoad(t *testing.T) {
	g := openArena()
	g.current().Events = []floorEvent{{Kind: eventGhost, Pos: pos{8, 5}, Gifts: g.rollGhostGifts(0)}}
	offer := append([]item(nil), g.current().Events[0].Gifts...)
	g.processCommand("d")
	g.processCommand("d")
	g.processCommand("y")
	g.processCommand("a")
	g.processCommand("d")
	g.processCommand("y")
	g.saveFile, g.timestampedSaves = filepath.Join(t.TempDir(), "gifts.json"), false
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	rngBefore := g.rng.State
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	loaded.processCommand("y")
	if !reflect.DeepEqual(offer, loaded.current().Events[0].Gifts) || loaded.rng.State != rngBefore || loaded.stats.Turns != 4 {
		t.Fatal("revisiting and loading must keep the exact choices without consuming randomness")
	}
	loaded.processCommand("choose 3")
	if err := loaded.save(); err != nil {
		t.Fatal(err)
	}
	claimed, err := loadGame(loaded.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	before := claimed.player
	claimed.processCommand("choose 1")
	if !claimed.current().Events[0].Used || !reflect.DeepEqual(offer, claimed.current().Events[0].Gifts) || !reflect.DeepEqual(before, claimed.player) || claimed.stats.Turns != 5 {
		t.Fatal("loading a claimed offer must not renew or reroll its gifts")
	}
}

func TestOlderSavesReceiveChoicesOnlyForUnusedGhosts(t *testing.T) {
	g := openArena()
	g.current().Events = []floorEvent{
		{Kind: eventGhost, Pos: pos{6, 5}},
		{Kind: eventGhost, Pos: pos{10, 5}, Used: true},
	}
	g.saveFile, g.timestampedSaves = filepath.Join(t.TempDir(), "old-gifts.json"), false
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.current().Events[0].Gifts) != 3 || len(loaded.current().Events[1].Gifts) != 0 || !loaded.current().Events[1].Used {
		t.Fatal("an older save should gain three choices without renewing a used ghost")
	}
	offer, rngState := append([]item(nil), loaded.current().Events[0].Gifts...), loaded.rng.State
	if err := loaded.save(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadGame(loaded.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(offer, reloaded.current().Events[0].Gifts) || reloaded.rng.State != rngState {
		t.Fatal("saving the upgraded offer should prevent further rerolls")
	}
}
