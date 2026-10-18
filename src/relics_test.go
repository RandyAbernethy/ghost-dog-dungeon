package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRelicsSpawnRarelyOnlyInInnerRoomsOnFloorsSevenThroughNine(t *testing.T) {
	if testing.Short() {
		t.Skip("1,000-seed relic and Epic sweep runs in the full suite")
	}
	relics, eligible, epicRuns := 0, 0, 0
	var example *game
	seenEpics := map[monsterKind]bool{}
	for seed := int64(1); seed <= 1000; seed++ {
		g := newGame(seed)
		if g.generationError != nil {
			t.Fatalf("seed %d: %v", seed, g.generationError)
		}
		runRelics, runEpics, nonRelicEpics := 0, 0, 0
		for _, lvl := range g.levels {
			floorRelics, floorEpics := 0, 0
			innerTiles := map[pos]bool{}
			roots := lvl.Secrets
			for _, root := range roots {
				if root == nil {
					continue
				}
				for inner := root.Next; inner != nil; inner = inner.Next {
					if relicDepth(lvl.Index) {
						eligible++
					}
					for _, p := range inner.Tiles {
						innerTiles[p] = true
					}
				}
			}
			for _, it := range lvl.Items {
				if !relicItem(it) {
					continue
				}
				runRelics++
				floorRelics++
				if !relicDepth(lvl.Index) || !innerTiles[it.Pos] || lvl.Tiles[it.Pos.Y][it.Pos.X] != '#' || g.fountainAt(lvl, it.Pos) != nil {
					t.Fatalf("seed %d: relic escaped its hidden inner chamber on floor %d", seed, lvl.Index+1)
				}
				example = g
			}
			ordinary := []monster{}
			for _, m := range lvl.Monsters {
				if !epicMonster(m.Kind) {
					ordinary = append(ordinary, *m)
					continue
				}
				runEpics++
				floorEpics++
				seenEpics[m.Kind] = true
				if !relicDepth(lvl.Index) || distance(m.Pos, lvl.Start) < 8 || (lvl.HasStair && distance(m.Pos, lvl.Stairs) < 6) || g.itemAt(lvl, m.Pos) != nil || g.fountainAt(lvl, m.Pos) != nil || lvl.eventAt(m.Pos) != nil {
					t.Fatal("Epic should be distant and clear of features")
				}
				bypass := cloneTiles(lvl.Tiles)
				bypass[m.Pos.Y][m.Pos.X] = '#'
				if len(tileDistances(bypass, lvl.Start)) != countFloorTiles(lvl.Tiles)-1 {
					t.Fatal("an Epic must never block the only route through its floor")
				}
			}
			if rosterDifficulty(ordinary) != lvl.InitialDifficulty || !fairFloorRoster(ordinary, lvl.Index) {
				t.Fatal("Epic encounter should not alter ordinary budgets")
			}
			if floorEpics > 1 || floorRelics > 0 && floorEpics != 1 {
				t.Fatalf("seed %d floor %d: %d relics require one Epic, got %d", seed, lvl.Index+1, floorRelics, floorEpics)
			}
			if floorRelics == 0 {
				nonRelicEpics += floorEpics
			}
		}
		if runRelics > 1 || runEpics > 2 || nonRelicEpics > 1 {
			t.Fatal("a dungeon should contain at most one room relic, its Epic, and one random Epic elsewhere")
		}
		relics += runRelics
		if runEpics > 0 {
			epicRuns++
		}
	}
	if relics == 0 || relics >= eligible/2 || epicRuns == 0 || epicRuns >= 500 || len(seenEpics) != len(epicMonsterKinds) {
		t.Fatalf("unexpected rarity: %d relics in %d eligible rooms, %d Epic runs, %v", relics, eligible, epicRuns, seenEpics)
	}
	repeated := newGame(example.seed)
	if !reflect.DeepEqual(example.levels, repeated.levels) || example.rng.State != repeated.rng.State {
		t.Fatal("relic and Epic placement must reproduce from the seed")
	}
	t.Run("nested discovery and persistence", func(t *testing.T) {
		g := repeated
		g.saveDir = filepath.Dir(filepath.Join(t.TempDir(), "nested.json"))
		var treasure item
		var outer, inner *secretRoom
		for depth, lvl := range g.levels {
			for _, it := range lvl.Items {
				if !relicItem(it) {
					continue
				}
				treasure = it
				g.currentLevel, g.player.Pos = depth, lvl.Start
				lvl.Visited = true
				for _, root := range lvl.Secrets {
					if root == nil {
						continue
					}
					for room := root.Next; room != nil; room = room.Next {
						for _, p := range room.Tiles {
							if p == it.Pos {
								outer, inner = root, room
							}
						}
					}
				}
			}
		}
		if inner == nil {
			t.Fatal("generated relic should have an inner chamber")
		}
		if err := g.save(); err != nil {
			t.Fatal(err)
		}
		loaded, err := loadGame(g.saveFile)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(g.levels, loaded.levels) || g.rng.State != loaded.rng.State {
			t.Fatal("loading a generated dungeon must not reroll relics or encounters")
		}
		revealRoom(g.current(), outer)
		for _, it := range g.visibleItems(mapWidth + mapHeight) {
			if relicItem(it) {
				t.Fatal("opening the outer room must leave its relic concealed")
			}
		}
		revealRoom(g.current(), inner)
		if g.current().Tiles[treasure.Pos.Y][treasure.Pos.X] == '#' {
			t.Fatal("opening the inner room should expose its relic")
		}
		g.player.Pos = treasure.Pos
		g.collectItems()
		if err := g.save(); err != nil {
			t.Fatal(err)
		}
		loaded, err = loadGame(g.saveFile)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(g.player, loaded.player) || !reflect.DeepEqual(g.levels, loaded.levels) || g.rng.State != loaded.rng.State {
			t.Fatal("collected relics must stay collected with exact powers after loading")
		}
	})
	t.Logf("1000 runs: %d relics in %d eligible chambers; %d runs with Epics", relics, eligible, epicRuns)
}

