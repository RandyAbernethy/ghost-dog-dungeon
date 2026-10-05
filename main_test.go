package main

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func TestNewGameHasTenLevelsBossDogAndVariety(t *testing.T) {
	g := newGame(7)
	if len(g.levels) != 10 {
		t.Fatalf("len(levels) = %d, want 10", len(g.levels))
	}
	if g.levels[1].DogChain == nil {
		t.Fatal("expected chained ghost dog on level 2")
	}
	if g.player.HP != 46 || g.player.MaxHP != 46 || g.player.Potions != 3 {
		t.Fatalf("expected gentler starting stats, got hp=%d/%d potions=%d", g.player.HP, g.player.MaxHP, g.player.Potions)
	}
	if g.player.HeadArmor.Name == "" || g.player.BodyArmor.Name == "" || g.player.Weapon.Name == "" {
		t.Fatal("expected starting equipment to be present")
	}
	foundBoss := false
	kinds := map[monsterKind]bool{}
	for i, lvl := range g.levels {
		if lvl.Tiles[lvl.Start.Y][lvl.Start.X] != '.' {
			t.Fatalf("level %d start is not on a floor tile", i+1)
		}
		if i > 0 && (!lvl.HasUpStair || lvl.UpStairs != lvl.Start) {
			t.Fatalf("level %d should have an up stair at its start", i+1)
		}
		if i < levelCount-1 && !lvl.HasStair {
			t.Fatalf("level %d missing stairs down", i+1)
		}
		for _, m := range lvl.Monsters {
			kinds[m.Kind] = true
			if m.Boss {
				foundBoss = true
				if i != levelCount-1 {
					t.Fatalf("boss spawned on level %d, want level %d", i+1, levelCount)
				}
				if distance(m.Pos, lvl.Start) < 10 {
					t.Fatalf("boss too close to bottom-floor entry stairs: distance=%d", distance(m.Pos, lvl.Start))
				}
			}
		}
	}
	if !foundBoss {
		t.Fatal("expected boss on final level")
	}
	bottom := g.levels[levelCount-1]
	if bottom.Secret == nil || !bottom.HasEscapeStair {
		t.Fatal("expected final level to contain a hidden escape sanctum")
	}
	if len(kinds) < 10 {
		t.Fatalf("found %d monster kinds, want at least 10", len(kinds))
	}
}

func TestDungeonGenerationCreatesRoomsAndReachableStairs(t *testing.T) {
	g := newGame(11)
	for i, lvl := range g.levels[:levelCount-1] {
		floors, walls := 0, 0
		for y := 0; y < mapHeight; y++ {
			for x := 0; x < mapWidth; x++ {
				if lvl.Tiles[y][x] == '.' {
					floors++
				} else {
					walls++
				}
			}
		}
		if floors != floorBudget(i) || walls < 120 {
			t.Fatalf("level %d should match its planned size and retain walls, got floors=%d walls=%d", i+1, floors, walls)
		}
		if !reachable(lvl.Tiles, lvl.Start, lvl.Stairs) {
			t.Fatalf("level %d stairs are not reachable from the start", i+1)
		}
	}
}

