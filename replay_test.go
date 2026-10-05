package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func openArena() *game {
	g := newGame(24)
	lvl := g.current()
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			lvl.Tiles[y][x] = '.'
		}
	}
	lvl.Monsters, lvl.Items, lvl.Fountains = nil, nil, nil
	lvl.Secret = nil
	lvl.ExtraSecrets, lvl.Events, lvl.Special = nil, nil, nil
	g.player.Pos = pos{5, 5}
	return g
}

func TestMonsterGroupsVaryWithinFloorPools(t *testing.T) {
	for depth := 0; depth < levelCount; depth++ {
		groups := map[string]bool{}
		seenNew := false
		allowed := map[monsterKind]bool{}
		for _, spec := range monsterPoolForLevel(depth) {
			allowed[spec.Kind] = true
		}
		for seed := int64(1); seed <= 100; seed++ {
			g := &game{rng: newSimpleRNG(seed)}
			monsters := g.monstersForLevel(depth)
			if len(monsters) != monsterCountForDepth(depth) {
				t.Fatalf("floor %d monster quota changed", depth+1)
			}
			kinds, bosses := []string{}, 0
			for _, m := range monsters {
				if !allowed[m.Kind] || m != scaleMonsterForDepth(newMonster(m.Kind), depth) {
					t.Fatalf("floor %d spawned a monster outside its difficulty pool: %+v", depth+1, m)
				}
				if m.Boss {
					bosses++
				}
				seenNew = seenNew || m.Kind == newMonsterKindsByFloor[depth]
				kinds = append(kinds, string(m.Kind))
			}
			if (depth == levelCount-1 && bosses != 1) || (depth < levelCount-1 && bosses != 0) {
				t.Fatalf("floor %d has %d bosses", depth+1, bosses)
			}
			sort.Strings(kinds)
			groups[strings.Join(kinds, ",")] = true
			repeat := (&game{rng: newSimpleRNG(seed)}).monstersForLevel(depth)
			if !reflect.DeepEqual(monsters, repeat) {
				t.Fatal("a seed should reproduce its encounters")
			}
		}
		if len(groups) < 2 || !seenNew {
			t.Fatalf("floor %d lacks variation or its new monster: groups=%d new=%t", depth+1, len(groups), seenNew)
		}
	}
}

func TestEverySecretRoomAddsOneSideStory(t *testing.T) {
	rooms := 0
	for seed := int64(1); seed <= 30; seed++ {
		g := newGame(seed)
		for _, lvl := range g.levels {
			if lvl.Secret == nil {
				continue
			}
			rooms += len(lvl.secretRooms())
			count := 0
			positions := map[pos]bool{}
			for _, it := range lvl.Items {
				if positions[it.Pos] {
					t.Fatalf("secret treasure overlaps on floor %d", lvl.Index+1)
				}
				positions[it.Pos] = true
				if it.Kind != itemSideStoryScroll {
					continue
				}
				count++
				inside := false
				for _, hidden := range lvl.secretRooms() {
					id := hidden.SideStoryID
					if id == 0 {
						id = lvl.Index + 1
					}
					for _, p := range hidden.floorTiles() {
						inside = inside || (p == it.Pos && it.SideStoryID == id)
					}
				}
				if !inside || it.StoryChapter != 0 {
					t.Fatalf("side story must be in its secret room and outside canonical chapters: %+v", it)
				}
			}
			if count != len(lvl.secretRooms()) {
				t.Fatalf("floor %d secret room contains %d side stories", lvl.Index+1, count)
			}
			g.addSecretSideStory(lvl)
			if len(lvl.Items) != len(positions) {
				t.Fatal("adding a side story twice should not duplicate it")
			}
		}
	}
	if rooms < 30 {
		t.Fatal("expected to check secret rooms, including every final sanctum")
	}
}

