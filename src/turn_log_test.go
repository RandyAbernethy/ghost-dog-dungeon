package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func crowdedLogArena() *game {
	g := openArena()
	g.rng = newSimpleRNG(1)
	g.stats.Turns = 80
	g.stats.MonstersKilled = 37
	g.messages = []string{"An event from the previous turn."}
	g.player.HP, g.player.MaxHP = 1000, 1000
	g.player.Weapon = weapon{Name: "Silver test blade", Min: 3, Max: 3, Magic: true}
	g.dog.Freed, g.dog.Pos = true, pos{5, 4}
	g.dog.HP, g.dog.MaxHP = 1000, 1000
	foes := []struct {
		kind monsterKind
		spot pos
	}{
		{monsterSkeleton, pos{6, 5}},
		{monsterOrc, pos{4, 4}}, {monsterOrc, pos{4, 5}}, {monsterOrc, pos{4, 6}},
		{monsterOrc, pos{5, 6}}, {monsterOrc, pos{6, 6}}, {monsterOrc, pos{6, 4}},
		{monsterVoidSeer, pos{8, 3}}, {monsterVoidSeer, pos{9, 4}}, {monsterVoidSeer, pos{10, 5}},
		{monsterGloomArcher, pos{8, 7}}, {monsterGloomArcher, pos{9, 8}}, {monsterGloomArcher, pos{10, 9}},
	}
	for i, foe := range foes {
		m := newMonster(foe.kind)
		m.Name, m.Pos = fmt.Sprintf("%s %d", m.Name, i+1), foe.spot
		m.HP, m.MaxHP, m.MinDamage, m.MaxDamage = 200, 200, 2, 2
		g.current().Monsters = append(g.current().Monsters, &m)
	}
	return g
}

func assertWholeTurnRendered(t *testing.T, g *game, output string) {
	t.Helper()
	header := fmt.Sprintf("Turn %d events (%d monsters killed):\n", g.stats.Turns, g.stats.MonstersKilled)
	_, remaining, ok := strings.Cut(output, header)
	if !ok {
		t.Fatalf("missing current turn heading %q", header)
	}
	remaining = strings.Join(strings.Fields(stripANSI(remaining)), " ")
	for _, message := range g.messages {
		_, rest, found := strings.Cut(remaining, strings.Join(strings.Fields(stripANSI("- "+message)), " "))
		if !found {
			t.Fatalf("missing or reordered event %q", message)
		}
		remaining = rest
	}
	if strings.Contains(output, "An event from the previous turn.") {
		t.Fatal("current turn must not display an earlier turn")
	}
}

func TestCrowdedCombatLogKeepsPlayerDogRangedAndMeleeEvents(t *testing.T) {
	g := crowdedLogArena()
	g.processCommand("d")
	if g.stats.Turns != 81 || len(g.messages) <= 8 || !strings.HasPrefix(g.messages[0], "You strike Skeleton 1") {
		t.Fatalf("combat should start a complete new turn with your strike: %v", g.messages)
	}
	joined := strings.Join(g.messages, "\n")
	for _, event := range []string{"flares against the undead", "Ghost dog tears into", "hits you", "tears at your thoughts", "fires a barbed arrow"} {
		if !strings.Contains(joined, event) {
			t.Fatalf("crowded turn lost %q: %s", event, joined)
		}
	}
	beforePlayer, beforeDog, beforeStats, beforeRNG := g.player, g.dog, g.stats, g.rng.State
	beforeLog := append([]string(nil), g.messages...)
	var out bytes.Buffer
	g.render(&out)
	assertWholeTurnRendered(t, g, out.String())
	if !reflect.DeepEqual(beforePlayer, g.player) || beforeDog != g.dog || beforeStats != g.stats || beforeRNG != g.rng.State || !reflect.DeepEqual(beforeLog, g.messages) {
		t.Fatal("rendering a complete turn must not change gameplay or its log")
	}
}