func TestMonsterCountsScaleWithWalkableFloorArea(t *testing.T) {
	g := newGame(21)
	previousCount := 0
	for depth, lvl := range g.levels {
		want := max(1, countFloorTiles(lvl.Tiles)/30)
		got := len(lvl.Monsters)
		if got != want {
			t.Fatalf("level %d monsters = %d, want %d from %d walkable tiles", depth+1, got, want, countFloorTiles(lvl.Tiles))
		}
		if depth > 0 && got <= previousCount {
			t.Fatalf("level %d should have more monsters than level %d: got %d and %d", depth+1, depth, got, previousCount)
		}
		previousCount = got
	}
	if got := len(g.levels[0].Monsters); got != 1 {
		t.Fatalf("level 1 monsters = %d, want 1", got)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	savePath := filepath.Join(t.TempDir(), "savegame.json")
	g := newGameWithSaveFile(13, savePath)
	g.currentLevel = 2
	g.player.Pos = g.levels[2].Start
	g.player.Potions = 5
	g.player.HP = 31
	g.player.BodyArmor = armor{Name: "Knight Mail", Slot: slotBody, Defense: 3, Rarity: rarityEpic}
	g.player.FrostCharms = 2
	g.knownMonsters[monsterSpider] = true
	g.dog.Freed = true
	g.dog.Pos = pos{X: g.player.Pos.X + 1, Y: g.player.Pos.Y}
	g.messages = []string{"one", "two"}
	g.levels[2].Monsters[0].HP = 3
	g.rng.State = 99

	if err := g.save(); err != nil {
		t.Fatalf("save() error = %v", err)
	}
	loaded, err := loadGame(savePath)
	if err != nil {
		t.Fatalf("loadGame() error = %v", err)
	}
	if loaded.currentLevel != 2 || loaded.player.Potions != 5 || loaded.player.HP != 31 || loaded.player.FrostCharms != 2 {
		t.Fatalf("loaded player state = %+v", loaded.player)
	}
	if loaded.player.BodyArmor.Name != "Knight Mail" || loaded.player.BodyArmor.Rarity != rarityEpic {
		t.Fatalf("loaded armor = %+v", loaded.player.BodyArmor)
	}
	if !loaded.knownMonsters[monsterSpider] {
		t.Fatal("expected known monsters to survive save/load")
	}
}

func TestCompareTooltipAndCarryWeaponAndArmor(t *testing.T) {
	g := newGame(5)
	g.player.Pos = g.current().Start
	g.current().Items = []item{{Pos: g.player.Pos, Kind: itemWeapon, Weapon: weapon{Name: "Moonblade", Min: 7, Max: 11}}, {Pos: g.player.Pos, Kind: itemArmor, Armor: armor{Name: "Knight Mail", Slot: slotBody, Defense: 3, Rarity: rarityEpic}}}
	g.collectItems()
	joined := stripANSI(strings.Join(g.messages, "\n"))
	if !strings.Contains(joined, "Compare to Rusty Knife") || !strings.Contains(joined, "Compare to Worn Coat") {
		t.Fatalf("missing compare tooltips in %q", joined)
	}
	if !strings.Contains(joined, "Epic Knight Mail") {
		t.Fatalf("missing colored/rarity armor name in %q", joined)
	}
	if g.player.Weapon.Name != "Rusty Knife" || g.player.BodyArmor.Name != "Worn Coat" {
		t.Fatal("finding equipment should preserve the player's chosen loadout")
	}
	if len(g.player.Weapons) != 2 || len(g.player.Armors) != 4 {
		t.Fatalf("found gear was not retained: weapons=%d armors=%d", len(g.player.Weapons), len(g.player.Armors))
	}
}

func TestMagicalWeaponDealsExtraDamageToUndead(t *testing.T) {
	g := newGame(5)
	g.player.Weapon = weapon{Name: "Emberbrand", Min: 5, Max: 5, Magic: true}
	target := newMonster(monsterSkeleton)
	g.attackMonster(&target)
	if target.HP != target.MaxHP-7 {
		t.Fatalf("magical weapon damage = %d, want %d", target.MaxHP-target.HP, 7)
	}
	if !strings.Contains(strings.Join(g.messages, "\n"), "flares against the undead") {
		t.Fatalf("missing magical weapon feedback: %q", strings.Join(g.messages, " | "))
	}
}

func TestInventoryScreenShowsEquipmentDescriptionsAndColors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"--seed", "3"}, strings.NewReader("i\nquit\n"), &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI color codes in inventory output: %q", out)
	}
	plain := stripANSI(out)
	for _, want := range []string{"=== Inventory & Equipment ===", "Head: Common Padded Hood (+1 guard, common)", "Body: Common Worn Coat (+1 guard, common)", "Fire Scroll x1", "Frost Charm x0"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("inventory output missing %q in %q", want, plain)
		}
	}
}

