package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestMapSymbolsAreUniqueAcrossPhysicalCategoriesMonstersAndTraps(t *testing.T) {
	seen := map[rune]string{}
	claim := func(ch rune, name string) {
		t.Helper()
		if ch < '!' || ch > '~' {
			t.Fatalf("%s has a missing or non-ASCII symbol %q", name, ch)
		}
		if previous, ok := seen[ch]; ok {
			t.Fatalf("symbol %q represents both %s and %s", ch, previous, name)
		}
		seen[ch] = name
	}
	for _, feature := range []struct {
		ch   rune
		name string
	}{
		{glyphWall, "wall"}, {glyphFloor, "floor"}, {glyphPlayer, "player"},
		{glyphDog, "Ghost Dog"}, {glyphChainedDog, "chained Ghost Dog"},
		{glyphStairsDown, "stairs down"}, {glyphStairsUp, "stairs up/out"},
		{glyphFountain, "fountain"}, {glyphSpentFountain, "spent fountain"},
		{glyphEpicWarning, "Epic warning"}, {glyphCryptEntrance, "crypt entrance"},
		{glyphCryptExit, "crypt exit"}, {glyphSealedCrypt, "sealed crypt entrance"},
	} {
		claim(feature.ch, feature.name)
	}
	for _, kind := range []eventKind{eventGhost, eventKeepsake, eventLever, eventShrine} {
		e := floorEvent{Kind: kind}
		claim(e.glyph(), string(kind))
	}
	for _, kind := range []objectiveKind{objectiveSpirit, objectiveSigils} {
		o := objectiveRoom{Kind: kind, Pos: pos{5, 5}, Sigils: []ritualSigil{{Pos: pos{5, 5}}}}
		claim(o.marker(o.Pos), string(kind))
	}
	if len(monsterGlyphs) != len(allMonsterKinds) {
		t.Fatal("the symbol catalog must cover the complete monster roster")
	}
	for _, kind := range allMonsterKinds {
		m := newMonster(kind)
		claim(m.glyph(), string(kind))
	}
	if len(itemGlyphs) != int(itemSideStoryScroll)+1 || len(trapGlyphs) != len(cryptTrapKinds) {
		t.Fatal("the symbol catalog must cover all item and trap kinds")
	}
	for kind := itemWeapon; kind <= itemSideStoryScroll; kind++ {
		it := item{Kind: kind}
		if previous, exists := seen[it.glyph()]; exists && previous == itemMapLabel(it) {
			continue
		}
		claim(it.glyph(), itemMapLabel(it))
	}
	for _, kind := range cryptTrapKinds {
		claim((cryptTrap{Kind: kind}).glyph(), "trap: "+string(kind))
	}
	if seen['F'] != "fountain" || seen['f'] != "spent fountain" {
		t.Fatal("F and f must remain reserved for fountains")
	}
	t.Logf("audited %d distinct map symbols", len(seen))
}

func concealedFloorItems() []item {
	return []item{
		{Kind: itemFireScroll, Name: "fire scroll"},
		{Kind: itemGhostRecallScroll, Name: "Ghost Dog recall scroll"},
		{Kind: itemStoryScroll, Name: "Ghost Dog story scroll 1", StoryChapter: 1},
		{Kind: itemSideStoryScroll, Name: "side-story scroll", SideStoryID: 1},
		{Kind: itemSunOrb, Name: "sun orb"},
		{Kind: itemStarfireOrb, Name: "starfire orb"},
		{Kind: itemPotion, Name: "healing potion"},
		{Kind: itemPhoenixAsh, Name: "Phoenix Ash"},
		{Kind: itemWardingCharm, Name: "warding charm"},
		{Kind: itemFrostCharm, Name: "frost charm"},
		{Kind: itemBlinkStone, Name: "blink stone"},
		{Kind: itemWeapon, Name: "Short Sword", Weapon: weapon{Name: "Short Sword", Min: 4, Max: 7}},
		{Kind: itemArmor, Name: "Leather Jerkin", Armor: armor{Name: "Leather Jerkin", Slot: slotBody, Defense: 2, Rarity: rarityUncommon}},
	}
}

func assertFloorItemIdentitiesHidden(t *testing.T, output string) {
	t.Helper()
	for _, identity := range []string{"fire scroll", "Ghost Dog recall scroll", "Ghost Dog story scroll", "side-story scroll", "sun orb", "starfire orb", "healing potion", "Phoenix Ash", "warding charm", "frost charm", "blink stone", "Short Sword", "Leather Jerkin", ghostDogStory[0], ghostDogSideStories[0].Title} {
		if strings.Contains(output, identity) {
			t.Fatalf("floor presentation disclosed %q", identity)
		}
	}
}

