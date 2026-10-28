package main

import (
	"bufio"
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func packArena(t *testing.T) *game {
	t.Helper()
	g := openArena()
	g.saveDir = t.TempDir()
	g.player.ensureCarriedEquipment()
	g.player.carryWeapon(weapon{Name: "Trial spear", Min: 8, Max: 12, Reach: 2})
	g.player.HP = 20
	return g
}

func TestPackActionsThroughLoopAndSaveLoad(t *testing.T) {
	g := packArena(t)
	var out bytes.Buffer
	// Item numbers are shared with advanced commands; empty stock is omitted.
	input := "i\n2\ne\ni\n6\nu\ni\n1\nd\nsave\nload\nquit\n"
	if err := g.loop(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	if g.stats.Turns != 3 || g.player.Weapon.Name != "Trial spear" || g.player.Potions != 2 || g.player.HP <= 20 || len(g.current().Items) != 1 {
		t.Fatalf("menu actions must equip, heal, and drop through ordinary turns: %+v", g.player)
	}
	if !strings.Contains(out.String(), "Compared to Rusty Knife") || !strings.Contains(out.String(), "Enter/u: Use") {
		t.Fatal("selection must show comparisons and available actions")
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.player, g.player) || loaded.stats != g.stats || loaded.rng.State != g.rng.State {
		t.Fatal("menu actions must persist exactly")
	}
}

func TestPackReadingAndCancellationAreFree(t *testing.T) {
	g := packArena(t)
	player, stats, rng := g.player, g.stats, g.rng.State
	var out bytes.Buffer
	if err := g.loop(strings.NewReader("i\n2\nmap\ni\n999\nmap\nh\ndetails\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(player, g.player) || stats != g.stats || rng != g.rng.State {
		t.Fatal("reading, invalid selection and cancellation must be free")
	}
	if strings.Contains(out.String(), "Keys act immediately") || !strings.Contains(out.String(), "command and press Enter") {
		t.Fatal("line-input help must describe its own input mode")
	}
}

func TestRawMenuArrowsEnterAndEscape(t *testing.T) {
	g := packArena(t)
	menu := &inventoryMenu{}
	in := &rawInput{Reader: bufio.NewReader(strings.NewReader("\x1b[B\r\x1b\x03"))}
	var out bytes.Buffer
	for i := 0; i < 3; i++ {
		cmd, eof, err := readRawMenuCommand(in, &out)
		if err != nil || eof {
			t.Fatalf("menu input: %q %v %v", cmd, eof, err)
		}
		handled, close := g.packCommand(menu, cmd)
		if !handled || close != (i == 2) {
			t.Fatal("arrows select, Enter opens details and Esc cancels")
		}
	}
	if menu.selected != 1 || !menu.detail || g.stats.Turns != 0 {
		t.Fatal("raw menu navigation must not act")
	}
}

func TestPackActionsThroughExecutable(t *testing.T) {
	if testing.Short() {
		t.Skip("compiled menu replay")
	}
	g := packArena(t)
	g.player.Weapons[1].Magic = true
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader("i\n2\ne\ni\n6\nu\ni\n1\nd\nsave\nload\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("menu executable: %v\n%s", err, output)
	}
	prose := strings.Join(strings.Fields(stripANSI(string(output))), " ")
	if !strings.Contains(prose, "You drop Rusty Knife on the floor.") || strings.Contains(prose, "Dropping takes a turn.") {
		t.Fatal("the executable must confirm the drop without a turn-cost reminder")
	}
	for _, want := range []string{
		"2. Trial spear (8-12, reach 2, +2 vs undead)",
		"Weapon Trial spear (8-12) | Reach 2 | +2 vs undead",
	} {
		if !strings.Contains(prose, want) {
			t.Fatalf("the executable must show weapon bonuses before and after equipping: missing %q", want)
		}
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	g.processCommand("equip 2")
	g.processCommand("p")
	g.processCommand("drop 1")
	if !reflect.DeepEqual(loaded.player, g.player) || !reflect.DeepEqual(loaded.levels, g.levels) || loaded.stats != g.stats || loaded.rng.State != g.rng.State {
		t.Fatal("compiled menu actions must match direct command semantics")
	}
}

func TestCompactMapAtSupportedWidths(t *testing.T) {
	for _, width := range []string{"40", "60", "80"} {
		t.Run(width, func(t *testing.T) {
			t.Setenv("COLUMNS", width)
			g, m := warnedEpicArena(monsterStormLeviathan)
			g.messages = nil
			g.player.ShieldTurns, g.player.HexedTurns, g.player.PoisonTurns = 2, 1, 3
			g.prepareEpicRetaliation(m)
			g.messages = nil // Width checks cover map chrome; the complete log is retained separately.
			g.player.Weapon.Name = "A rather long named weapon of twilight"
			g.player.Weapon.Magic, g.player.Weapon.Power = true, powerChainLightning
			g.player.HeadArmor.SpellWard, g.player.HeadArmor.StrikeBonus = 4, 3
			var out bytes.Buffer
			g.render(&out)
			for _, line := range strings.Split(stripANSI(out.String()), "\n") {
				if len([]rune(line)) > uiWidth() {
					t.Fatalf("line exceeds width %s: %s", width, line)
				}
			}
			if strings.Contains(out.String(), "Recall 0") || strings.Contains(out.String(), "Starfire 0") {
				t.Fatal("map must not list absent consumables")
			}
		})
	}
}

func TestWeaponBonusesOnMapAndInventoryList(t *testing.T) {
	weapons := []struct {
		weapon weapon
		bonus  string
	}{
		{weapon{Name: "Trial blade", Min: 4, Max: 6}, ""},
		{weapon{Name: "Blessed blade", Min: 4, Max: 6, Magic: true}, "+2 vs undead"},
		{artifactWeapons[0], "+2 vs undead, Life drain 20% (max 8 HP/hit)"},
		{artifactWeapons[1], "+2 vs undead, Freeze 25%"},
		{artifactWeapons[2], "+2 vs undead, Ignores armor"},
		{artifactWeapons[3], "+2 vs undead, Chain lightning 12 damage"},
	}
	for _, equipped := range weapons {
		for _, width := range []string{"40", "60", "80"} {
			t.Run(equipped.weapon.Name+"/"+width, func(t *testing.T) {
				t.Setenv("COLUMNS", width)
				g := openArena()
				g.player.Weapon = equipped.weapon
				g.player.Weapons = nil
				for _, carried := range weapons {
					g.player.Weapons = append(g.player.Weapons, carried.weapon)
				}
				player, dog, stats, rng := g.player, g.dog, g.stats, g.rng.State
				player.Weapons = append([]weapon(nil), g.player.Weapons...)
				var out bytes.Buffer
				g.render(&out)
				prose := strings.Join(strings.Fields(stripANSI(out.String())), " ")
				want := fmt.Sprintf("%s (%d-%d) | Reach %d", equipped.weapon.Name, equipped.weapon.Min, equipped.weapon.Max, equipped.weapon.effectiveReach())
				if equipped.bonus != "" {
					want += " | " + equipped.bonus
				}
				_, header, found := strings.Cut(prose, "Weapon ")
				header, _, ended := strings.Cut(header, " Ghost dog:")
				if !found || !ended || header != want {
					t.Fatalf("weapon header = %q, want only its active bonuses: %q", header, want)
				}
				for _, raw := range []bool{false, true} {
					var pack bytes.Buffer
					g.renderPack(&pack, &inventoryMenu{}, raw)
					text := strings.Join(strings.Fields(stripANSI(pack.String())), " ")
					for i, carried := range weapons {
						bonus := ""
						if carried.bonus != "" {
							bonus = ", " + carried.bonus
						}
						row := fmt.Sprintf("%d. %s (%d-%d, reach %d%s)", i+1, stripANSI(coloredWeaponName(carried.weapon)), carried.weapon.Min, carried.weapon.Max, carried.weapon.effectiveReach(), bonus)
						if carried.weapon == equipped.weapon {
							row += " [equipped]"
						}
						if !strings.Contains(text, row) {
							t.Errorf("raw=%v: inventory must show each weapon's own bonuses without details: missing %q", raw, row)
						}
					}
					out.Write(pack.Bytes())
				}
				for _, line := range strings.Split(stripANSI(out.String()), "\n") {
					if len([]rune(line)) > uiWidth() {
						t.Fatalf("weapon bonuses exceed width %s: %s", width, line)
					}
				}
				if !reflect.DeepEqual(player, g.player) || dog != g.dog || stats != g.stats || rng != g.rng.State {
					t.Fatal("displaying weapon bonuses must not change equipment, health, turns or RNG")
				}
			})
		}
	}
}

func TestMapHeaderShowsOnlyActiveEquipmentBonuses(t *testing.T) {
	for _, tc := range []struct {
		name         string
		ward, strike int
	}{
		{"none", 0, 0}, {"ward only", 4, 0}, {"strike only", 0, 3}, {"both", 4, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.messages = nil
			g.player.BodyArmor.SpellWard, g.player.BodyArmor.StrikeBonus = tc.ward, tc.strike
			player, stats, rng := g.player, g.stats, g.rng.State
			var out bytes.Buffer
			g.render(&out)
			text := strings.Join(strings.Fields(out.String()), " ")
			if strings.Contains(text, "Spell ward 0") || strings.Contains(text, "Strike bonus +0") {
				t.Fatal("inactive equipment bonuses must be omitted")
			}
			if strings.Contains(text, "Spell ward 4") != (tc.ward != 0) || strings.Contains(text, "Strike bonus +3") != (tc.strike != 0) {
				t.Fatalf("active bonuses must both be shown: %s", text)
			}
			if !reflect.DeepEqual(player, g.player) || stats != g.stats || rng != g.rng.State {
				t.Fatal("rendering bonuses must not mutate gameplay")
			}
		})
	}
}

func TestInventoryListShowsEachArmorAttributes(t *testing.T) {
	g := packArena(t)
	g.player.Armors = []armor{
		{Name: "Trial Hood", Slot: slotHead, Defense: 2},
		{Name: "Trial Coat", Slot: slotBody, Defense: 4, SpellWard: 3, StrikeBonus: 1},
		{Name: "Trial Boots", Slot: slotLeg, Defense: 1, SpellWard: 2, StrikeBonus: 5},
	}
	g.player.HeadArmor = g.player.Armors[0]
	var out bytes.Buffer
	if err := g.loop(strings.NewReader("i\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	text := strings.Join(strings.Fields(stripANSI(out.String())), " ")
	for _, want := range []string{
		"Trial Hood (head): 2 guard, 0 spell ward, +0 strike damage [equipped]",
		"Trial Coat (body): 4 guard, 3 spell ward, +1 strike damage",
		"Trial Boots (leg): 1 guard, 2 spell ward, +5 strike damage",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("inventory list missing %q: %s", want, text)
		}
	}
}

func TestRawInventoryHighlightedActions(t *testing.T) {
	for _, action := range []string{"e", "d"} {
		t.Run(action, func(t *testing.T) {
			g := packArena(t)
			menu := &inventoryMenu{}
			in := &rawInput{Reader: bufio.NewReader(strings.NewReader("\x1b[B" + action))}
			var out bytes.Buffer
			for step := 0; step < 2; step++ {
				cmd, eof, err := readRawMenuCommand(in, &out)
				if err != nil || eof {
					t.Fatalf("raw input: %q %v %v", cmd, eof, err)
				}
				handled, close := g.packCommand(menu, cmd)
				if !handled || close != (step == 1) || menu.detail {
					t.Fatal("highlighted action should work without opening details")
				}
			}
			if g.stats.Turns != 1 {
				t.Fatalf("action must spend one turn, got %d", g.stats.Turns)
			}
			if action == "e" && g.player.Weapon.Name != "Trial spear" {
				t.Fatal("highlighted weapon was not equipped")
			}
			if action == "d" && (len(g.player.Weapons) != 1 || len(g.current().Items) != 1 || g.current().Items[0].Weapon.Name != "Trial spear") {
				t.Fatal("highlighted weapon was not dropped")
			}
		})
	}
}

func TestInventoryListActionsThroughLoopAndSaveLoad(t *testing.T) {
	g := packArena(t)
	var out bytes.Buffer
	if err := g.loop(strings.NewReader("i\ns\ne\ni\nd\nsave\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.stats.Turns != 2 || loaded.player.Weapon.Name != "Trial spear" || len(loaded.player.Weapons) != 1 || len(loaded.current().Items) != 1 || loaded.current().Items[0].Weapon.Name != "Rusty Knife" {
		t.Fatal("direct list actions must equip and drop through the loop and persist")
	}
}

func TestInventoryListShowsConsumableUseKeys(t *testing.T) {
	g := packArena(t)
	commands := []string{"p", "f", "g", "b", "u", "t", "o", "n", "r"}
	for _, line := range g.inventoryLines() {
		*line.quantity = 1
	}
	for _, raw := range []bool{false, true} {
		var out bytes.Buffer
		g.renderPack(&out, &inventoryMenu{}, raw)
		text := strings.Join(strings.Fields(stripANSI(out.String())), " ")
		for i, line := range g.inventoryLines() {
			want := line.name + " x1 (map " + string(itemGlyphs[line.kind]) + ") [use " + commands[i] + "]"
			if !strings.Contains(text, want) {
				t.Errorf("raw=%v: inventory missing %q", raw, want)
			}
		}
	}
}