func TestAllRelicsArePossibleAndExcludedFromOrdinaryLootAndGifts(t *testing.T) {
	seen := map[string]bool{}
	for seed := int64(1); seed <= 400; seed++ {
		g := varietyArena(seed)
		lvl := g.current()
		lvl.Index = 6
		inner := &secretRoom{Tiles: []pos{{12, 8}, {13, 8}, {12, 9}, {13, 9}}}
		lvl.Secrets = []*secretRoom{{Tiles: []pos{{11, 8}}, Next: inner}}
		lvl.Items = []item{{Pos: pos{12, 8}, Kind: itemPotion, Name: "healing potion", Glyph: '!'}}
		g.addLinkedRelic(lvl)
		if relicItem(lvl.Items[0]) {
			seen[lvl.Items[0].Name] = true
		}
	}
	if len(seen) != len(relicPool()) {
		t.Fatalf("not every relic is possible: %v", seen)
	}
	g := varietyArena(1)
	for depth := 0; depth < levelCount; depth++ {
		for _, it := range g.secretTreasurePool(depth) {
			if relicItem(it) {
				t.Fatal("ordinary secret treasure must not contain relics")
			}
		}
		for _, w := range weaponPoolForLevel(depth) {
			if w.Rarity == rarityArtifact {
				t.Fatal("ordinary weapons must not contain Artifacts")
			}
		}
		for _, a := range armorPoolForLevel(depth) {
			if a.Rarity == rarityGod {
				t.Fatal("ordinary armor must not contain God armor")
			}
		}
		for i := 0; i < 100; i++ {
			for _, it := range g.rollGhostGifts(depth) {
				if relicItem(it) {
					t.Fatal("friendly ghosts must not offer relics")
				}
			}
		}
	}
}

func TestRoomRelicChanceIsOneRollPerEligibleFloor(t *testing.T) {
	// Fixed seeds exercise both sides of the 25% boundary. More chambers must
	// supply more candidate positions, without granting additional chance rolls.
	for _, tc := range []struct {
		seed int64
		roll int
		want bool
	}{{116, 0, true}, {100, 24, true}, {29, 25, false}, {79, 99, false}} {
		for rooms := 1; rooms <= 3; rooms++ {
			t.Run(fmt.Sprintf("roll_%d/rooms_%d", tc.roll, rooms), func(t *testing.T) {
				g := openArena()
				g.currentLevel = 6
				g.rng = newSimpleRNG(tc.seed)
				lvl := g.current()
				root := &secretRoom{Tiles: []pos{{7, 5}}}
				lvl.Secrets = []*secretRoom{root}
				last := root
				for room := 0; room < rooms; room++ {
					p := pos{10 + 3*room, 5}
					last.Next = &secretRoom{Door: pos{p.X - 1, p.Y}, Tiles: []pos{p}}
					last = last.Next
					lvl.Items = append(lvl.Items, item{Pos: p, Kind: itemPotion})
				}
				expected := newSimpleRNG(tc.seed)
				if roll := expected.Intn(100); roll != tc.roll {
					t.Fatalf("seed fixture produced roll %d, want %d", roll, tc.roll)
				}
				g.addLinkedRelic(lvl)
				count := 0
				for _, it := range lvl.Items {
					if relicItem(it) {
						count++
					}
				}
				wantCount := 0
				if tc.want {
					wantCount = 1
				}
				if count != wantCount || len(lvl.Items) != rooms {
					t.Fatalf("roll %d with %d rooms produced %d relics and %d items", tc.roll, rooms, count, len(lvl.Items))
				}
				if !tc.want && g.rng.State != expected.State {
					t.Fatal("a rejected floor roll must not retry for additional chambers")
				}
				if tc.want {
					next := g.levels[7]
					next.Secrets = lvl.Secrets
					for _, it := range lvl.Items {
						next.Items = append(next.Items, item{Pos: it.Pos, Kind: itemPotion})
					}
					before := g.rng.State
					g.addLinkedRelic(next)
					if g.rng.State != before {
						t.Fatal("an existing room relic must suppress subsequent floor rolls")
					}
				}
			})
		}
	}
}