func TestPhysicalAppearancesConcealFloorIdentities(t *testing.T) {
	g := openArena()
	g.messages = nil
	for i, it := range concealedFloorItems() {
		it.Pos = pos{6 + i%3, 5 + i/3}
		g.current().Items = append(g.current().Items, it)
	}
	rng, stats := g.rng.State, g.stats
	var out bytes.Buffer
	g.render(&out)
	g.renderInspect(&out)
	assertFloorItemIdentitiesHidden(t, out.String())
	rows := symbolMapRows(t, out.String())
	for _, it := range g.current().Items {
		var want rune
		switch it.Kind {
		case itemFireScroll, itemGhostRecallScroll, itemStoryScroll, itemSideStoryScroll:
			want = '?'
		case itemSunOrb, itemStarfireOrb:
			want = '='
		case itemPotion, itemPhoenixAsh:
			want = '!'
		case itemWardingCharm, itemFrostCharm:
			want = '%'
		case itemBlinkStone:
			want = '*'
		case itemWeapon:
			want = ')'
		case itemArmor:
			want = '['
		}
		if it.glyph() != want || rune(rows[it.Pos.Y][it.Pos.X]) != want {
			t.Fatalf("%s must use physical appearance %c", it.Name, want)
		}
	}
	var legend bytes.Buffer
	g.renderLegend(&legend)
	for _, label := range []string{"? scroll", "= orb", "! potion", "% charm"} {
		if strings.Count(legend.String(), label) != 1 {
			t.Fatalf("legend must list %s once", label)
		}
	}
	var help bytes.Buffer
	g.renderHelp(&help, false)
	if strings.Count(help.String(), "\n? scroll\n") != 1 || strings.Count(help.String(), "\n= orb\n") != 1 {
		t.Fatal("help must list each category symbol once")
	}
	if g.rng.State != rng || g.stats != stats {
		t.Fatal("concealing identities cannot draw RNG or spend turns")
	}
}

func TestPickupRevealsPhysicalItemIdentityAfterSaveLoad(t *testing.T) {
	pickupMessages := map[itemKind]string{
		itemPotion:            "You stash a healing potion; press p to drink it.",
		itemFireScroll:        "You pocket a fire scroll; press f to read it.",
		itemBlinkStone:        "You take a blink stone; press g to use it.",
		itemWardingCharm:      "You take a warding charm; press b to use it.",
		itemSunOrb:            "You cradle a sun orb; press u to use it.",
		itemFrostCharm:        "You take a frost charm; press t to use it.",
		itemStarfireOrb:       "You take a starfire orb; press o to use it.",
		itemPhoenixAsh:        "You gather phoenix ash; press n to use it.",
		itemGhostRecallScroll: "You find a Ghost Dog recall scroll; press r to read it.",
	}
	for _, it := range concealedFloorItems() {
		t.Run(it.Name, func(t *testing.T) {
			g := openArena()
			g.saveDir = t.TempDir()
			g.messages = nil
			it.Pos = pos{6, 5}
			g.current().Items = []item{it}
			if err := g.save(); err != nil {
				t.Fatal(err)
			}
			loaded, err := loadGame(g.saveFile)
			if err != nil {
				t.Fatal(err)
			}
			var before bytes.Buffer
			loaded.render(&before)
			loaded.renderInspect(&before)
			assertFloorItemIdentitiesHidden(t, before.String())
			if !reflect.DeepEqual(loaded.current().Items, g.current().Items) || loaded.rng.State != g.rng.State {
				t.Fatal("loading must preserve actual identities without disclosing or rerolling them")
			}
			loaded.processCommand("d")
			if len(loaded.current().Items) != 0 || loaded.stats.Turns != 1 {
				t.Fatal("pickup must remain an ordinary movement action")
			}
			if want := pickupMessages[it.Kind]; want != "" && !reflect.DeepEqual(loaded.messages, []string{"You move east.", want}) {
				t.Fatalf("pickup must identify the consumable and its key without explaining effects: %q", loaded.messages)
			}
			var after bytes.Buffer
			loaded.renderInventory(&after)
			loaded.renderBook(&after)
			text := strings.Join(loaded.messages, "\n") + after.String()
			identity := it.Name
			if it.Kind == itemStoryScroll {
				identity = ghostDogStory[0]
			}
			if it.Kind == itemSideStoryScroll {
				identity = ghostDogSideStories[0].Title
			}
			if !strings.Contains(strings.ToLower(text), strings.ToLower(identity)) {
				t.Fatalf("pickup must reveal %q", identity)
			}
			if err := loaded.save(); err != nil {
				t.Fatal(err)
			}
			reloaded, err := loadGame(loaded.saveFile)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(reloaded.player, loaded.player) || reloaded.rng.State != loaded.rng.State {
				t.Fatal("identified inventory must survive saving exactly")
			}
		})
	}
}

