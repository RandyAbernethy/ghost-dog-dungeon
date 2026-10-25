package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func cryptArena() *game {
	g := openArena()
	g.currentLevel = 3
	g.player.HP, g.player.MaxHP = 200, 200
	g.crypt = &lichCrypt{ParentFloor: 3, Entrance: pos{6, 5}, ReturnDoor: pos{32, 10}, Exit: pos{36, 5},
		Floor: &level{Index: cryptDepth, Tiles: cloneTiles(g.current().Tiles), Start: pos{2, 5}}}
	g.current().Tiles[5][6] = '#'
	g.current().Tiles[10][32] = '#'
	g.crypt.Floor.Tiles[5][36] = '#'
	return g
}

func TestCryptEntranceAndFarExitAreOneWayAndPersist(t *testing.T) {
	g := cryptArena()
	g.dog.Freed, g.dog.Pos = true, pos{4, 5}
	g.processCommand("d")
	if g.stats.Turns != 0 || g.inCrypt {
		t.Fatal("the unrevealed door must be solid stone")
	}
	rng := g.rng.State
	g.processCommand("k")
	if !g.crypt.EntranceRevealed || g.inCrypt || g.stats.Turns != 1 || g.rng.State != rng || !strings.Contains(strings.Join(g.messages, "\n"), "entrance seals behind you") {
		t.Fatal("search must reveal and explain the one-way entrance")
	}
	g = saveCryptAndReload(t, g)
	g.player.ShieldTurns = 3
	g.current().Fountains = []fountain{{Pos: pos{4, 6}, Used: true}}
	g.processCommand("d")
	if !g.inCrypt || g.current() != g.crypt.Floor || g.player.Pos != g.crypt.Floor.Start || g.player.ShieldTurns != 0 || g.currentLevel != 3 || g.stats.Turns != 2 || distance(g.player.Pos, g.dog.Pos) > 1 {
		t.Fatal("entering must take one turn, clear wards and move the living companion to the crypt")
	}
	if g.levels[3].Fountains[0].Used {
		t.Fatal("leaving the parent floor for the crypt must refill its fountain")
	}
	g = saveCryptAndReload(t, g)
	g.processCommand("<")
	g.processCommand(">")
	if !g.inCrypt || g.stats.Turns != 2 {
		t.Fatal("ordinary stairs must not escape the crypt")
	}
	g.player.Pos = pos{35, 5}
	g.processCommand("d")
	if !g.inCrypt || g.stats.Turns != 2 {
		t.Fatal("the far secret exit must be found before use")
	}
	g.processCommand("k")
	if !g.crypt.ExitRevealed {
		t.Fatal("searching beside the far wall must reveal the exit")
	}
	g = saveCryptAndReload(t, g)
	g.processCommand("d")
	if g.inCrypt || !g.crypt.Completed || !g.crypt.ReturnRevealed || g.player.Pos != g.crypt.ReturnDoor || g.currentLevel != 3 || g.stats.Turns != 4 || distance(g.player.Pos, g.dog.Pos) > 1 {
		t.Fatal("only the far door returns both companions to the entrance floor")
	}
	g = saveCryptAndReload(t, g)
	g.player.Pos = pos{5, 5}
	g.processCommand("d")
	if g.inCrypt || g.stats.Turns != 4 {
		t.Fatal("the consumed entrance must stay sealed after returning and loading")
	}
}

func saveCryptAndReload(t *testing.T, g *game) *game {
	t.Helper()
	loaded := saveAndReloadAdventure(t, g)
	if !reflect.DeepEqual(g.crypt, loaded.crypt) || g.inCrypt != loaded.inCrypt || g.dog != loaded.dog || !reflect.DeepEqual(g.messages, loaded.messages) {
		t.Fatal("all crypt terrain, doors, actors, traps and logs must survive saving exactly")
	}
	return loaded
}

