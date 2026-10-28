package main

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func testFriendlyGhost(p pos) floorEvent {
	return floorEvent{Kind: eventGhost, Pos: p, Gifts: []item{
		{Kind: itemPotion, Name: "healing potion"},
		{Kind: itemBlinkStone, Name: "blink stone"},
		{Kind: itemGhostRecallScroll, Name: "Ghost Dog recall scroll"},
	}}
}

func TestFriendlyGhostOffersOneChoiceWithoutSpendingAReadingTurn(t *testing.T) {
	for _, choice := range []string{"choose 1", "choose 2", "choose 3"} {
		g := openArena()
		g.current().Events = []floorEvent{testFriendlyGhost(pos{6, 5})}
		potions, stones, recalls := g.player.Potions, g.player.BlinkStones, g.player.GhostRecallScrolls
		g.processCommand("y")
		if g.stats.Turns != 0 || g.current().Events[0].Used || !strings.Contains(strings.Join(g.messages, " "), "offers one gift") {
			t.Fatal("reading the choice should be free")
		}
		g.processCommand(choice)
		if g.stats.Turns != 1 || !g.current().Events[0].Used {
			t.Fatal("accepting one gift should spend one turn")
		}
		if (choice == "choose 1" && (g.player.Potions != potions+1 || g.player.BlinkStones != stones || g.player.GhostRecallScrolls != recalls)) ||
			(choice == "choose 2" && (g.player.Potions != potions || g.player.BlinkStones != stones+1 || g.player.GhostRecallScrolls != recalls)) ||
			(choice == "choose 3" && (g.player.Potions != potions || g.player.BlinkStones != stones || g.player.GhostRecallScrolls != recalls+1)) {
			t.Fatal("the ghost should grant only the chosen gift")
		}
		before := g.player
		g.processCommand(choice)
		if !reflect.DeepEqual(before, g.player) || g.stats.Turns != 1 {
			t.Fatal("a gift cannot be collected again")
		}
	}
}

func TestHealingShrineAndKeepsakeSupportOnlyALivingCompanion(t *testing.T) {
	g := openArena()
	g.current().Events = []floorEvent{{Kind: eventKeepsake, Pos: pos{6, 5}}}
	g.processCommand("y")
	if g.current().Events[0].Used || g.stats.Turns != 0 {
		t.Fatal("the keepsake should wait for Ghost Dog")
	}
	g.dog.Freed, g.dog.Alive, g.dog.HP, g.dog.Pos = true, false, 0, pos{5, 4}
	g.processCommand("y")
	if g.dog.Alive || g.current().Events[0].Used || g.stats.Turns != 0 {
		t.Fatal("a keepsake should not revive a fallen Ghost Dog")
	}
	g.dog.Alive, g.dog.HP = true, g.dog.MaxHP-5
	wards := g.player.WardingCharms
	g.processCommand("y")
	if g.dog.HP != g.dog.MaxHP || g.player.WardingCharms != wards+1 || g.stats.Turns != 1 {
		t.Fatal("recognizing the keepsake should heal Ash and grant one ward")
	}
	g.current().Events = []floorEvent{{Kind: eventShrine, Pos: pos{6, 5}}}
	g.processCommand("y")
	if g.current().Events[0].Used || g.stats.Turns != 1 {
		t.Fatal("a shrine should wait while both companions are healthy")
	}
	g.player.HP, g.dog.HP = g.player.MaxHP-4, g.dog.MaxHP-6
	g.processCommand("y")
	if g.player.HP != g.player.MaxHP || g.dog.HP != g.dog.MaxHP || !g.current().Events[0].Used || g.stats.Turns != 2 {
		t.Fatal("the shrine should heal both companions once and cap health")
	}
	g.processCommand("y")
	if g.stats.Turns != 2 {
		t.Fatal("a used shrine cannot be farmed")
	}
}