func TestGenericFloorAppearancesThroughExecutable(t *testing.T) {
	if testing.Short() {
		t.Skip("compiled appearance and pickup replay")
	}
	g := openArena()
	g.saveDir = t.TempDir()
	g.messages = nil
	items := concealedFloorItems()
	for i := range items {
		items[i].Pos = pos{6, 5}
	}
	g.current().Items = items
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader("m\nsave\nload\nm\nd\ni\nbook\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("appearance replay: %v\n%s", err, output)
	}
	text := string(output)
	before, after, ok := strings.Cut(text, "You move east.")
	if !ok {
		t.Fatal("the replay must reach pickup")
	}
	assertFloorItemIdentitiesHidden(t, before)
	after = strings.Join(strings.Fields(after), " ")
	if !strings.Contains(before, "unidentified scroll") {
		t.Fatal("inspection must describe an unidentified scroll")
	}
	for _, identity := range []string{"fire scroll", "Ghost Dog recall scroll", "sun orb", "starfire orb", ghostDogStory[0], ghostDogSideStories[0].Title} {
		if !strings.Contains(after, identity) {
			t.Fatalf("pickup must reveal %q", identity)
		}
	}
	loaded, err := loadGame(lastSavePath(t, text))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.player.FireScrolls != g.player.FireScrolls+1 || loaded.player.GhostRecallScrolls != 1 || loaded.player.SunOrbs != 1 || loaded.player.StarfireOrbs != 1 || len(loaded.player.StoryChapters) != 1 || len(loaded.player.SideStories) != 1 || len(loaded.current().Items) != 0 {
		t.Fatal("shared symbols cannot merge or change item identities")
	}
}

func TestMonsterDropMessagesConcealItemIdentities(t *testing.T) {
	for seed := int64(1); seed <= 40; seed++ {
		g := openArena()
		g.rng = newSimpleRNG(seed)
		g.messages = nil
		m := newMonster(monsterStormLeviathan)
		m.Pos = pos{7, 5}
		g.dropEpicTreasure(&m)
		reward := g.current().Items[0]
		text := strings.Join(g.messages, "\n")
		assertFloorItemIdentitiesHidden(t, text)
		if strings.Contains(text, reward.Name) {
			t.Fatal("drop announcement must conceal exact reward name")
		}
		if !strings.Contains(text, itemMapLabel(reward)) {
			t.Fatal("drop announcement must still name the physical category")
		}
	}
	g := openArena()
	g.messages = nil
	m := newMonster(monsterBoss)
	m.StoryChapter = 10
	m.Pos = pos{7, 5}
	g.current().Monsters = []*monster{&m}
	g.damageMonster(&m, 10000, "burns")
	text := strings.Join(g.messages, "\n") + monsterPower(monsterBoss)
	if strings.Contains(text, "tenth") || strings.Contains(text, "story scroll") || !strings.Contains(text, "drops a scroll (?)") {
		t.Fatal("boss inspection and drops must not identify the floor scroll")
	}
}

func symbolMapRows(t *testing.T, output string) []string {
	t.Helper()
	var rows []string
	mapStarted := false
	for _, line := range strings.Split(stripANSI(output), "\n") {
		if strings.HasPrefix(line, "=== Ghost Dog Dungeon:") {
			mapStarted = true
		}
		if mapStarted && len(line) == mapWidth && (line[0] == '#' || line[0] == ' ') {
			rows = append(rows, line)
			if len(rows) == mapHeight {
				return rows
			}
		}
	}
	t.Fatalf("missing map rows in output:\n%s", output)
	return nil
}

func TestSymbolMapRowsIgnoreOpeningArt(t *testing.T) {
	g := openArena()
	var mapOutput, openingOutput bytes.Buffer
	g.render(&mapOutput)
	renderOpeningArt(&openingOutput)
	// Width and wall-like characters alone do not identify a dungeon row.
	fmt.Fprintln(&openingOutput, strings.Repeat("#", mapWidth))
	openingOutput.WriteString(mapOutput.String())
	if !reflect.DeepEqual(symbolMapRows(t, openingOutput.String()), symbolMapRows(t, mapOutput.String())) {
		t.Fatal("map assertions must ignore artwork before the dungeon heading")
	}
}