func TestCodexScreenShowsKnownEnemies(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"--seed", "3"}, strings.NewReader("c\nquit\n"), &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	plain := stripANSI(stdout.String())
	for _, want := range []string{"=== Enemy Codex ===", newGame(3).current().Monsters[0].Name, "Trick:"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("codex output missing %q in %q", want, plain)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestInspectScreenShowsThreatsLootAndStairs(t *testing.T) {
	g := newGame(3)
	lvl := g.current()
	for y := range lvl.Tiles {
		for x := range lvl.Tiles[y] {
			lvl.Tiles[y][x] = '#'
		}
	}
	for _, p := range []pos{{5, 5}, {6, 5}, {7, 5}, {5, 6}, {6, 6}, {7, 6}} {
		lvl.Tiles[p.Y][p.X] = '.'
	}
	g.player.Pos = pos{X: 5, Y: 5}
	lvl.HasStair = true
	lvl.Stairs = pos{X: 7, Y: 5}
	lvl.Items = []item{{Pos: pos{X: 6, Y: 5}, Kind: itemWeapon, Weapon: weapon{Name: "Short Sword", Min: 4, Max: 7}}, {Pos: pos{X: 6, Y: 6}, Kind: itemArmor, Armor: armor{Name: "Leather Jerkin", Slot: slotBody, Defense: 2, Rarity: rarityUncommon}}}
	lvl.Monsters = []*monster{{Kind: monsterRat, Name: "Dungeon Rat", Glyph: 'r', Pos: pos{X: 7, Y: 6}, HP: 8, MaxHP: 8, MinDamage: 2, MaxDamage: 4}}
	var out bytes.Buffer
	g.renderInspect(&out)
	plain := stripANSI(out.String())
	for _, want := range []string{"=== Inspect ===", "Nearby threats:", "Nearby loot:", "weapon Short Sword", "armor for the body slot", "Stairs down:"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("inspect output missing %q in %q", want, plain)
		}
	}
}