func TestCryptMagicConsumesItemsWithoutBlinkOrWard(t *testing.T) {
	g := cryptArena()
	g.inCrypt = true
	g.player.Pos = g.crypt.Floor.Start
	g.dog.Freed, g.dog.Pos = true, pos{3, 5}
	g.player.BlinkStones, g.player.WardingCharms = 2, 2
	rng, playerPos, dogPos := g.rng.State, g.player.Pos, g.dog.Pos
	g.processCommand("g")
	if !strings.Contains(strings.Join(g.messages, " "), "The power of the crypt consumes your Blink Stone's magic without effect") {
		t.Fatal("suppressed blink must explain the crypt consumed its magic")
	}
	g.processCommand("b")
	if !strings.Contains(strings.Join(g.messages, " "), "The power of the crypt consumes your Warding Charm's magic without effect") {
		t.Fatal("suppressed ward must explain the crypt consumed its magic")
	}
	if g.player.BlinkStones != 1 || g.player.WardingCharms != 1 || g.player.Pos != playerPos || g.dog.Pos != dogPos || g.player.ShieldTurns != 0 || g.stats.Turns != 2 || g.rng.State != rng {
		t.Fatal("suppressed magic must spend its item and a turn without teleporting, shielding or drawing RNG")
	}
	g = saveCryptAndReload(t, g)
	g.processCommand("g")
	g.processCommand("b")
	g.processCommand("g")
	g.processCommand("b")
	if g.player.BlinkStones != 0 || g.player.WardingCharms != 0 || g.stats.Turns != 4 || g.player.Pos != playerPos {
		t.Fatal("suppression must persist after loading; empty inventory attempts remain free")
	}
	g.player.ShieldTurns = 3
	g.player.HeadArmor, g.player.BodyArmor, g.player.LegArmor = armor{}, armor{}, armor{}
	g.hurtPlayer(10, "test strike")
	g.hurtDog(10, "test strike")
	if g.player.HP != 190 || g.dog.HP != 24 {
		t.Fatal("an active ward cannot reduce damage anywhere inside the crypt")
	}
}

func TestCryptSightLimitsMapInspectCodexAndTargeting(t *testing.T) {
	g := cryptArena()
	g.inCrypt = true
	g.player.Pos = pos{10, 8}
	near := newMonster(monsterSkeleton)
	near.Pos = pos{13, 8}
	far := newMonster(monsterUnderLich)
	far.Pos = pos{14, 8}
	behindWall := newMonster(monsterGraveKnight)
	behindWall.Pos = pos{10, 10}
	g.current().Tiles[9][10] = '#'
	g.current().Monsters = []*monster{&near, &far, &behindWall}
	g.current().Items = []item{{Kind: itemPotion, Name: "near potion", Pos: pos{12, 8}}, {Kind: itemPotion, Name: "hidden treasure", Pos: pos{15, 8}}}
	beforeTiles, beforeRNG, beforeStats := encodeTiles(g.current().Tiles), g.rng.State, g.stats
	var out bytes.Buffer
	g.render(&out)
	g.renderInspect(&out)
	g.renderCodex(&out)
	if !g.canSee(near.Pos) || g.canSee(far.Pos) || g.canSee(behindWall.Pos) || !g.canSee(pos{10, 9}) || len(g.visibleThreats(6)) != 1 || len(g.visibleItems(6)) != 1 || g.nearestMonster(6) != &near {
		t.Fatal("three-tile sight must stop at walls and apply to inspection and target acquisition")
	}
	if strings.Contains(out.String(), "Under Lich") || strings.Contains(out.String(), "Grave Knight") || strings.Contains(out.String(), "hidden treasure") || !g.knownMonsters[near.Kind] || g.knownMonsters[far.Kind] || g.knownMonsters[behindWall.Kind] {
		t.Fatal("reading must not reveal unseen enemies, loot or codex entries")
	}
	if beforeTiles != encodeTiles(g.current().Tiles) || beforeRNG != g.rng.State || beforeStats != g.stats {
		t.Fatal("rendering the crypt must not mutate terrain, RNG or turns")
	}
	g.monsterTurn(&behindWall)
	if g.player.HP != 200 {
		t.Fatal("an unseen enemy cannot attack through a wall")
	}
}