func TestBookCollectsOnlyFoundStoriesAndSurvivesSave(t *testing.T) {
	g := openArena()
	g.saveFile = filepath.Join(t.TempDir(), "book.json")
	g.timestampedSaves = false
	for _, chapter := range []int{3, 1, 3} {
		g.current().Items = []item{{Pos: g.player.Pos, Kind: itemStoryScroll, StoryChapter: chapter}}
		g.collectItems()
	}
	for _, id := range []int{2, 2} {
		g.current().Items = []item{{Pos: g.player.Pos, Kind: itemSideStoryScroll, SideStoryID: id}}
		g.collectItems()
	}
	if len(g.player.StoryChapters) != 2 || len(g.player.SideStories) != 1 {
		t.Fatal("the book duplicated a collected scroll")
	}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	var book bytes.Buffer
	loaded.renderBook(&book)
	text := book.String()
	for _, want := range []string{ghostDogStory[0], ghostDogStory[2], ghostDogSideStories[1].Text} {
		if !strings.Contains(text, want) {
			t.Fatalf("book lost collected text %q", want)
		}
	}
	if strings.Contains(text, ghostDogStory[1]) {
		t.Fatal("the book exposed an uncollected story")
	}
	for id, story := range ghostDogSideStories {
		if id == 1 {
			if strings.Count(text, story.Title) != 1 || strings.Count(text, story.Text) != 1 {
				t.Fatal("a collected side story should appear exactly once")
			}
		} else if strings.Contains(text, story.Title) || strings.Contains(text, story.Text) {
			t.Fatalf("the book exposed uncollected side story %q", story.Title)
		}
	}
	if strings.Index(text, "Chapter 1/10") > strings.Index(text, "Chapter 3/10") {
		t.Fatal("canonical chapters should read in order")
	}
}

func TestReachAttackRespectsRangeWallsAndNearbyTargets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		reach   int
		target  pos
		wall    bool
		wantHit bool
	}{
		{"spear", 2, pos{7, 5}, false, true},
		{"sword", 1, pos{7, 5}, false, false},
		{"too far", 2, pos{8, 5}, false, false},
		{"wall", 2, pos{7, 5}, true, false},
		{"diagonal", 2, pos{7, 7}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.player.Weapon = weapon{Name: "Test weapon", Min: 4, Max: 4, Reach: tc.reach}
			m := newMonster(monsterOrc)
			m.Pos = tc.target
			g.current().Monsters = []*monster{&m}
			if tc.wall {
				g.current().Tiles[5][6] = '#'
			}
			cmd := "d"
			if tc.target.Y != 5 {
				cmd = "x"
			}
			g.tryMove(cmd)
			if (m.HP < m.MaxHP) != tc.wantHit {
				t.Fatalf("hit=%t, want %t", m.HP < m.MaxHP, tc.wantHit)
			}
			if tc.wantHit && g.player.Pos != (pos{5, 5}) {
				t.Fatal("a reach strike should not move the player")
			}
		})
	}
	g := openArena()
	g.player.Weapon = weapon{Min: 4, Max: 4, Reach: 2}
	near, far := newMonster(monsterOrc), newMonster(monsterOrc)
	near.Pos, far.Pos = pos{6, 5}, pos{7, 5}
	g.current().Monsters = []*monster{&near, &far}
	g.tryMove("d")
	if near.HP != near.MaxHP-4 || far.HP != far.MaxHP {
		t.Fatal("a spear should strike the nearest enemy, not pass through it")
	}
}

func TestReachAvoidsContactRetaliation(t *testing.T) {
	g := openArena()
	g.player.Weapon = weapon{Min: 1, Max: 1, Reach: 2}
	g.player.HeadArmor, g.player.BodyArmor, g.player.LegArmor = armor{}, armor{}, armor{}
	m := newMonster(monsterSlime)
	m.Pos = pos{7, 5}
	g.current().Monsters = []*monster{&m}
	g.tryMove("d")
	if g.player.HP != g.player.MaxHP {
		t.Fatal("reaching from two tiles away should avoid acid contact")
	}
	m.Pos = pos{6, 5}
	g.tryMove("d")
	if g.player.HP != g.player.MaxHP-1 {
		t.Fatal("an adjacent attack should still suffer acid retaliation")
	}
}

func TestEquipChoicesCostOneTurnAndAreRetained(t *testing.T) {
	g := openArena()
	starter := g.player.Weapon
	spear := weapon{Name: "Spear", Min: 2, Max: 4, Reach: 2}
	g.player.carryWeapon(spear)
	g.processCommand("equip 2")
	if g.player.Weapon != spear || g.stats.Turns != 1 || g.player.Weapons[0] != starter {
		t.Fatal("equipping should keep the old weapon and spend one turn")
	}
	g.processCommand("equip 2")
	g.processCommand("equip 999")
	if g.stats.Turns != 1 {
		t.Fatal("invalid or unchanged equipment should not cost a turn")
	}
	ward := armor{Name: "Rune Hood", Slot: slotHead, SpellWard: 3}
	g.player.carryArmor(ward)
	g.processCommand(fmt.Sprintf("equip %d", len(g.player.Weapons)+len(g.player.Armors)))
	if g.player.HeadArmor != ward || g.stats.Turns != 2 {
		t.Fatal("armor choices should equip their slot and spend one turn")
	}
	g.processCommand("equip 1")
	if g.player.Weapon != starter || g.stats.Turns != 3 {
		t.Fatal("the player should be able to return to earlier equipment")
	}
}