func TestEachSuccessfulActionReplacesThePreviousTurnLog(t *testing.T) {
	g := openArena()
	g.messages = []string{"An event from the previous turn."}
	g.processCommand("d")
	if !reflect.DeepEqual(g.messages, []string{"You move east."}) || g.stats.Turns != 1 {
		t.Fatal("quiet movement should begin its own turn")
	}
	potion := item{Pos: pos{6, 4}, Kind: itemPotion, Name: "healing potion"}
	g.current().Items = []item{potion}
	g.processCommand("w")
	if len(g.messages) != 2 || g.messages[0] != "You move north." || !strings.Contains(g.messages[1], "stash a healing potion") || g.stats.Turns != 2 {
		t.Fatalf("movement should precede its pickup: %v", g.messages)
	}
	g.processCommand(".")
	if !reflect.DeepEqual(g.messages, []string{"You wait and let the dungeon move around you."}) || g.stats.Turns != 3 {
		t.Fatal("a quiet wait must not retain the last move or pickup")
	}
	g.player.BoundTurns, g.player.Binding = 1, bindingWebs
	beforePos := g.player.Pos
	g.processCommand("d")
	if !reflect.DeepEqual(g.messages, []string{"Sticky webs hold you in place for a turn."}) || g.stats.Turns != 4 || g.player.Pos != beforePos {
		t.Fatal("a turn spent entangled should replace the previous log too")
	}
}

func TestFailedActionsReplaceActivityWithoutSpendingTurns(t *testing.T) {
	g := openArena()
	g.processCommand(".")
	current := append([]string(nil), g.messages...)
	g.player.Potions, g.player.FireScrolls = 0, 0
	g.current().Tiles[5][4] = '#'
	g.current().Events = []floorEvent{testFriendlyGhost(pos{6, 5})}
	beforePlayer, beforeDog, beforeStats, beforeRNG := g.player, g.dog, g.stats, g.rng.State
	for _, tc := range []struct{ command, message string }{
		{"a", "Stone blocks your path."},
		{"f", "You have no fire scrolls."},
		{"p", "You have no healing potions."},
		{"equip 999", "Choose an equipment number shown in your inventory."},
		{"drop 1", "You cannot drop an equipped item. Equip something else first."},
		{"unknown", "Unknown command. Press h for help."},
		{"f", "You have no fire scrolls."},
	} {
		g.processCommand(tc.command)
		if !reflect.DeepEqual(g.messages, []string{tc.message}) || !reflect.DeepEqual(g.player, beforePlayer) || g.dog != beforeDog || g.stats != beforeStats || g.rng.State != beforeRNG {
			t.Fatalf("failed command %q must show only its feedback and leave gameplay unchanged: %q", tc.command, g.messages)
		}
		var screen bytes.Buffer
		g.render(&screen)
		if strings.Contains(screen.String(), current[0]) {
			t.Fatal("a failed command must not replay the previous action")
		}
	}
	g.processCommand("y")
	if len(g.messages) == 0 || !strings.HasPrefix(g.messages[0], "The ghost of a friendly villager offers one gift.") || !reflect.DeepEqual(g.player, beforePlayer) || g.stats != beforeStats || g.rng.State != beforeRNG {
		t.Fatal("reading gift choices must show only the offer without advancing gameplay")
	}
	beforeLog := append([]string(nil), g.messages...)
	var out bytes.Buffer
	if err := g.loop(strings.NewReader("h\ni\nc\nm\nbook\nequipment\nchallenges\n\x1b\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeLog, g.messages) || !reflect.DeepEqual(g.player, beforePlayer) || g.dog != beforeDog || g.stats != beforeStats || g.rng.State != beforeRNG {
		t.Fatal("reading screens and Esc should preserve the latest activity")
	}
	g.processCommand(".")
	if len(g.messages) != 1 || g.messages[0] != current[0] || g.stats.Turns != beforeStats.Turns+1 {
		t.Fatal("the next real turn must replace the free-command feedback")
	}
}

func TestSaveLoadFailuresReplaceActivityWithoutSpendingTurns(t *testing.T) {
	for _, tc := range []struct{ name, command, prefix string }{
		{"save", "save", "Save failed:"},
		{"load", "load", "Load failed:"},
		{"load without snapshot", "load", "No save file is selected;"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.processCommand(".")
			missing := filepath.Join(t.TempDir(), "missing")
			switch tc.name {
			case "save":
				g.saveDir = missing
			case "load":
				g.saveFile = missing
			}
			player, dog, stats, rng := g.player, g.dog, g.stats, g.rng.State
			g.processCommand(tc.command)
			if len(g.messages) != 1 || !strings.HasPrefix(g.messages[0], tc.prefix) {
				t.Fatalf("failed %s must show only its own response: %q", tc.name, g.messages)
			}
			if !reflect.DeepEqual(g.player, player) || g.dog != dog || g.stats != stats || g.rng.State != rng {
				t.Fatal("a failed save/load command must not advance gameplay")
			}
		})
	}
}