func TestCryptSunOrbsStillReachUnseenUndeadWithoutCodexDiscovery(t *testing.T) {
	g := cryptArena()
	g.inCrypt = true
	g.player.Pos = pos{5, 5}
	near, far := newMonster(monsterSkeleton), newMonster(monsterUnderLich)
	near.Pos, near.HP = pos{7, 5}, 1
	far.Pos, far.HP = pos{25, 10}, 1
	g.current().Monsters = []*monster{&near, &far}
	g.player.SunOrbs = 1
	g.processCommand("u")
	if near.HP != 0 || far.HP != 0 || g.stats.MonstersKilled != 2 || !g.knownMonsters[near.Kind] || g.knownMonsters[far.Kind] {
		t.Fatal("Sun Orbs must retain their floor-wide effect without granting visual codex knowledge of unseen enemies")
	}
}

func TestUnderLichHoldsItsCrossingAndUsesCommonDeathAccounting(t *testing.T) {
	g := cryptArena()
	g.inCrypt = true
	g.player.Pos = pos{10, 8}
	m := newMonster(monsterUnderLich)
	m.Pos = pos{11, 8}
	g.current().Monsters = []*monster{&m}
	start := m.Pos
	g.monsterTurn(&m)
	hp := g.player.HP
	g.monsterTurn(&m)
	if hp >= 200 || g.player.HP != hp || m.Pos != start {
		t.Fatal("the guardian must attack every other turn while holding its crossing")
	}
	m.FleeTurns = 2
	g.monsterTurn(&m)
	if m.Pos != start {
		t.Fatal("fear must not move the guardian away from the compulsory crossing")
	}
	g.dog.Freed, g.dog.Alive, g.dog.HP = true, false, 0
	want := monsterDifficulty(m)
	g.damageMonster(&m, 10000, "sears")
	g.damageMonster(&m, 10000, "sears")
	if m.HP != 0 || g.stats.MonstersKilled != 1 || g.stats.DifficultyDefeated != want || g.dog.Alive || g.won || g.crypt.ExitRevealed || len(g.player.StoryChapters) != 0 {
		t.Fatal("the Under Lich must count once without reviving Ash, dropping the Lich's chapter or opening the campaign escape")
	}
}

func TestUnderLichFightContinuesExactlyAcrossSaveLoad(t *testing.T) {
	g := cryptArena()
	g.inCrypt, g.crypt.Floor.Visited = true, true
	g.player.Pos = pos{5, 5}
	g.player.Weapon = weapon{Name: "Test blade", Min: 20, Max: 20}
	g.player.FrostCharms = 1
	g.dog.Freed, g.dog.Pos = true, pos{4, 5}
	m := newMonster(monsterUnderLich)
	m.Pos = pos{6, 5}
	g.current().Monsters = []*monster{&m}
	for _, key := range []string{"d", "d", "t"} {
		g.processCommand(key)
	}
	if m.HP <= 0 || m.FrozenTurns == 0 {
		t.Fatal("checkpoint must preserve a living, frozen guardian during combat")
	}
	loaded := saveCryptAndReload(t, g)
	for turn := 0; turn < 20 && m.HP > 0 && g.player.HP > 0; turn++ {
		g.processCommand("d")
		loaded.processCommand("d")
		if !reflect.DeepEqual(g.player, loaded.player) || g.dog != loaded.dog || g.stats != loaded.stats || g.rng.State != loaded.rng.State || !reflect.DeepEqual(g.crypt, loaded.crypt) {
			t.Fatal("loading must not reset guardian health, cadence, freezing, companion actions or RNG")
		}
	}
	if m.HP != 0 || g.player.HP <= 0 || g.stats.MonstersKilled != 1 {
		t.Fatal("the continued arena fight must defeat the guardian once")
	}
}