func TestNormalizeCommandSupportsArrowKeys(t *testing.T) {
	tests := map[string]string{"\x1b[A": "w", "\x1b[B": "s", "\x1b[C": "d", "\x1b[D": "a", "\x1b[H": "q", "\x1b[5~": "e", "\x1b[F": "z", "\x1b[6~": "x", "home": "q", "pgup": "e", "end": "z", "pgdn": "x", ".": "."}
	for input, want := range tests {
		if got := normalizeCommand(strings.ToLower(input)); got != want {
			t.Fatalf("normalizeCommand(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestWaitCommandAdvancesEnemies(t *testing.T) {
	g := newGame(1)
	g.player.Pos = pos{X: 5, Y: 5}
	g.current().Monsters = []*monster{{Kind: monsterOrc, Name: "Orc Brute", Glyph: 'o', Pos: pos{X: 6, Y: 5}, HP: 16, MaxHP: 16, MinDamage: 4, MaxDamage: 7}}
	startHP := g.player.HP
	g.processCommand(".")
	if g.player.HP >= startHP {
		t.Fatalf("wait should let enemies act; hp stayed at %d", g.player.HP)
	}
}

func TestDiagonalMoveLetsPlayerAttackDiagonally(t *testing.T) {
	g := openArena()
	g.player.Pos = pos{X: 5, Y: 5}
	g.current().Monsters = []*monster{{Kind: monsterRat, Name: "Dungeon Rat", Glyph: 'r', Pos: pos{X: 6, Y: 6}, HP: 1, MaxHP: 8, MinDamage: 2, MaxDamage: 4}}
	if !g.tryMove("x") {
		t.Fatal("expected diagonal action to succeed")
	}
	if g.current().Monsters[0].HP != 0 {
		t.Fatalf("expected diagonal attack to hit monster, hp=%d", g.current().Monsters[0].HP)
	}
}

func TestGhostDogTargetsPlayersFight(t *testing.T) {
	g := newGame(8)
	g.dog.Freed = true
	g.dog.Alive = true
	g.player.Pos = pos{X: 5, Y: 5}
	g.dog.Pos = pos{X: 4, Y: 5}
	target := &monster{Kind: monsterOrc, Name: "Orc Brute", Glyph: 'o', Pos: pos{X: 6, Y: 5}, HP: 12, MaxHP: 12, MinDamage: 4, MaxDamage: 7}
	g.current().Monsters = []*monster{target}
	g.attackMonster(target)
	if g.dogFocus != target {
		t.Fatal("expected dog to focus the monster the player attacked")
	}
	beforeDogAttack := target.HP
	g.dogTurn()
	if target.HP >= beforeDogAttack {
		t.Fatalf("expected dog to damage focused target, hp=%d", target.HP)
	}
}

func TestWardingCharmProtectsGhostDogForThreeEnemyTurns(t *testing.T) {
	g := newGame(10)
	g.player.Pos = pos{X: 5, Y: 5}
	g.player.WardingCharms = 1
	g.dog.Freed = true
	g.dog.Pos = pos{X: 6, Y: 5}
	g.current().Monsters = []*monster{{Kind: monsterOrc, Name: "Orc Brute", Pos: pos{X: 7, Y: 5}, HP: 1000, MaxHP: 1000, MinDamage: 7, MaxDamage: 7}}

	for turn, command := range []string{"b", ".", ".", "."} {
		beforeHP := g.dog.HP
		g.processCommand(command)
		wantDamage := 3
		if turn == 3 {
			wantDamage = 6
		}
		if got := beforeHP - g.dog.HP; got != wantDamage {
			t.Fatalf("enemy turn %d dealt %d damage to dog, want %d", turn+1, got, wantDamage)
		}
		if got, want := g.player.ShieldTurns, max(0, 2-turn); got != want {
			t.Fatalf("enemy turn %d left %d ward turns, want %d", turn+1, got, want)
		}
	}
	if g.player.WardingCharms != 0 {
		t.Fatal("casting the ward should consume one charm")
	}
}

func TestWardCannotHealGhostDogWithSmallHits(t *testing.T) {
	g := newGame(10)
	g.dog.Freed = true
	g.player.ShieldTurns = 3
	for _, damage := range []int{1, 2, 3} {
		g.hurtDog(damage, "A weak blow")
		if g.dog.HP != g.dog.MaxHP || !g.dog.Alive {
			t.Fatalf("warded hit of %d should leave dog unharmed, got %+v", damage, g.dog)
		}
	}
}

func TestWardProtectsGhostDogFromLightning(t *testing.T) {
	damage := func(shieldTurns int) int {
		g := newGame(10)
		g.player.Pos = pos{X: 5, Y: 5}
		g.player.ShieldTurns = shieldTurns
		g.dog.Freed = true
		g.dog.Pos = pos{X: 6, Y: 5}
		g.rng = newSimpleRNG(2)
		herald := newMonster(monsterStormHerald)
		herald.Pos = pos{X: 8, Y: 5}
		g.monsterTurn(&herald)
		return g.dog.MaxHP - g.dog.HP
	}
	unwarded, warded := damage(0), damage(3)
	if unwarded < 3 || unwarded > 6 {
		t.Fatalf("expected lightning to hit dog for 3-6 damage, got %d", unwarded)
	}
	if warded != unwarded-3 {
		t.Fatalf("warded lightning dealt %d damage, want %d", warded, unwarded-3)
	}
}

func TestGhostDogDeathExplainsHowToFreeIt(t *testing.T) {
	g := newGame(10)
	g.dog.Freed = true
	g.dog.Alive = true
	g.dog.HP = 1
	attacker := newMonster(monsterOrc)
	g.attackDog(&attacker)
	if g.dog.Alive || g.dog.HP != 0 {
		t.Fatalf("ghost dog should be dead at 0 HP, got alive=%t hp=%d", g.dog.Alive, g.dog.HP)
	}
	if !strings.Contains(strings.Join(g.messages, "\n"), "The Ghost Dog can only be freed now by a magic spell or by killing the Dread Lich") {
		t.Fatalf("missing ghost-dog death guidance: %q", strings.Join(g.messages, " | "))
	}
}

func TestDreadLichRestoresFallenGhostDog(t *testing.T) {
	g := newGame(18)
	g.currentLevel = levelCount - 1
	g.player.Pos = g.current().Start
	g.dog.Freed = true
	g.dog.Alive = false
	g.dog.HP = 0
	boss := findBoss(g.current())
	if boss == nil {
		t.Fatal("expected Dread Lich on the final level")
	}
	g.damageMonster(boss, boss.HP, "hits")
	if !g.dog.Alive || g.dog.HP != g.dog.MaxHP {
		t.Fatalf("Dread Lich death should restore the ghost dog, got alive=%t hp=%d/%d", g.dog.Alive, g.dog.HP, g.dog.MaxHP)
	}
	if !strings.Contains(strings.Join(g.messages, "\n"), "Ghost Dog. It reappears at full hit points") {
		t.Fatalf("missing ghost-dog restoration message: %q", strings.Join(g.messages, " | "))
	}
}

func TestGhostDogHealsBetweenFloors(t *testing.T) {
	g := newGame(10)
	g.currentLevel = 0
	g.player.Pos = g.levels[0].Stairs
	g.dog.Freed = true
	g.dog.Alive = true
	g.dog.HP = 20
	if !g.tryDescend() {
		t.Fatal("expected descend to succeed")
	}
	if g.dog.HP <= 20 {
		t.Fatalf("expected ghost dog to heal between floors, hp=%d", g.dog.HP)
	}
}

func TestDogPrefersNotToSitInDoorway(t *testing.T) {
	g := newGame(11)
	lvl := g.current()
	for y := range lvl.Tiles {
		for x := range lvl.Tiles[y] {
			lvl.Tiles[y][x] = '#'
		}
	}
	for _, p := range []pos{{5, 5}, {6, 5}, {7, 5}, {8, 5}, {7, 4}, {7, 6}} {
		lvl.Tiles[p.Y][p.X] = '.'
	}
	g.player.Pos = pos{X: 6, Y: 5}
	g.dog.Pos = pos{X: 8, Y: 5}
	best := g.bestDogNeighborNearPlayer()
	if best == nil {
		t.Fatal("expected a best neighbor for the dog")
	}
	if *best == (pos{X: 5, Y: 5}) {
		t.Fatalf("dog chose doorway tile %+v instead of room tile", *best)
	}
}

func TestGhostDogStaysNearPlayerWhenPossible(t *testing.T) {
	g := openArena()
	g.dog.Freed = true
	g.dog.Alive = true
	g.player.Pos = pos{X: 5, Y: 5}
	g.dog.Pos = pos{X: 10, Y: 10}
	g.current().Monsters = nil
	g.dogTurn()
	if distance(g.dog.Pos, g.player.Pos) > 1 {
		t.Fatalf("expected dog to return to player's room/side, dog=%+v player=%+v", g.dog.Pos, g.player.Pos)
	}
}

func TestMovingIntoGhostDogMovesDogAside(t *testing.T) {
	g := openArena()
	g.dog.Freed = true
	g.dog.Alive = true
	g.player.Pos = pos{X: 5, Y: 5}
	g.dog.Pos = pos{X: 6, Y: 5}
	g.current().Monsters = nil
	if !g.tryMove("d") {
		t.Fatal("expected move into ghost dog tile to succeed")
	}
	if g.player.Pos != (pos{X: 6, Y: 5}) {
		t.Fatalf("player pos = %+v, want {6 5}", g.player.Pos)
	}
	if g.dog.Pos == g.player.Pos {
		t.Fatalf("ghost dog did not move aside: dog=%+v player=%+v", g.dog.Pos, g.player.Pos)
	}
	if distance(g.dog.Pos, g.player.Pos) > 1 {
		t.Fatalf("ghost dog moved too far away: dog=%+v player=%+v", g.dog.Pos, g.player.Pos)
	}
}

func TestGhostDogIsTougherAndStaysClose(t *testing.T) {
	g := openArena()
	if g.dog.HP != 34 || g.dog.MaxHP != 34 {
		t.Fatalf("ghost dog hp = %d/%d, want 34/34", g.dog.HP, g.dog.MaxHP)
	}
	g.dog.Freed = true
	g.dog.Alive = true
	g.player.Pos = pos{X: 5, Y: 5}
	g.dog.Pos = pos{X: 8, Y: 5}
	g.current().Monsters = nil
	g.dogTurn()
	if distance(g.dog.Pos, g.player.Pos) >= 3 {
		t.Fatalf("ghost dog did not move closer to player: dog=%+v player=%+v", g.dog.Pos, g.player.Pos)
	}
}

func TestSearchRevealsSecretRoomAndHiddenLoot(t *testing.T) {
	g := newGame(12)
	lvl := g.current()
	for y := range lvl.Tiles {
		for x := range lvl.Tiles[y] {
			lvl.Tiles[y][x] = '#'
		}
	}
	for _, p := range []pos{{2, 2}, {3, 2}, {2, 3}, {3, 3}} {
		lvl.Tiles[p.Y][p.X] = '.'
	}
	g.player.Pos = pos{X: 3, Y: 3}
	lvl.Secret = &secretRoom{
		Door:  pos{X: 4, Y: 3},
		Tiles: []pos{{X: 5, Y: 2}, {X: 5, Y: 3}, {X: 5, Y: 4}, {X: 6, Y: 2}, {X: 6, Y: 3}, {X: 6, Y: 4}},
	}
	lvl.Items = []item{{Pos: pos{X: 5, Y: 3}, Kind: itemSunOrb, Name: "sun orb", Glyph: '*'}}
	if got := len(g.visibleItems(10)); got != 0 {
		t.Fatalf("hidden loot should stay invisible before reveal, got %d visible items", got)
	}
	if !g.search() {
		t.Fatal("search should consume a turn")
	}
	if !lvl.Secret.Revealed {
		t.Fatal("search should reveal the secret room")
	}
	if lvl.Tiles[3][4] != '.' || lvl.Tiles[3][5] != '.' {
		t.Fatal("secret door and room tiles should become passable floors")
	}
	if got := len(g.visibleItems(10)); got == 0 {
		t.Fatal("revealed secret loot should become visible")
	}
}

func TestBottomEscapeStaysSealedUntilLichDies(t *testing.T) {
	g := newGame(15)
	g.currentLevel = levelCount - 1
	lvl := g.current()
	if lvl.Secret == nil || !lvl.HasEscapeStair {
		t.Fatal("expected bottom floor to have a hidden escape sanctum")
	}
	guard, ok := floorNextToDoor(g, lvl.Secret.Door)
	if !ok {
		t.Fatal("expected a floor tile next to the hidden sanctum door")
	}
	g.player.Pos = guard
	if !g.bossAlive(lvl) {
		t.Fatal("expected the boss to still be alive")
	}
	g.search()
	if lvl.Secret.Revealed {
		t.Fatal("escape sanctum should remain sealed while the boss lives")
	}
	if !strings.Contains(strings.Join(g.messages, "\n"), "Lich still binds") {
		t.Fatalf("expected a sealed-door message, got %q", strings.Join(g.messages, " | "))
	}
}

func TestBossDeathUnlocksEscapeButDoesNotImmediatelyWin(t *testing.T) {
	g := newGame(16)
	g.currentLevel = levelCount - 1
	lvl := g.current()
	boss := findBoss(lvl)
	if boss == nil {
		t.Fatal("expected boss on final level")
	}
	g.damageMonster(boss, boss.HP, "hits")
	if g.won {
		t.Fatal("killing the boss should not immediately win the game")
	}
	if lvl.Secret == nil || !lvl.Secret.Revealed || !lvl.HasEscapeStair {
		t.Fatal("boss death should reveal the escape sanctum")
	}
	g.player.Pos = lvl.EscapeStairs
	if !g.tryAscend() {
		t.Fatal("expected hidden exit stairs to work")
	}
	if !g.won {
		t.Fatal("using the hidden exit stairs should win the game")
	}
}

func TestAscendReturnsToHigherLevel(t *testing.T) {
	g := newGame(17)
	startDown := g.levels[0].Stairs
	g.player.Pos = startDown
	if !g.tryDescend() {
		t.Fatal("expected descend to succeed")
	}
	if g.currentLevel != 1 || g.player.Pos != g.levels[1].UpStairs {
		t.Fatalf("descend landed at level=%d pos=%+v", g.currentLevel, g.player.Pos)
	}
	if !g.tryAscend() {
		t.Fatal("expected ascend to succeed from the up stairs")
	}
	if g.currentLevel != 0 || g.player.Pos != startDown {
		t.Fatalf("ascend returned to level=%d pos=%+v, want level 0 at %+v", g.currentLevel, g.player.Pos, startDown)
	}
}

func TestContentPoolsExposeNewMonstersWeaponsArmorAndMagic(t *testing.T) {
	seenMonsters := map[monsterKind]bool{}
	seenItems := map[string]bool{}
	for depth := 0; depth < levelCount; depth++ {
		g := newGame(int64(100 + depth))
		for _, m := range g.monstersForLevel(depth) {
			seenMonsters[m.Kind] = true
		}
	}
	for seed := int64(1); seed <= 30; seed++ {
		g := newGame(seed)
		for depth := 0; depth < levelCount; depth++ {
			for _, it := range g.itemsForLevel(depth) {
				seenItems[it.Name] = true
				seenItems[it.Weapon.Name] = seenItems[it.Weapon.Name] || it.Weapon.Name != ""
				seenItems[it.Armor.Name] = seenItems[it.Armor.Name] || it.Armor.Name != ""
			}
			for _, it := range g.secretTreasurePool(depth) {
				seenItems[it.Name] = true
				seenItems[it.Weapon.Name] = seenItems[it.Weapon.Name] || it.Weapon.Name != ""
				seenItems[it.Armor.Name] = seenItems[it.Armor.Name] || it.Armor.Name != ""
			}
		}
	}
	for _, want := range []monsterKind{monsterBoneHound, monsterGargoyle, monsterGraveKnight, monsterCinderDrake, monsterVoidSeer} {
		if !seenMonsters[want] {
			t.Fatalf("missing monster kind %q from dungeon content", want)
		}
	}
	for _, want := range []string{"Chapel Mace", "Hooked Glaive", "Dawnstar Flail", "Starforged Hammer", "Emberbrand", "Tempest Spear", "Voidglass Dagger", "Lichbane Greatsword", "Scout Cap", "Wolfhide Boots", "Scale Coat", "Sunplate Cuirass", "frost charm"} {
		if !seenItems[want] {
			t.Fatalf("missing content item %q from dungeon pools", want)
		}
	}
}

func TestRunShowsDungeonSaveLoadAndQuit(t *testing.T) {
	savePath := filepath.Join(t.TempDir(), "savegame.json")
	var stdout, stderr bytes.Buffer
	err := run([]string{"--seed", "3", "--save-file", savePath}, strings.NewReader("save\nload\nquit\n"), &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	plain := stripANSI(stdout.String())
	for _, want := range []string{"Ghost Dog Dungeon: Level 1/10", "Game saved to " + savePath, "Game loaded from " + savePath, "You retreat before the dungeon can claim you."} {
		if !strings.Contains(plain, want) {
			t.Fatalf("output missing %q in %q", want, plain)
		}
	}
}

func findBoss(lvl *level) *monster {
	for _, m := range lvl.Monsters {
		if m.Boss {
			return m
		}
	}
	return nil
}

func floorNextToDoor(g *game, door pos) (pos, bool) {
	for _, d := range []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		p := pos{X: door.X + d.X, Y: door.Y + d.Y}
		if g.inBounds(p) && g.current().Tiles[p.Y][p.X] == '.' {
			return p, true
		}
	}
	return pos{}, false
}

func reachable(tiles [][]rune, start, goal pos) bool {
	seen := map[pos]bool{start: true}
	queue := []pos{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == goal {
			return true
		}
		for _, next := range []pos{{X: cur.X + 1, Y: cur.Y}, {X: cur.X - 1, Y: cur.Y}, {X: cur.X, Y: cur.Y + 1}, {X: cur.X, Y: cur.Y - 1}} {
			if next.X < 0 || next.X >= mapWidth || next.Y < 0 || next.Y >= mapHeight || seen[next] || tiles[next.Y][next.X] == '#' {
				continue
			}
			seen[next] = true
			queue = append(queue, next)
		}
	}
	return false
}