func TestRelicPowersActivateOnlyWhileEquipped(t *testing.T) {
	g := varietyArena(2)
	g.dog.Freed = true
	g.player.carryArmor(godArmorPieces[0])
	g.hurtDog(10, "test")
	if g.dog.HP != 24 {
		t.Fatal("packed God armor must not protect the dog")
	}
	g.player.BodyArmor = godArmorPieces[0]
	g.hurtDog(10, "test")
	if g.dog.HP != 20 {
		t.Fatal("Sanctuary must block six damage near its wearer")
	}
	g.dog.Pos = pos{1, 1}
	g.hurtDog(10, "test")
	if g.dog.HP != 10 {
		t.Fatal("Sanctuary must not protect a distant companion")
	}
	g.dog.Alive = false
	g.hurtDog(1, "test")
	if g.dog.Alive {
		t.Fatal("Sanctuary must never revive Ghost Dog")
	}
	g.player.HeadArmor = godArmorPieces[1]
	g.hexPlayer()
	g.bindPlayer(bindingWebs)
	if g.player.HexedTurns != 0 || g.player.BoundTurns != 0 {
		t.Fatal("Oracle Crown must prevent curses and bindings")
	}
	g.player.HeadArmor = armor{}
	g.hexPlayer()
	g.bindPlayer(bindingWebs)
	if g.player.HexedTurns != 1 || g.player.BoundTurns != 1 {
		t.Fatal("immunity should end when the crown is removed")
	}
	g.player.LegArmor = godArmorPieces[2]
	m := newMonster(monsterOrc)
	m.Pos, m.MinDamage, m.MaxDamage = pos{6, 5}, 40, 40
	g.current().Monsters = []*monster{&m}
	g.player.HexedTurns, g.player.BoundTurns = 0, 0
	g.attackPlayer(&m)
	if m.HP != m.MaxHP-12 || g.player.HP >= 46 {
		t.Fatal("Starthorn should retaliate after a survived damaging melee hit")
	}
	before := g.player.HP
	g.hurtPlayer(1, "test")
	if g.player.HP >= before {
		t.Fatal("God armor must not make the player invulnerable")
	}
}

func TestArtifactPowersUseActualDamageAndRespectRange(t *testing.T) {
	g := varietyArena(1)
	m := newMonster(monsterIronColossus)
	m.Pos = pos{6, 5}
	g.current().Monsters = []*monster{&m}
	g.player.HP = 10
	g.player.Weapon = artifactWeapons[0]
	before := m.HP
	g.attackMonster(&m)
	dealt := before - m.HP
	if g.player.HP != 10+min(8, dealt/5) {
		t.Fatal("Dawnbreaker should heal from damage after armor")
	}
	g.player.HP = 46
	g.player.Weapon = artifactWeapons[2]
	g.rng = newSimpleRNG(1)
	roll := newSimpleRNG(1)
	want := g.player.Weapon.Min + roll.Intn(g.player.Weapon.Max-g.player.Weapon.Min+1)
	before = m.HP
	g.attackMonster(&m)
	if before-m.HP != want {
		t.Fatal("Worldsplitter must ignore armor")
	}
	g.player.Weapon = artifactWeapons[1]
	for i := 0; i < 50 && m.FrozenTurns == 0; i++ {
		m.HP, m.MaxHP = 1000, 1000
		g.attackMonster(&m)
	}
	if m.FrozenTurns != 1 {
		t.Fatal("Winter's Last Word should sometimes freeze its target")
	}
	g = varietyArena(4)
	g.player.Weapon = artifactWeapons[3]
	a, b, c := newMonster(monsterOrc), newMonster(monsterOrc), newMonster(monsterOrc)
	a.Pos, b.Pos, c.Pos = pos{6, 5}, pos{7, 5}, pos{10, 5}
	a.HP, a.MaxHP = 100, 100
	g.current().Monsters = []*monster{&a, &b, &c}
	g.attackMonster(&a)
	if b.HP != b.MaxHP-12 || c.HP != c.MaxHP {
		t.Fatal("lightning must hit only a nearby secondary enemy")
	}
}