func TestCurrentTurnLogSurvivesSaveLoadWithoutMergingOtherTurns(t *testing.T) {
	g := crowdedLogArena()
	g.saveDir = filepath.Dir(filepath.Join(t.TempDir(), "turn.json"))
	g.processCommand("d")
	want := append([]string(nil), g.messages...)
	beforeStats, beforeRNG := g.stats, g.rng.State
	g.processCommand("save")
	if !reflect.DeepEqual(want, g.messages[:len(want)]) || g.stats != beforeStats || g.rng.State != beforeRNG {
		t.Fatal("saving should retain the combat log and spend no turn")
	}
	g.processCommand(".")
	g.processCommand("load")
	if len(g.messages) != len(want)+1 || !reflect.DeepEqual(want, g.messages[:len(want)]) || !strings.HasPrefix(g.messages[len(want)], "Game loaded from ") || g.stats != beforeStats || g.rng.State != beforeRNG {
		t.Fatal("loading must restore the saved turn, without merging another turn's events")
	}
	var out bytes.Buffer
	g.render(&out)
	assertWholeTurnRendered(t, g, out.String())
}

func TestPlayerActionPrecedesMagicFountainAndFloorTransition(t *testing.T) {
	t.Run("hexed magic strike", func(t *testing.T) {
		g := crowdedLogArena()
		g.player.HexedTurns = 1
		g.processCommand("d")
		if !strings.HasPrefix(g.messages[0], "You strike ") || !strings.Contains(strings.Join(g.messages, "\n"), "The hex weakens your strike.") {
			t.Fatal("the attack must precede its magic and curse details")
		}
	})
	t.Run("sun orb", func(t *testing.T) {
		g := crowdedLogArena()
		g.player.SunOrbs = 1
		g.processCommand("u")
		if g.messages[0] != "You unleash the sun orb's fire." {
			t.Fatal("using the orb should precede its victims' reactions")
		}
	})
	t.Run("fountain", func(t *testing.T) {
		g := openArena()
		g.dog.Freed, g.dog.Pos, g.dog.HP = true, pos{5, 4}, 10
		g.player.HP = 10
		g.current().Fountains = []fountain{{Pos: g.player.Pos}}
		g.processCommand("v")
		if !strings.HasPrefix(g.messages[0], "You drink from the fountain") || !strings.Contains(g.messages[1], "restores") {
			t.Fatal("your drink should precede Ghost Dog's healing")
		}
	})
	t.Run("stairs", func(t *testing.T) {
		g := openArena()
		g.current().Stairs, g.current().HasStair = g.player.Pos, true
		g.levels[1].Monsters = nil
		g.dog.Freed, g.dog.HP = true, 10
		g.processCommand(">")
		if g.messages[0] != "You descend to level 2." || !strings.Contains(g.messages[1], "recovers") {
			t.Fatal("descending should precede companion recovery")
		}
	})
}

func TestCompleteTurnLogThroughRealExecutableAndSavedGame(t *testing.T) {
	dir := t.TempDir()
	g := crowdedLogArena()
	g.saveDir = filepath.Dir(filepath.Join(dir, "combat.json"))
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", dir)
	cmd.Env = testEnvironment(dir)
	cmd.Stdin = strings.NewReader("d\ni\n\x1b\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, output)
	}
	g.processCommand("d")
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.messages, loaded.messages) || loaded.stats.Turns != 81 || len(loaded.messages) <= 8 {
		t.Fatal("the executable must preserve the entire crowded turn through reading screens and saving")
	}
	for _, screen := range strings.Split(string(output), "=== Ghost Dog Dungeon:")[1:] {
		if strings.Contains(screen, "Turn 81 events (") {
			assertWholeTurnRendered(t, loaded, screen)
		}
	}
}

