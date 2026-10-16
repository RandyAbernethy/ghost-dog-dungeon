package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func assertEliteTreasure(t *testing.T, treasure item) {
	t.Helper()
	switch treasure.Kind {
	case itemWeapon:
		for _, w := range artifactWeapons {
			if treasure.Weapon == w && treasure.Name == w.Name && treasure.Glyph == ')' {
				return
			}
		}
	case itemArmor:
		for _, a := range godArmorPieces {
			if treasure.Armor == a && treasure.Name == a.Name && treasure.Glyph == '[' {
				return
			}
		}
	case itemStarfireOrb, itemSunOrb, itemPhoenixAsh, itemGhostRecallScroll:
		return
	}
	t.Fatalf("Epic dropped a treasure outside the elite tiers: %+v", treasure)
}

func TestEveryEpicDropsOneEliteTreasureOnlyOnDeath(t *testing.T) {
	seen := map[string]bool{}
	for _, kind := range epicMonsterKinds {
		for seed := int64(1); seed <= 128; seed++ {
			kill := func() *game {
				g := varietyArena(seed)
				m := newMonster(kind)
				m.Pos = pos{6, 5}
				g.current().Monsters = []*monster{&m}
				beforeRNG := g.rng.State
				g.damageMonster(&m, monsterGuard(kind)+1, "hits")
				if m.HP <= 0 || len(g.current().Items) != 0 || g.stats.MonstersKilled != 0 || g.rng.State != beforeRNG {
					t.Fatal("wounding an Epic must not roll or award a treasure")
				}
				g.damageMonster(&m, m.MaxHP+monsterGuard(kind), "hits")
				if m.HP != 0 || len(g.current().Items) != 1 || g.stats.MonstersKilled != 1 || g.stats.DifficultyDefeated != monsterDifficulty(m) {
					t.Fatal("a dead Epic must award one treasure and one full kill score")
				}
				treasure := g.current().Items[0]
				assertEliteTreasure(t, treasure)
				if treasure.Pos != m.Pos || !strings.Contains(strings.Join(g.messages, "\n"), m.Name+" drops an elite treasure:") {
					t.Fatal("the reward should appear at the corpse and be announced")
				}
				beforeRNG = g.rng.State
				g.damageMonster(&m, m.MaxHP+monsterGuard(kind), "hits")
				if len(g.current().Items) != 1 || g.current().Items[0] != treasure || g.stats.MonstersKilled != 1 || g.rng.State != beforeRNG {
					t.Fatal("a dead monster must not award or reroll its reward again")
				}
				return g
			}
			first, repeated := kill(), kill()
			if !reflect.DeepEqual(first.current().Items, repeated.current().Items) || first.rng.State != repeated.rng.State {
				t.Fatal("the same seed and attacks must reproduce the Epic's reward")
			}
			seen[first.current().Items[0].Name] = true
		}
	}
	if len(seen) != len(artifactWeapons)+len(godArmorPieces)+4 {
		t.Fatalf("all supreme equipment and four elite magic rewards should be possible: %v", seen)
	}
	for _, kind := range []monsterKind{monsterRat, monsterGraveKnight} {
		g := varietyArena(1)
		m := newMonster(kind)
		m.Pos = pos{6, 5}
		g.current().Monsters = []*monster{&m}
		g.damageMonster(&m, m.MaxHP+monsterGuard(kind), "hits")
		if len(g.current().Items) != 0 {
			t.Fatal("ordinary monsters must not drop Epic rewards")
		}
	}
}