func TestRelicEquipmentAndEpicStateSurviveCommandsAndSave(t *testing.T) {
	g := newGameWithSaveDir(1, t.TempDir())
	lvl := g.current()
	lvl.Monsters = nil
	g.player.carryWeapon(artifactWeapons[0])
	g.player.carryArmor(godArmorPieces[1])
	wNumber := len(g.player.Weapons)
	aNumber := wNumber + len(g.player.Armors)
	g.player.HexedTurns, g.player.BoundTurns = 1, 1
	g.processCommand(fmt.Sprintf("equip %d", wNumber))
	g.processCommand(fmt.Sprintf("equip %d", aNumber))
	if g.stats.Turns != 2 || g.player.HexedTurns != 0 || g.player.BoundTurns != 0 {
		t.Fatal("equipping the crown should clear effects and cost one turn")
	}
	epic := newMonster(monsterGraveTitan)
	epic.Pos = pos{8, 8}
	epic.HP = 42
	epic.ActionTurns, epic.Awakened = 2, true
	lvl.Monsters = []*monster{&epic}
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			lvl.Tiles[y][x] = '.'
		}
	}
	g.player.Pos = pos{5, 5}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.player, loaded.player) || !reflect.DeepEqual(g.levels, loaded.levels) || g.rng.State != loaded.rng.State {
		t.Fatal("loading must preserve gear powers and Epic cadence, wounds, and placement")
	}
	var out bytes.Buffer
	loaded.renderInventory(&out)
	if !strings.Contains(out.String(), "Artifact Dawnbreaker") || !strings.Contains(out.String(), "Life drain:") || !strings.Contains(out.String(), "God Oracle Crown") || !strings.Contains(out.String(), "Clear mind:") {
		t.Fatal("inventory must explain the tiers and powers")
	}
}

func TestReadingRelicScreensDoesNotSpendTurnsOrActivatePowers(t *testing.T) {
	g := varietyArena(1)
	g.player.Weapon = artifactWeapons[0]
	g.player.HeadArmor = godArmorPieces[1]
	g.player.HP = 20
	g.player.ensureCarriedEquipment()
	m := newMonster(monsterGraveTitan)
	m.Pos = pos{6, 5}
	g.current().Monsters = []*monster{&m}
	beforeHP, beforeStats, beforeRNG := g.player.HP, g.stats, g.rng.State
	var out bytes.Buffer
	g.renderInventory(&out)
	g.renderInspect(&out)
	g.renderCodex(&out)
	if g.player.HP != beforeHP || g.stats != beforeStats || g.rng.State != beforeRNG || m.Awakened || m.ActionTurns != 0 {
		t.Fatal("reading must not heal, awaken an Epic, spend a turn, or roll powers")
	}
}

func TestRelicsThroughRealExecutablePickupEquipDropAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "run.json")
	g := openArena()
	g.saveDir = filepath.Dir(path)
	w, a := makeWeaponItem(artifactWeapons[2]), makeArmorItem(godArmorPieces[1])
	w.Pos, a.Pos = pos{6, 5}, pos{6, 5}
	g.current().Items = []item{w, a}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	path = g.saveFile
	cmd := gameCommand(t, "--load-file", path, "--save-file-path", dir)
	cmd.Env = testEnvironment(dir)
	cmd.Stdin = strings.NewReader("d\ni\nequip 2\nequip 6\ndrop 6\ni\nequip 1\ndrop 2\nsave\nload\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, output)
	}
	for _, want := range []string{"Artifact Worldsplitter", "Worldshatter:", "God Oracle Crown", "Clear mind:", "cannot drop an equipped item", "You drop Worldsplitter", "Game loaded"} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("executable did not display %q", want)
		}
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.player.HeadArmor != a.Armor || loaded.stats.Turns != 5 {
		t.Fatal("CLI must preserve God armor and action accounting")
	}
	found := false
	for _, it := range loaded.current().Items {
		if it.Weapon == w.Weapon {
			found = true
		}
	}
	if !found {
		t.Fatal("dropped Artifact must retain its rarity, power, and exact damage through save/load")
	}
	if _, err := os.Stat(filepath.Join(dir, ".ghost-dog-data.json")); !os.IsNotExist(err) {
		t.Fatal("saving and quitting must not submit unfinished challenge records")
	}
}