func TestShrineMessagesDescribeOnlyHealingRecipients(t *testing.T) {
	const playerHealing = "Your wounds ease in the shrine's gentle light."
	const dogHealing = "Ash's pale outline steadies in the shrine's light."
	const noHealing = "The shrine glows softly, but you feel no change."
	for _, tc := range []struct {
		name                    string
		playerHP, dogHP         int
		freed, alive            bool
		wantPlayerHP, wantDogHP int
		messages                []string
	}{
		{name: "healthy together", playerHP: 46, dogHP: 34, freed: true, alive: true, wantPlayerHP: 46, wantDogHP: 34, messages: []string{noHealing}},
		{name: "player only", playerHP: 40, dogHP: 34, freed: true, alive: true, wantPlayerHP: 46, wantDogHP: 34, messages: []string{playerHealing}},
		{name: "Ash only", playerHP: 46, dogHP: 24, freed: true, alive: true, wantPlayerHP: 46, wantDogHP: 34, messages: []string{dogHealing}},
		{name: "both", playerHP: 40, dogHP: 24, freed: true, alive: true, wantPlayerHP: 46, wantDogHP: 34, messages: []string{playerHealing, dogHealing}},
		{name: "Ash not freed", playerHP: 40, dogHP: 24, alive: true, wantPlayerHP: 46, wantDogHP: 24, messages: []string{playerHealing}},
		{name: "Ash fallen", playerHP: 40, freed: true, wantPlayerHP: 46, messages: []string{playerHealing}},
		{name: "healthy player and fallen Ash", playerHP: 46, freed: true, wantPlayerHP: 46, messages: []string{noHealing}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.current().Events = []floorEvent{{Kind: eventShrine, Pos: pos{6, 5}}}
			g.player.HP = tc.playerHP
			g.dog.HP, g.dog.Freed, g.dog.Alive, g.dog.Pos = tc.dogHP, tc.freed, tc.alive, pos{5, 4}
			rng := g.rng.State
			g.processCommand("y")
			if !reflect.DeepEqual(g.messages, tc.messages) {
				t.Fatalf("healing messages = %q, want %q", g.messages, tc.messages)
			}
			wantTurns := 0
			if tc.wantPlayerHP > tc.playerHP || tc.wantDogHP > tc.dogHP {
				wantTurns = 1
			}
			if g.player.HP != tc.wantPlayerHP || g.dog.HP != tc.wantDogHP || g.stats.Turns != wantTurns || g.current().Events[0].Used != (wantTurns == 1) || g.rng.State != rng {
				t.Fatal("recipient messages must preserve healing, turn costs, one-time use and RNG")
			}
		})
	}
}

func TestKeepsakeHealingThroughExecutableAndSaveLoad(t *testing.T) {
	g := openArena()
	g.current().Events = []floorEvent{{Kind: eventKeepsake, Pos: pos{6, 5}}}
	g.dog.Freed, g.dog.Pos, g.dog.HP = true, pos{5, 4}, 10
	wards := g.player.WardingCharms
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader("y\nsave\nload\ny\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executable: %v\n%s", err, output)
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.dog.HP != 18 || loaded.player.WardingCharms != wards+1 || !loaded.current().Events[0].Used || loaded.stats.Turns != 1 {
		t.Fatalf("keepsake should heal 8 HP and grant one ward once across save/load: dog HP=%d wards=%d used=%t turns=%d",
			loaded.dog.HP, loaded.player.WardingCharms, loaded.current().Events[0].Used, loaded.stats.Turns)
	}
	if loaded.rng.State != g.rng.State {
		t.Fatal("keepsake healing should preserve RNG through save/load")
	}
	if !strings.Contains(string(output), "Ghost dog: adopted (18/34 HP)") || !strings.Contains(string(output), "nothing nearby") {
		t.Fatalf("keepsake healing should appear in the status and remain spent after loading:\n%s", output)
	}
}

func TestLeverCreatesAndPreservesAShortcut(t *testing.T) {
	g := blinkArena()
	lvl := g.current()
	for y := range lvl.Tiles {
		for x := range lvl.Tiles[y] {
			lvl.Tiles[y][x] = '#'
		}
	}
	for x := 5; x <= 9; x++ {
		lvl.Tiles[5][x] = '.'
		lvl.Tiles[9][x] = '.'
	}
	for y := 6; y < 9; y++ {
		lvl.Tiles[y][9] = '.'
	}
	lvl.Tiles[7][5], lvl.Tiles[8][5] = '.', '.'
	g.player.Pos, lvl.Start = pos{5, 5}, pos{5, 5}
	g.dog.Freed = false
	cut := pos{5, 6}
	lvl.Events = []floorEvent{{Kind: eventLever, Pos: pos{6, 5}, Shortcut: &cut}}
	before := tileDistances(lvl.Tiles, pos{5, 5})[pos{5, 7}]
	g.processCommand("y")
	after := tileDistances(lvl.Tiles, pos{5, 5})[pos{5, 7}]
	if after != 2 || before <= after || g.stats.Turns != 1 || !lvl.Events[0].Used {
		t.Fatal("the lever should shorten an existing route and spend one turn")
	}
	g.saveDir = filepath.Dir(filepath.Join(t.TempDir(), "lever.json"))
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	loaded.processCommand("y")
	if !reflect.DeepEqual(lvl.Events, loaded.current().Events) || !reflect.DeepEqual(lvl.Tiles, loaded.current().Tiles) || loaded.stats.Turns != 1 {
		t.Fatal("loading should preserve the opened shortcut and prevent a second activation")
	}
}