func TestInvalidCryptSavesFailWithUsefulErrors(t *testing.T) {
	g := trappedArena(trapBlock)
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*saveState)
		want   string
	}{
		{"missing", func(s *saveState) { s.Crypt = nil }, "missing its dungeon"},
		{"tiles", func(s *saveState) { s.CryptTiles = "broken" }, "decode crypt"},
		{"parent", func(s *saveState) { s.Crypt.ParentFloor = 8 }, "invalid crypt"},
		{"floor", func(s *saveState) { s.Crypt.Floor = nil }, "invalid crypt"},
		{"door", func(s *saveState) { s.Crypt.Exit = pos{100, 100} }, "invalid crypt door"},
		{"player", func(s *saveState) { s.Player.Pos = pos{-1, -1} }, "outside the crypt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var state saveState
			if err := json.Unmarshal(data, &state); err != nil {
				t.Fatal(err)
			}
			tc.change(&state)
			invalid, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "invalid-crypt.json")
			if err := os.WriteFile(path, invalid, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadGame(path); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("malformed crypt save must report %q: %v", tc.want, err)
			}
		})
	}
}

func TestCryptGenerationAcrossSeeds(t *testing.T) {
	seeds := int64(500)
	if testing.Short() {
		seeds = 20
	}
	kinds := map[trapKind]bool{}
	floors := map[int]bool{}
	for seed := int64(1); seed <= seeds; seed++ {
		g := newGame(seed)
		if g.generationError != nil {
			t.Fatal(g.generationError)
		}
		c := g.crypt
		if c == nil || c.ParentFloor < 3 || c.ParentFloor > 6 || len(g.levels) != levelCount {
			t.Fatalf("seed %d: the crypt must be an extra floor reached from levels four through seven", seed)
		}
		floors[c.ParentFloor] = true
		lvl := c.Floor
		if len(lvl.Fountains) != 0 || len(lvl.Secrets) != 0 || len(lvl.Events) != 0 || lvl.Objective != nil || lvl.HasStair || lvl.HasUpStair || lvl.HasEscapeStair {
			t.Fatal("the crypt must have no ordinary stairs, fountains, events or secret rooms")
		}
		low, high := floorDifficultyBounds(cryptDepth)
		total, guardians := 0, 0
		var gate pos
		occupied := map[pos]bool{lvl.Start: true, {c.Exit.X - 1, c.Exit.Y}: true}
		for _, m := range lvl.Monsters {
			if !m.Undead || m.Boss || epicMonster(m.Kind) || occupied[m.Pos] {
				t.Fatalf("seed %d: invalid or overlapping crypt monster %+v", seed, m)
			}
			occupied[m.Pos] = true
			total += monsterDifficulty(*m)
			if m.Kind == monsterUnderLich {
				guardians++
				gate = m.Pos
			}
		}
		if guardians != 1 || total < low || total > high || total != lvl.InitialDifficulty {
			t.Fatalf("seed %d: crypt difficulty %d outside floor-eight bounds %d–%d", seed, total, low, high)
		}
		if len(tileDistances(lvl.Tiles, lvl.Start)) != countFloorTiles(lvl.Tiles) || lvl.Start.X != 1 || c.Exit.X != mapWidth-1 {
			t.Fatal("the maze must fill the map from one side to the other and remain connected")
		}
		for y := 1; y < mapHeight-1; y++ {
			open := false
			for x := 1; x < mapWidth-1; x++ {
				open = open || lvl.Tiles[y][x] != '#'
			}
			if !open {
				t.Fatal("the maze must use every interior row")
			}
		}
		for x := 1; x < mapWidth-1; x++ {
			open := false
			for y := 1; y < mapHeight-1; y++ {
				open = open || lvl.Tiles[y][x] != '#'
			}
			if !open {
				t.Fatal("the maze must use every interior column")
			}
		}
		blocked := cloneTiles(lvl.Tiles)
		blocked[gate.Y][gate.X] = '#'
		if reachable8(blocked, lvl.Start, pos{c.Exit.X - 1, c.Exit.Y}) {
			t.Fatalf("seed %d: a diagonal route bypasses the Under Lich", seed)
		}
		weapons, armors, magic := 0, 0, 0
		for _, it := range lvl.Items {
			if occupied[it.Pos] {
				t.Fatal("crypt loot overlaps an actor or door")
			}
			occupied[it.Pos] = true
			switch it.Kind {
			case itemWeapon:
				weapons++
				matched := false
				for _, w := range weaponPoolForLevel(cryptDepth) {
					matched = matched || it.Weapon == makeWeaponLoot(w).Weapon
				}
				if !matched {
					t.Fatal("crypt weapons must come from the ordinary floor-eight pool")
				}
				if it.Pos.X >= gate.X {
					t.Fatal("offer floor-eight equipment before the guardian")
				}
			case itemArmor:
				armors++
				matched := false
				for _, a := range armorPoolForLevel(cryptDepth) {
					matched = matched || it.Armor == a
				}
				if !matched {
					t.Fatal("crypt armor must come from the ordinary floor-eight pool")
				}
				if it.Pos.X >= gate.X {
					t.Fatal("offer floor-eight equipment before the guardian")
				}
			case itemStoryScroll, itemSideStoryScroll:
				t.Fatal("the crypt must not duplicate campaign stories")
			default:
				magic++
			}
		}
		if weapons == 0 || armors == 0 || magic == 0 || len(c.Traps) < 1 || len(c.Traps) > 4 {
			t.Fatal("crypt must have equipment, magic and one to four traps")
		}
		used := map[trapKind]bool{}
		for _, trap := range c.Traps {
			if used[trap.Kind] || occupied[trap.Pos] || trap.Spent || trap.Discovered {
				t.Fatal("traps must be distinct, hidden and clear of other features")
			}
			used[trap.Kind], kinds[trap.Kind], occupied[trap.Pos] = true, true, true
		}
		parent := g.levels[c.ParentFloor]
		reserved := parent.secretFootprint(true)
		for _, p := range []pos{c.Entrance, c.ReturnDoor} {
			if parent.Tiles[p.Y][p.X] != '#' || reserved[p] {
				t.Fatal("crypt doors must start hidden and clear of existing secret rooms")
			}
			for _, hidden := range parent.secretRooms() {
				if distance(hidden.Door, p) <= 2 {
					t.Fatal("a search must not compete between crypt and ordinary secret doors")
				}
			}
			for _, event := range parent.Events {
				if event.Shortcut != nil && *event.Shortcut == p {
					t.Fatal("crypt doors must not occupy shortcut lever panels")
				}
			}
		}
		if seed <= 10 {
			repeat := newGame(seed)
			if !reflect.DeepEqual(c, repeat.crypt) || g.rng.State != repeat.rng.State {
				t.Fatal("crypt generation must reproduce from its seed")
			}
		}
	}
	if len(kinds) != 8 || len(floors) != 4 {
		t.Fatalf("all trap types and entrance floors should occur: kinds=%v floors=%v", kinds, floors)
	}
}