func TestEpicTreasureDropsThroughPlayerCompanionMagicAndGearPowers(t *testing.T) {
	for _, source := range []string{"player", "Ghost Dog", "fire scroll", "starfire orb", "chain lightning", "God reprisal"} {
		t.Run(source, func(t *testing.T) {
			g := varietyArena(1)
			m := newMonster(monsterAshenWyrm)
			m.Pos, m.HP = pos{6, 5}, 1
			g.current().Monsters = []*monster{&m}
			cmd := "d"
			switch source {
			case "player":
				g.player.Weapon = weapon{Name: "Test blade", Min: 20, Max: 20}
			case "Ghost Dog":
				g.dog.Freed, g.dog.Pos = true, pos{5, 4}
				cmd = "."
			case "fire scroll":
				g.player.FireScrolls, cmd = 1, "f"
			case "starfire orb":
				g.player.StarfireOrbs, cmd = 1, "o"
			case "chain lightning":
				g.player.Weapon = artifactWeapons[3]
				primary := newMonster(monsterRat)
				primary.Pos, primary.HP, primary.MaxHP = pos{6, 5}, 200, 200
				m.Pos = pos{7, 5}
				g.current().Monsters = append(g.current().Monsters, &primary)
			case "God reprisal":
				g.player.LegArmor = godArmorPieces[2]
				m.MinDamage, m.MaxDamage, cmd = 20, 20, "."
			}
			g.processCommand(cmd)
			if m.HP != 0 || len(g.current().Items) != 1 || g.stats.MonstersKilled != 1 || g.stats.Turns != 1 {
				t.Fatalf("%s's kill should drop a treasure through the normal turn: hp=%d items=%v", source, m.HP, g.current().Items)
			}
			assertEliteTreasure(t, g.current().Items[0])
			if g.current().Items[0].Pos != m.Pos {
				t.Fatal("the reward must use the slain monster's position")
			}
		})
	}
}

func TestEpicDropUsesMagicWhenEveryRelicIsAlreadyPresent(t *testing.T) {
	for seed := int64(1); seed <= 32; seed++ {
		g := varietyArena(seed)
		g.player.Weapon = artifactWeapons[0]
		g.player.Weapons = []weapon{artifactWeapons[1], artifactWeapons[2]}
		reserved := makeWeaponItem(artifactWeapons[3])
		reserved.Pos = pos{12, 12}
		g.current().Items = []item{reserved}
		g.player.HeadArmor = godArmorPieces[1]
		g.player.Armors = []armor{godArmorPieces[0]}
		g.levels = append(g.levels, &level{Items: []item{makeArmorItem(godArmorPieces[2])}})
		m := newMonster(monsterGraveTitan)
		m.Pos = pos{6, 5}
		g.current().Monsters = []*monster{&m}
		g.damageMonster(&m, m.MaxHP+monsterGuard(m.Kind), "hits")
		if len(g.current().Items) != 2 || g.current().Items[0] != reserved {
			t.Fatal("the guaranteed reward must preserve existing loot")
		}
		if treasure := g.current().Items[1]; treasure.Kind == itemWeapon || treasure.Kind == itemArmor {
			t.Fatal("rewards must avoid carried, equipped, and undiscovered duplicate gear")
		} else {
			assertEliteTreasure(t, treasure)
		}
	}
}

func TestEpicRewardPreservesTheFallenDogRecallBonus(t *testing.T) {
	withBonus, withoutBonus := false, false
	for seed := int64(1); seed <= 128; seed++ {
		g := varietyArena(seed)
		g.dog.Freed, g.dog.Alive, g.dog.HP = true, false, 0
		m := newMonster(monsterAshenWyrm)
		m.Pos = pos{6, 5}
		g.current().Monsters = []*monster{&m}
		g.damageMonster(&m, m.MaxHP+monsterGuard(m.Kind), "hits")
		items := g.current().Items
		if len(items) < 1 || len(items) > 2 || g.dog.Alive || g.dog.HP != 0 {
			t.Fatal("an Epic's treasure must not revive Ghost Dog automatically")
		}
		assertEliteTreasure(t, items[0])
		if len(items) == 2 {
			withBonus = true
			if items[1].Kind != itemGhostRecallScroll || items[1].Pos != m.Pos {
				t.Fatal("the only additional drop should be the existing recall-scroll bonus")
			}
		} else {
			withoutBonus = true
		}
		beforeRNG := g.rng.State
		g.damageMonster(&m, 1000, "hits")
		if !reflect.DeepEqual(items, g.current().Items) || g.rng.State != beforeRNG || g.stats.MonstersKilled != 1 {
			t.Fatal("neither the treasure nor the recall bonus may repeat for a dead Epic")
		}
	}
	if !withBonus || !withoutBonus {
		t.Fatal("the recall bonus should remain possible rather than guaranteed")
	}
}