func TestEveryMonsterAndItemDisplaysItsCurrentSymbol(t *testing.T) {
	g := openArena()
	for i, kind := range allMonsterKinds {
		m := newMonster(kind)
		m.Pos = pos{2 + i%30, 2 + i/30}
		g.current().Monsters = append(g.current().Monsters, &m)
	}
	for kind := itemWeapon; kind <= itemSideStoryScroll; kind++ {
		g.current().Items = append(g.current().Items, item{Kind: kind, Pos: pos{2 + int(kind), 7}})
	}
	rng, turns := g.rng.State, g.stats.Turns
	var output bytes.Buffer
	g.render(&output)
	rows := symbolMapRows(t, output.String())
	for _, m := range g.current().Monsters {
		if rune(rows[m.Pos.Y][m.Pos.X]) != monsterGlyphs[m.Kind] {
			t.Fatalf("%s should display the symbol for its kind", m.Kind)
		}
	}
	for _, it := range g.current().Items {
		if rune(rows[it.Pos.Y][it.Pos.X]) != itemGlyphs[it.Kind] {
			t.Fatalf("item kind %d should display the symbol for its kind", it.Kind)
		}
	}
	if g.rng.State != rng || g.stats.Turns != turns {
		t.Fatal("displaying the current symbols must not draw randomness or spend turns")
	}
}

func TestDiscoveredTrapsDisplayDistinctSymbolsAndSpentTrapsDisappear(t *testing.T) {
	for _, kind := range cryptTrapKinds {
		t.Run(string(kind), func(t *testing.T) {
			g := trappedArena(kind)
			g.crypt.Traps[0].Discovered = true
			p := g.crypt.Traps[0].Pos
			var output bytes.Buffer
			g.render(&output)
			rows := symbolMapRows(t, output.String())
			if rune(rows[p.Y][p.X]) != trapGlyphs[kind] || !strings.Contains(output.String(), fmt.Sprintf("Trap: %s (%c)", kind, trapGlyphs[kind])) {
				t.Fatal("the map and trap description must identify the same discovered hazard")
			}
			g.crypt.Traps[0].Spent = true
			output.Reset()
			g.render(&output)
			if symbolMapRows(t, output.String())[p.Y][p.X] != '.' {
				t.Fatal("a spent trap must no longer appear as a hazard")
			}
		})
	}
}

func TestMapSymbolsThroughExecutableAndSaveLoad(t *testing.T) {
	g := openArena()
	m := newMonster(monsterFrostMonarch)
	m.Pos = pos{8, 5}
	g.current().Monsters = []*monster{&m}
	g.current().Fountains = []fountain{{Pos: pos{6, 5}}, {Pos: pos{6, 6}, Used: true}}
	g.current().Items = []item{{Kind: itemWardingCharm, Name: "warding charm", Pos: pos{7, 5}}}
	g.player.WardingCharms = 1
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader("m\nc\ni\nsave\nload\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executable: %v\n%s", err, output)
	}
	rows := symbolMapRows(t, string(output))
	if rows[5][6] != 'F' || rows[6][6] != 'f' || rows[5][7] != '%' || rows[5][8] != '5' {
		t.Fatal("the fountain, spent fountain, warding charm and Frost Monarch must be visually distinct after loading")
	}
	plainOutput := strings.Join(strings.Fields(stripANSI(string(output))), " ")
	for _, want := range []string{"Epic Frost Monarch (5)", "- 5 Epic Frost Monarch", "Warding Charm x1 (map %)", "F fountain", "f spent fountain"} {
		if !strings.Contains(plainOutput, want) {
			t.Fatalf("map, inspection, inventory and codex must use consistent symbols: missing %q", want)
		}
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.levels, g.levels) || !reflect.DeepEqual(loaded.player, g.player) || loaded.dog != g.dog || loaded.stats != g.stats || loaded.rng.State != g.rng.State {
		t.Fatal("displaying and saving a run must preserve gameplay state")
	}
}

func TestNearbyLegendIncludesUnfreedGhostDog(t *testing.T) {
	for _, tc := range []struct {
		name   string
		chain  pos
		freed  bool
		hidden bool
		want   bool
	}{
		{"nearby chained star", pos{6, 5}, false, false, true},
		{"at nearby boundary", pos{9, 5}, false, false, true},
		{"too far", pos{10, 5}, false, false, false},
		{"concealed", pos{6, 5}, false, true, false},
		{"already rescued", pos{6, 5}, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.current().DogChain = &tc.chain
			g.dog.Freed = tc.freed
			if tc.hidden {
				g.current().Tiles[tc.chain.Y][tc.chain.X] = '#'
			}
			dog, rng, turns := g.dog, g.rng.State, g.stats.Turns
			var out bytes.Buffer
			g.renderLegend(&out)
			text := strings.Join(strings.Fields(stripANSI(out.String())), " ")
			if got := strings.Contains(text, "d chained Ghost Dog"); got != tc.want {
				t.Fatalf("legend contains chained Ghost Dog=%t, want %t: %s", got, tc.want, text)
			}
			if g.dog != dog || g.rng.State != rng || g.stats.Turns != turns {
				t.Fatal("rendering must not change companion or gameplay state")
			}
		})
	}
}