func reachable8(tiles [][]rune, start, target pos) bool {
	queue := []pos{start}
	seen := map[pos]bool{start: true}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p == target {
			return true
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				n := pos{p.X + dx, p.Y + dy}
				if n.X >= 0 && n.X < mapWidth && n.Y >= 0 && n.Y < mapHeight && tiles[n.Y][n.X] != '#' && !seen[n] {
					seen[n] = true
					queue = append(queue, n)
				}
			}
		}
	}
	return false
}

func TestCryptRestrictionsAreDiscoveredThroughPlay(t *testing.T) {
	g := cryptArena()
	g.player.Pos = pos{5, 5}
	g.processCommand("k")
	var out bytes.Buffer
	g.render(&out)
	if strings.Contains(out.String(), "no fountains") || strings.Contains(out.String(), "Warding Charm") || strings.Contains(out.String(), "Blink Stone") {
		t.Fatal("entrance discovery and hints must not reveal crypt restrictions")
	}
	g.processCommand("d")
	out.Reset()
	g.render(&out)
	text := strings.Join(strings.Fields(out.String()), " ")
	if !strings.Contains(text, "Darkness limits sight to three tiles; existing wards fade.") {
		t.Fatal("entry must announce sight and fading wards")
	}
	for _, unwanted := range []string{"Crypt rules:", "no fountains", "There are no fountains", "consumed without effect"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("entry reveals unwanted rules: %q", unwanted)
		}
	}
}