func TestArmorTradeoffsAffectCombat(t *testing.T) {
	g := openArena()
	g.player.HeadArmor, g.player.BodyArmor, g.player.LegArmor = armor{}, armor{}, armor{}
	g.player.BodyArmor = armor{SpellWard: 3, StrikeBonus: 2}
	g.hurtPlayerSpell(8, "a spell")
	if g.player.HP != g.player.MaxHP-5 {
		t.Fatal("spell ward should reduce magical damage")
	}
	g.hurtPlayer(8, "an arrow")
	if g.player.HP != g.player.MaxHP-13 {
		t.Fatal("spell ward should not reduce physical arrows")
	}
	g.player.Weapon = weapon{Min: 4, Max: 4}
	m := newMonster(monsterOrc)
	m.Pos = pos{6, 5}
	g.attackMonster(&m)
	if m.HP != m.MaxHP-6 {
		t.Fatal("hunter armor should add strike damage")
	}
}

func TestNewMonsterDefensesAndPowers(t *testing.T) {
	for _, kind := range []monsterKind{monsterAshBeetle, monsterChapelSentinel, monsterIronrootBrute} {
		g := openArena()
		m := newMonster(kind)
		g.damageMonster(&m, 5, "hits")
		if m.HP != m.MaxHP-5+monsterGuard(kind) {
			t.Fatalf("%s did not block its advertised damage", kind)
		}
	}
	g := openArena()
	brute := newMonster(monsterIronrootBrute)
	brute.Pos = pos{6, 5}
	g.monsterTurn(&brute)
	hp := g.player.HP
	g.monsterTurn(&brute)
	if g.player.HP != hp {
		t.Fatal("Ironroot Brute should rest on its second turn")
	}
	knight := newMonster(monsterCrownlessKnight)
	g.player.ShieldTurns = 3
	g.attackPlayer(&knight)
	if g.player.ShieldTurns != 0 {
		t.Fatal("Crownless Knight should break the ward")
	}
	for _, kind := range []monsterKind{monsterChainImp, monsterLanternWisp, monsterGloomArcher, monsterBellRevenant, monsterRiftWeaver} {
		acted := false
		for seed := int64(1); seed <= 100 && !acted; seed++ {
			g := openArena()
			g.rng = newSimpleRNG(seed)
			m := newMonster(kind)
			m.Pos = pos{7, 5}
			acted = g.newMonsterPower(&m)
		}
		if !acted {
			t.Fatalf("%s never used its special power", kind)
		}
	}
}

func TestRunCountersCountActionsKillsAndPermanentDogFalls(t *testing.T) {
	g := openArena()
	g.current().HasStair = false
	g.processCommand("h")
	g.processCommand(">")
	g.processCommand("equip 999")
	if g.stats.Turns != 0 {
		t.Fatal("information and failed actions should not count")
	}
	g.processCommand(".")
	g.player.WebbedTurns = 1
	g.processCommand("d")
	if g.stats.Turns != 2 {
		t.Fatal("waiting and losing a move to webbing should count")
	}
	rat := newMonster(monsterRat)
	g.damageMonster(&rat, rat.HP, "burns")
	g.damageMonster(&rat, 100, "hits")
	rat = newMonster(monsterRat)
	rat.HP = 1
	g.dogAttack(&rat)
	if g.stats.MonstersKilled != 2 {
		t.Fatal("kills should count once, including Ghost Dog's kills")
	}
	g.dog.Freed, g.dog.HP = true, 1
	g.hurtDog(100, "a monster")
	g.hurtDog(100, "another monster")
	g.restoreGhostDogAfterLichDeath()
	if g.stats.GhostDogFalls != 1 || !g.dog.Alive {
		t.Fatal("revival should preserve the failed safe-escape challenge")
	}
}