func TestFailedCommandAfterCombatThroughExecutableAndSaveLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("compiled failed-command replay")
	}
	g := crowdedLogArena()
	g.saveDir = t.TempDir()
	g.processCommand("d")
	g.player.FireScrolls = 0
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader("f\nsave\nload\nf\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed-command replay: %v\n%s", err, output)
	}
	screens := strings.Split(string(output), "=== Ghost Dog Dungeon:")
	if len(screens) < 3 {
		t.Fatal("the executable must display combat, then the failed command")
	}
	assertWholeTurnRendered(t, g, screens[1])
	for _, screen := range screens[2:] {
		if !strings.Contains(screen, "You have no fire scrolls.") || strings.Contains(screen, "You strike ") || strings.Contains(screen, "Ghost dog tears into ") || strings.Contains(screen, "fires a barbed arrow") {
			t.Fatal("failure screens must show the failed command without replaying combat")
		}
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.messages, []string{"You have no fire scrolls."}) {
		t.Fatalf("save/load must preserve the latest failure response: %q", loaded.messages)
	}
	if !reflect.DeepEqual(loaded.player, g.player) || loaded.dog != g.dog || !reflect.DeepEqual(loaded.levels, g.levels) || loaded.stats != g.stats || loaded.rng.State != g.rng.State {
		t.Fatal("failed commands and save/load must not advance combat, health, turns or RNG")
	}
}

func TestEquipmentPickupLogWrapsAt80VisibleColumns(t *testing.T) {
	g := openArena()
	g.current().Items = []item{{Pos: pos{6, 5}, Kind: itemArmor, Armor: armor{Name: "Scout Cap", Slot: slotHead, Defense: 2, Rarity: rarityUncommon}}}
	g.processCommand("d")
	messages := append([]string(nil), g.messages...)
	player, stats, rng := g.player, g.stats, g.rng.State
	var colored bytes.Buffer
	g.render(&colored)
	assertWholeTurnRendered(t, g, colored.String())
	_, log, ok := strings.Cut(stripANSI(colored.String()), "Turn 1 events (0 monsters killed):\n")
	if !ok {
		t.Fatal("missing pickup turn log")
	}
	if !strings.Contains(log, "You pack Uncommon Scout Cap / head.\n  Compared to Padded Hood:") {
		t.Fatalf("comparison must start on a new line: %s", log)
	}
	continuations := 0
	for _, line := range strings.Split(strings.TrimSuffix(log, "\n"), "\n") {
		if len(line) > 80 {
			t.Fatalf("log exceeds 80 visible columns: %s", line)
		}
		if strings.HasPrefix(line, "  ") {
			continuations++
		} else if !strings.HasPrefix(line, "- ") {
			t.Fatalf("invalid log indentation: %s", line)
		}
	}
	if continuations < 1 || !strings.Contains(colored.String(), "\x1b[") {
		t.Fatal("exercise colored equipment prose spanning several lines")
	}
	if !reflect.DeepEqual(messages, g.messages) || !reflect.DeepEqual(player, g.player) || stats != g.stats || rng != g.rng.State {
		t.Fatal("word wrapping must not change gameplay or stored messages")
	}
	for i := range g.messages {
		g.messages[i] = stripANSI(g.messages[i])
	}
	var plain bytes.Buffer
	g.render(&plain)
	if stripANSI(colored.String()) != plain.String() {
		t.Fatal("color codes must not change line breaks")
	}
}

func TestArmorPickupMessageIsConcise(t *testing.T) {
	g := newGame(5)
	g.player.HeadArmor = armor{Name: "Padded Hood", Slot: slotHead, Defense: 1}
	g.messages = nil
	g.collectItem(makeArmorItem(armor{Name: "Riveted Sallet", Slot: slotHead, Defense: 1, SpellWard: 1, Rarity: rarityUncommon}))
	want := "You pack Uncommon Riveted Sallet / head. Compared to Padded Hood: +0 guard, +1 spell ward, +0 strike damage."
	if len(g.messages) != 1 || stripANSI(g.messages[0]) != want {
		t.Fatalf("want concise pickup %q, got %v", want, g.messages)
	}
	var out bytes.Buffer
	g.render(&out)
	wantRendered := "- " + strings.Replace(want, ". Compared to ", ".\n  Compared to ", 1) + "\n"
	if !strings.Contains(stripANSI(out.String()), wantRendered) {
		t.Fatalf("pickup comparison must start on its own indented line: %s", stripANSI(out.String()))
	}
	if stripANSI(g.messages[0]) != want {
		t.Fatal("rendering must preserve the stored pickup message")
	}
}