func TestIndependentSecretRoomsCanBeDiscoveredInEitherOrder(t *testing.T) {
	g := openArena()
	lvl := g.current()
	first := &secretRoom{Door: pos{6, 5}, Tiles: []pos{{7, 5}, {7, 6}, {8, 5}, {8, 6}}, SideStoryID: 1}
	second := &secretRoom{Door: pos{15, 5}, Tiles: []pos{{16, 5}, {16, 6}, {17, 5}, {17, 6}}, SideStoryID: 2}
	lvl.Secrets = []*secretRoom{first, second}
	for p := range lvl.secretFootprint(false) {
		lvl.Tiles[p.Y][p.X] = '#'
	}
	g.addSecretSideStory(lvl)
	g.addSecretFountain(lvl)
	g.player.Pos = pos{14, 5}
	g.processCommand("search")
	if first.Revealed || !second.Revealed {
		t.Fatal("an undiscovered primary room must not block searching another room")
	}
	g.player.Pos = pos{5, 5}
	g.processCommand("search")
	if !first.Revealed || g.stats.Turns != 2 {
		t.Fatal("the first room should remain independently discoverable")
	}
	g.saveDir = filepath.Dir(filepath.Join(t.TempDir(), "rooms.json"))
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(lvl.secretRooms(), loaded.current().secretRooms()) || len(loaded.player.SideStories) != 0 {
		t.Fatal("loading must preserve room discovery without revealing uncollected book stories")
	}
}

func TestBlinkStoneExcludesAdditionalSecretRooms(t *testing.T) {
	g := blinkArena(pos{3, 5}, pos{10, 10}, pos{11, 10})
	foe := newMonster(monsterOrc)
	foe.Pos = pos{3, 5}
	g.current().Monsters = []*monster{&foe}
	extra := &secretRoom{Door: pos{29, 10}, Tiles: []pos{{30, 10}, {30, 11}}}
	g.current().Secrets = []*secretRoom{extra}
	revealRoom(g.current(), extra)
	if !g.useBlinkStone() || g.player.Pos != (pos{11, 10}) || g.dog.Pos != (pos{10, 10}) {
		t.Fatal("the farthest additional secret room must remain excluded even after discovery")
	}
}

func TestRunEventsAndMetadataSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	g := openArena()
	g.saveDir = filepath.Dir(filepath.Join(dir, "event.json"))
	g.current().Layout, g.current().Theme = layoutHall, themeChapel
	g.current().Events = []floorEvent{testFriendlyGhost(pos{6, 5})}
	g.current().Special = &specialRoom{Kind: roomVault, Area: room{X: 4, Y: 4, W: 3, H: 3}, Pos: pos{6, 6}}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--load-file", g.saveFile, "--save-file-path", dir, "--records-file", filepath.Join(dir, "records.json")}, strings.NewReader("y\nchoose 3\nsave\nload\ny\nsave\nquit\n"), &stdout, &stderr); err != nil {
		t.Fatalf("run events: %v %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ruined chapel — pillared halls") || !strings.Contains(stdout.String(), "offers one gift") || !strings.Contains(stdout.String(), "3. Ghost Dog recall scroll") || !strings.Contains(stdout.String(), "one Ghost Dog recall scroll") || !strings.Contains(stdout.String(), "nothing nearby") {
		t.Fatal("the command flow should show theme, choices, and the event's completion")
	}
	loaded, err := loadGame(lastSavePath(t, stdout.String()))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.player.GhostRecallScrolls != 1 || loaded.player.BlinkStones != 0 || loaded.stats.Turns != 1 || !loaded.current().Events[0].Used || loaded.current().Layout != layoutHall || loaded.current().Theme != themeChapel || !reflect.DeepEqual(loaded.current().Special, g.current().Special) {
		t.Fatal("save/load should retain the floor identity, room, used event, gift, and turn count")
	}
	if cmd, _, ok := rawKeyCommand('y'); !ok || cmd != "y" {
		t.Fatal("y should interact immediately in raw-key mode")
	}
}