func TestChallengeRecordsPersistBestResultsAcrossRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.json")
	finish := func(seed int64, turns, kills, falls int, escaped bool) {
		t.Helper()
		g := newGame(seed)
		g.recordsFile = path
		g.stats = runStats{Turns: turns, MonstersKilled: kills, GhostDogFalls: falls, HistoryKnown: true}
		g.won, g.dog.Freed = escaped, true
		if !escaped {
			g.player.HP = 0
		}
		if err := g.recordChallenges(); err != nil {
			t.Fatal(err)
		}
	}
	finish(1, 80, 5, 0, true)
	finish(2, 60, 20, 1, true)
	finish(3, 12, 25, 0, false)
	records, err := readChallengeRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	if records.FewestTurns.Turns != 60 || records.SafeEscape.Turns != 80 || records.MostKills.Kills != 25 || records.MostKills.Escaped {
		t.Fatalf("wrong records: %+v", records)
	}
	g := newGame(4)
	g.recordsFile = path
	g.stats.MonstersKilled = 100
	g.quit = true
	if err := g.recordChallenges(); err != nil {
		t.Fatal(err)
	}
	unchanged, _ := readChallengeRecords(path)
	if !reflect.DeepEqual(records, unchanged) {
		t.Fatal("an unfinished run should not set a record")
	}
}

func TestSavePreservesNewFeaturesAndLegacySavesRemainPlayable(t *testing.T) {
	g := newGameWithSaveFile(15, filepath.Join(t.TempDir(), "save.json"))
	g.player.carryWeapon(weapon{Name: "Spear", Min: 3, Max: 5, Reach: 2})
	g.player.Weapon = g.player.Weapons[1]
	g.player.carryArmor(armor{Name: "Rune Hood", Slot: slotHead, SpellWard: 3, Rarity: rarityRare})
	g.player.HeadArmor = g.player.Armors[len(g.player.Armors)-1]
	g.player.SideStories, g.player.StoryChapters = []int{2}, []int{1, 3}
	g.stats = runStats{Turns: 34, MonstersKilled: 7, GhostDogFalls: 1, HistoryKnown: true}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.player, loaded.player) || g.stats != loaded.stats || g.recordsFile != loaded.recordsFile {
		t.Fatalf("save/load lost state: before=%+v after=%+v stats=%+v/%+v records=%s/%s", g.player, loaded.player, g.stats, loaded.stats, g.recordsFile, loaded.recordsFile)
	}
	data, _ := os.ReadFile(g.saveFile)
	var legacy map[string]json.RawMessage
	if err := json.Unmarshal(data, &legacy); err != nil {
		t.Fatal(err)
	}
	delete(legacy, "stats")
	delete(legacy, "records_file")
	var levels []*level
	if err := json.Unmarshal(legacy["levels"], &levels); err != nil {
		t.Fatal(err)
	}
	for _, lvl := range levels {
		kept := lvl.Items[:0]
		for _, it := range lvl.Items {
			if it.Kind != itemSideStoryScroll {
				kept = append(kept, it)
			}
		}
		lvl.Items = kept
	}
	legacy["levels"], _ = json.Marshal(levels)
	var p map[string]json.RawMessage
	json.Unmarshal(legacy["player"], &p)
	delete(p, "weapons")
	delete(p, "armors")
	delete(p, "side_stories")
	legacy["player"], _ = json.Marshal(p)
	data, _ = json.Marshal(legacy)
	if err := os.WriteFile(g.saveFile, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err = loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.stats.HistoryKnown || len(loaded.player.Weapons) != 1 || len(loaded.player.Armors) != 3 || len(loaded.player.StoryChapters) != 2 {
		t.Fatal("legacy saves should retain gear and chapters without inventing challenge history")
	}
	for _, lvl := range loaded.levels {
		if lvl.Secret == nil {
			continue
		}
		count := 0
		for _, it := range lvl.Items {
			if it.Kind == itemSideStoryScroll {
				count++
			}
		}
		if count != len(lvl.secretRooms()) {
			t.Fatal("loading an older save should supply its secret rooms with side stories")
		}
	}
}