func epicLootSaveArena(path string) *game {
	g := openArena()
	g.saveDir = filepath.Dir(path)
	g.rng = newSimpleRNG(7)
	g.player.Weapon = weapon{Name: "Test blade", Min: 20, Max: 20}
	g.player.ensureCarriedEquipment()
	m := newMonster(monsterAshenWyrm)
	m.Pos, m.HP = pos{6, 5}, 1
	g.current().Monsters = []*monster{&m}
	return g
}

func TestEpicRewardSurvivesSavingBeforeDeathAfterDeathAndAfterPickup(t *testing.T) {
	g := epicLootSaveArena(filepath.Join(t.TempDir(), "epic.json"))
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	repeated, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	g.processCommand("d")
	repeated.processCommand("d")
	if !reflect.DeepEqual(g.current().Items, repeated.current().Items) || g.rng.State != repeated.rng.State {
		t.Fatal("loading before the kill must preserve the reward roll")
	}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.current(), loaded.current()) || g.stats != loaded.stats || g.rng.State != loaded.rng.State {
		t.Fatal("loading after the kill must preserve the corpse, treasure, score, and RNG")
	}
	loaded.processCommand("d")
	if len(loaded.current().Items) != 0 || loaded.stats.Turns != 2 || loaded.stats.MonstersKilled != 1 {
		t.Fatal("walking onto the corpse should collect its reward normally")
	}
	if err := loaded.save(); err != nil {
		t.Fatal(err)
	}
	collected, err := loadGame(loaded.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.player, collected.player) || len(collected.current().Items) != 0 || collected.rng.State != loaded.rng.State {
		t.Fatal("the collected treasure must survive loading without another drop")
	}
	collected.processCommand(".")
	if len(collected.current().Items) != 0 || collected.stats.MonstersKilled != 1 {
		t.Fatal("revisiting a corpse must not regenerate its treasure")
	}
}

func TestEpicDropPickupAndSaveThroughRealExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "epic.json")
	g := epicLootSaveArena(path)
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	path = g.saveFile
	cmd := gameCommand(t, "--load-file", path, "--save-file-path", dir)
	cmd.Env = testEnvironment(dir)
	cmd.Stdin = strings.NewReader("d\nsave\nload\nd\ni\nsave\nload\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, output)
	}
	for _, want := range []string{"Ghost Dog Dungeon version 1.0.1", "Epic Ashen Wyrm drops an elite treasure:", "=== Inventory & Equipment ===", "Game loaded from "} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("the executable did not display %q", want)
		}
	}
	g.processCommand("d")
	reward := g.current().Items[0]
	if !strings.Contains(stripANSI(string(output)), reward.Name) {
		t.Fatal("the executable should name the dropped reward")
	}
	g.processCommand("d")
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.player, loaded.player) || len(loaded.current().Items) != 0 || loaded.current().Monsters[0].HP != 0 || loaded.stats.MonstersKilled != 1 || loaded.stats.Turns != 2 || loaded.rng.State != g.rng.State {
		t.Fatal("the executable must collect and preserve exactly one elite reward and kill")
	}
	if _, err := os.Stat(filepath.Join(dir, ".ghost-dog-data.json")); !os.IsNotExist(err) {
		t.Fatal("killing an Epic and quitting must not submit an unfinished run")
	}
}