func TestLegacyLegArmorRemainsUsableAndSavesWithNewNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "armor.json")
	g := openArena()
	g.saveFile, g.timestampedSaves = path, false
	g.player.LegArmor = armor{Name: "Runespun leg armor", Slot: slotLeg, Defense: 4, SpellWard: 2, Rarity: rarityRare}
	g.player.ensureCarriedEquipment()
	found := armor{Name: "Hunter's leg armor", Slot: slotLeg, Defense: 2, StrikeBonus: 3, Rarity: rarityRare}
	loot := makeArmorItem(found)
	loot.Pos = pos{6, 5}
	g.current().Items = []item{loot}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.ReplaceAll(data, []byte("\"leg_armor\""), []byte("\"feet_armor\""))
	data = bytes.ReplaceAll(data, []byte("\"slot\": \"leg\""), []byte("\"slot\": \"feet\""))
	data = bytes.ReplaceAll(data, []byte(" leg armor"), []byte(" feet armor"))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.player, g.player) || loaded.player.armorDefense() != 6 || loaded.player.spellWard() != 2 || !reflect.DeepEqual(loaded.current().Items, g.current().Items) {
		t.Fatal("old equipped, carried, and floor armor should retain their properties with the new leg names")
	}
	var stdout, stderr bytes.Buffer
	commands := fmt.Sprintf("i\nd\nequip %d\ni\nsave\nquit\n", len(g.player.Weapons)+len(g.player.Armors)+1)
	args := []string{"--load-file", path, "--records-file", filepath.Join(dir, "records.json")}
	if err := run(args, strings.NewReader(commands), &stdout, &stderr); err != nil {
		t.Fatalf("run legacy armor commands: %v\n%s", err, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "Leg:") || !strings.Contains(output, "You equip Hunter's leg armor") || strings.Contains(output, "Feet:") || strings.Contains(output, "feet armor") || strings.Contains(output, "feet slot") {
		t.Fatal("inventory, pickup, and equip text should consistently use the leg armor name")
	}
	loaded, err = loadGame(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.player.LegArmor != found || loaded.player.strikeBonus() != 3 || loaded.stats.Turns != 2 {
		t.Fatal("migrated floor armor should be collectable, equippable, and saveable in the leg slot")
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("\"leg_armor\"")) || bytes.Contains(data, []byte("\"feet_armor\"")) || bytes.Contains(data, []byte("\"slot\": \"feet\"")) || bytes.Contains(data, []byte(" feet armor")) {
		t.Fatal("new saves should write only the new armor field, slot, and names")
	}
}

func TestRunBookEquipmentChallengesAndVictory(t *testing.T) {
	dir := t.TempDir()
	testHome := filepath.Join(dir, "home")
	if err := os.Mkdir(testHome, 0700); err != nil {
		t.Fatal(err)
	}
	testEnv := append(os.Environ(), "HOME="+testHome)
	recordsPath := filepath.Join(testHome, ".ghost-dog-data.json")
	path := filepath.Join(dir, "save.json")
	g := newGameWithSaveFile(31, path)
	g.currentLevel = levelCount - 1
	lvl := g.current()
	g.revealSecretRoom(lvl)
	lvl.Monsters = nil
	g.player.Pos = lvl.EscapeStairs
	g.dog.Freed = true
	g.placeDogNearPlayer()
	g.player.StoryChapters, g.player.SideStories = []int{1, 3}, []int{2}
	g.player.carryWeapon(weapon{Name: "Test Spear", Min: 3, Max: 5, Reach: 2})
	g.stats = runStats{Turns: 23, MonstersKilled: 7, HistoryKnown: true}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	commands := "book\nequipment\nequip 2\nsave\nload\nchallenges\n<\n"
	binary := filepath.Join(dir, "ghostdog")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build executable: %v\n%s", err, output)
	}
	command := exec.Command(binary, "--load", "--save-file", path)
	command.Env = testEnv
	command.Stdin, command.Stdout, command.Stderr = strings.NewReader(commands), &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("run executable: %v\n%s", err, stderr.String())
	}
	for _, want := range []string{"=== Book of Ghost Dog ===", ghostDogStory[0], ghostDogSideStories[1].Text, "You equip Test Spear", "Victory!", "Fewest turns record: 25", "Ghost Dog safe escape record: achieved", "Most monsters killed record: 7"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("game CLI did not show %q", want)
		}
	}
	for id, story := range ghostDogSideStories {
		if id != 1 && (strings.Contains(stdout.String(), story.Title) || strings.Contains(stdout.String(), story.Text)) {
			t.Fatalf("the executable exposed uncollected side story %q", story.Title)
		}
	}
	output := stdout.String()
	if strings.Index(output, "=== Personal Records ===") > strings.Index(output, "=== Ghost Dog Dungeon:") || strings.LastIndex(output, "=== Personal Records ===") < strings.Index(output, "Victory!") {
		t.Fatal("personal records should appear at startup and after victory")
	}
	records, err := readChallengeRecords(recordsPath)
	if err != nil || records.FewestTurns == nil || records.FewestTurns.Turns != 25 {
		t.Fatalf("victory did not persist its final action: records=%+v err=%v", records, err)
	}
	for key, want := range map[byte]string{'B': "book", 'E': "equipment", 'j': "challenges"} {
		cmd, _, ok := rawKeyCommand(key)
		if !ok || cmd != want {
			t.Fatalf("immediate key %q did not open %s", key, want)
		}
	}
	stdout.Reset()
	stderr.Reset()
	freshSave := filepath.Join(dir, "fresh.json")
	command = exec.Command(binary, "--seed", "1", "--save-file", freshSave)
	command.Env = testEnv
	command.Stdin = strings.NewReader("book\ni\ndrop 1\ndrop 5\nsave\nquit\n")
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("run drop commands: %v\n%s", err, stderr.String())
	}
	output = stdout.String()
	if strings.Contains(output, "Search secret rooms for extra scrolls.") || !strings.Contains(output, "5. Healing Potion x3") || !strings.Contains(output, "cannot drop an equipped item") || !strings.Contains(output, "You drop Healing Potion") {
		t.Fatal("the executable did not show the updated book and numbered drop behavior")
	}
	for _, story := range ghostDogSideStories {
		if strings.Contains(output, story.Title) || strings.Contains(output, story.Text) {
			t.Fatalf("a fresh game's book exposed side story %q", story.Title)
		}
	}
	fresh, err := loadGame(freshSave)
	if err != nil || fresh.player.Potions != 2 || fresh.stats.Turns != 1 {
		t.Fatalf("drop command did not survive the CLI save: err=%v", err)
	}
	if !strings.Contains(output, "Fewest turns record: 25") || strings.Count(output, "=== Personal Records ===") != 1 {
		t.Fatal("a new game should show the previous game's records at startup")
	}
	unchanged, err := readChallengeRecords(recordsPath)
	if err != nil || !reflect.DeepEqual(records, unchanged) {
		t.Fatal("saving or quitting should not submit an unfinished run")
	}
	deathSave := filepath.Join(dir, "death.json")
	death := openArena()
	death.saveFile, death.timestampedSaves = deathSave, false
	// An older save's record path must not override the current user's home.
	death.recordsFile = filepath.Join(dir, "ghostdog-records.json")
	death.player.HP = 1
	death.stats = runStats{Turns: 10, MonstersKilled: 12, HistoryKnown: true}
	attacker := newMonster(monsterOrc)
	attacker.Pos, attacker.MinDamage, attacker.MaxDamage = pos{6, 5}, 5, 5
	death.current().Monsters = []*monster{&attacker}
	if err := death.save(); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	command = exec.Command(binary, "--load-file", deathSave)
	command.Env = testEnv
	command.Stdin = strings.NewReader(".\n")
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("run death without saving: %v\n%s", err, stderr.String())
	}
	output = stdout.String()
	deathIndex := strings.Index(output, "You fall in the dark.")
	if deathIndex < 0 || strings.Count(output, "=== Personal Records ===") != 2 || strings.LastIndex(output, "Most monsters killed record: 12") < deathIndex {
		t.Fatal("death should automatically show the updated personal records")
	}
	records, err = readChallengeRecords(recordsPath)
	if err != nil || records.MostKills == nil || records.MostKills.Kills != 12 || records.MostKills.Turns != 11 || records.MostKills.Escaped || records.FewestTurns.Turns != 25 {
		t.Fatalf("death should automatically update home-directory records: %+v, err=%v", records, err)
	}
	if _, err := os.Stat(death.recordsFile); !os.IsNotExist(err) {
		t.Fatal("loaded games should not write records beside an older save")
	}
	eventSave := filepath.Join(dir, "event.json")
	eventGame := openArena()
	eventGame.saveFile, eventGame.timestampedSaves = eventSave, false
	eventGame.current().Events = []floorEvent{{Kind: eventGhost, Pos: pos{6, 5}}}
	if err := eventGame.save(); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	command = exec.Command(binary, "--load-file", eventSave)
	command.Env = testEnv
	command.Stdin = strings.NewReader("y\nchoose 2\nsave\nload\ny\nsave\nquit\n")
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("run executable event: %v\n%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "offers one gift") || !strings.Contains(stdout.String(), "one blink stone") || !strings.Contains(stdout.String(), "nothing nearby") {
		t.Fatal("the executable did not complete the one-use event flow")
	}
	eventLoaded, err := loadGame(eventSave)
	if err != nil || eventLoaded.player.BlinkStones != 1 || eventLoaded.stats.Turns != 1 || !eventLoaded.current().Events[0].Used {
		t.Fatal("the executable should preserve the gift and its one-turn cost after loading")
	}
}
