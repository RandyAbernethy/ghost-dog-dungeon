package main

import (
	"fmt"
	"strings"
)

func (g *game) processCommand(cmd string) {
	if g.won || g.quit || g.player.HP <= 0 {
		return
	}
	g.turnDamage = make(map[*monster]int)
	if g.player.WebbedTurns > 0 && isMoveCommand(cmd) {
		g.player.WebbedTurns--
		g.addMessage("Sticky webs hold you in place for a turn.")
		g.stats.Turns++
		g.advanceEnemies()
		return
	}

	acted := false
	if strings.HasPrefix(cmd, "equip ") {
		acted = g.equipCommand(cmd)
	} else if cmd == "drop" || strings.HasPrefix(cmd, "drop ") {
		acted = g.dropCommand(cmd)
	} else {
		switch cmd {
		case "w", "a", "s", "d", "q", "e", "z", "x":
			acted = g.tryMove(cmd)
		case ".":
			acted = true
			g.addMessage("You wait and let the dungeon move around you.")
		case ">":
			acted = g.tryDescend()
		case "<":
			acted = g.tryAscend()
		case "p":
			acted = g.usePotion()
		case "f":
			acted = g.useFireScroll()
		case "b":
			acted = g.useWardingCharm()
		case "g":
			acted = g.useBlinkStone()
		case "u":
			acted = g.useSunOrb()
		case "t":
			acted = g.useFrostCharm()
		case "o":
			acted = g.useStarfireOrb()
		case "n":
			acted = g.usePhoenixAsh()
		case "r":
			acted = g.useGhostRecallScroll()
		case "v":
			acted = g.drinkFromFountain()
		case "k":
			acted = g.search()
		case "search":
			acted = g.search()
		case "y", "interact":
			acted = g.interactEvent()
		case "choose 1", "choose 2":
			acted = g.chooseGhostGift(cmd[len(cmd)-1] - '0')
		case "save":
			if err := g.save(); err != nil {
				g.addMessage("Save failed: " + err.Error())
			} else {
				g.addMessage("Game saved to " + g.saveFile)
			}
		case "load":
			if g.saveFile == "" {
				g.addMessage("No save file is selected; restart with --load-file path to load a run.")
				break
			}
			loaded, err := loadGame(g.saveFile)
			if err != nil {
				g.addMessage("Load failed: " + err.Error())
			} else {
				recordsFile := g.recordsFile
				*g = *loaded
				g.recordsFile = recordsFile
				g.dogFocus = nil
				g.addMessage("Game loaded from " + g.saveFile)
			}
		case "h":
			g.addMessage("Keys: wasd/arrows move, qezx diagonals, . wait, </> stairs, p/f/b/g/u/t/o/n items, r recall, v fountain, y interact, i inventory, E equipment, B Book of Ghost Dog, j challenges, c codex, m inspect, k search, Esc map, S save, L load. Type :equip NUMBER, :drop NUMBER, or :choose 1/2 + Enter; Ctrl-C quits.")
		case "quit", "exit":
			g.quit = true
		default:
			g.addMessage("Unknown command. Press h for help.")
		}
	}

	if acted {
		g.stats.Turns++
		if !g.won && g.player.HP > 0 {
			g.advanceEnemies()
		}
	}
}

func normalizeCommand(cmd string) string {
	switch cmd {
	case "\x1b":
		return "map"
	case "[a", "up":
		return "w"
	case "[b", "down":
		return "s"
	case "[d", "left":
		return "a"
	case "[c", "right":
		return "d"
	case "[h", "[1~", "oh", "home":
		return "q"
	case "[5~", "pgup", "pg up", "pageup":
		return "e"
	case "[f", "[4~", "[8~", "of", "end":
		return "z"
	case "[6~", "pgdn", "pg dn", "pagedown":
		return "x"
	default:
		return cmd
	}
}

func isMoveCommand(cmd string) bool {
	switch cmd {
	case "w", "a", "s", "d", "q", "e", "z", "x":
		return true
	default:
		return false
	}
}

func moveDelta(cmd string) (int, int, bool) {
	switch normalizeCommand(cmd) {
	case "w":
		return 0, -1, true
	case "s":
		return 0, 1, true
	case "a":
		return -1, 0, true
	case "d":
		return 1, 0, true
	case "q":
		return -1, -1, true
	case "e":
		return 1, -1, true
	case "z":
		return -1, 1, true
	case "x":
		return 1, 1, true
	default:
		return 0, 0, false
	}
}

func (g *game) tryMove(cmd string) bool {
	dx, dy, ok := moveDelta(cmd)
	if !ok {
		return false
	}
	next := pos{X: g.player.Pos.X + dx, Y: g.player.Pos.Y + dy}
	lvl := g.current()
	if !g.inBounds(next) || lvl.Tiles[next.Y][next.X] == '#' {
		g.addMessage("Stone blocks your path.")
		return false
	}
	if m := g.monsterAt(lvl, next); m != nil {
		g.attackMonster(m)
		return true
	}
	for step := 2; step <= g.player.Weapon.effectiveReach(); step++ {
		target := pos{X: g.player.Pos.X + dx*step, Y: g.player.Pos.Y + dy*step}
		if !g.inBounds(target) || lvl.Tiles[target.Y][target.X] == '#' {
			break
		}
		if m := g.monsterAt(lvl, target); m != nil {
			g.attackMonster(m)
			return true
		}
	}
	if g.dog.Freed && g.dog.Alive && g.dog.Pos == next {
		if !g.stepDogAside(next, g.player.Pos) {
			g.addMessage("Ghost dog has no space to drift aside.")
			return false
		}
		g.player.Pos = next
		g.addMessage("Ghost dog drifts aside and keeps pace with you.")
		g.collectItems()
		return true
	}
	g.player.Pos = next
	if lvl.DogChain != nil && *lvl.DogChain == next && !g.dog.Freed {
		g.dog.Freed = true
		g.placeDogNearPlayer()
		g.addMessage("You free the ghost dog from its silver chain. It chooses you immediately.")
	}
	g.collectItems()
	return true
}

func (g *game) stepDogAside(target, fallback pos) bool {
	candidates := []pos{fallback}
	for _, p := range g.openNeighbors(target, true) {
		candidates = append(candidates, p)
	}
	seen := map[pos]bool{}
	for _, p := range candidates {
		if seen[p] {
			continue
		}
		seen[p] = true
		if g.canDogOccupyAfterPlayerMove(p, target) {
			g.dog.Pos = p
			return true
		}
	}
	return false
}

func (g *game) canDogOccupyAfterPlayerMove(p, futurePlayer pos) bool {
	if !g.inBounds(p) || g.current().Tiles[p.Y][p.X] == '#' || p == futurePlayer {
		return false
	}
	return g.monsterAt(g.current(), p) == nil
}

func (g *game) search() bool {
	lvl := g.current()
	hidden := lvl.hiddenRoomNear(g.player.Pos)
	if hidden == nil {
		g.addMessage("You search the walls, but find no hidden seams.")
		return true
	}
	if distance(g.player.Pos, hidden.Door) > 1 {
		g.addMessage("You search nearby stone, but uncover nothing unusual.")
		return true
	}
	if hidden == lvl.Secret && lvl.HasEscapeStair && g.bossAlive(lvl) {
		g.addMessage("Ancient death-magic locks the hidden seam tight. The Lich still binds it.")
		return true
	}
	revealRoom(lvl, hidden)
	g.addMessage("Your search finds a hollow seam. A secret door swings open.")
	return true
}

func (g *game) revealSecretRoom(lvl *level) {
	revealRoom(lvl, lvl.Secret)
}

func revealRoom(lvl *level, hidden *secretRoom) {
	if hidden == nil || hidden.Revealed {
		return
	}
	hidden.Revealed = true
	lvl.Tiles[hidden.Door.Y][hidden.Door.X] = '.'
	for _, p := range hidden.floorTiles() {
		lvl.Tiles[p.Y][p.X] = '.'
	}
}

func (g *game) tryDescend() bool {
	lvl := g.current()
	if !lvl.HasStair || g.player.Pos != lvl.Stairs {
		g.addMessage("There are no stairs beneath your feet.")
		return false
	}
	for i := range lvl.Fountains {
		lvl.Fountains[i].Used = false
	}
	g.currentLevel++
	nextLevel := g.current()
	g.player.Pos = nextLevel.UpStairs
	firstVisit := !nextLevel.Visited
	nextLevel.Visited = true
	g.afterFloorTransition(fmt.Sprintf("You descend to level %d.", g.currentLevel+1), firstVisit)
	g.collectItems()
	return true
}

func (g *game) tryAscend() bool {
	lvl := g.current()
	if lvl.HasEscapeStair && lvl.Secret != nil && lvl.Secret.Revealed && g.player.Pos == lvl.EscapeStairs {
		g.won = true
		g.addMessage("You climb the hidden stairs and leave the dungeon behind.")
		return true
	}
	if !lvl.HasUpStair || g.player.Pos != lvl.UpStairs {
		g.addMessage("There are no stairs leading up here.")
		return false
	}
	for i := range lvl.Fountains {
		lvl.Fountains[i].Used = false
	}
	g.currentLevel--
	prevLevel := g.current()
	prevLevel.Visited = true
	g.player.Pos = prevLevel.Stairs
	g.afterFloorTransition(fmt.Sprintf("You climb back to level %d.", g.currentLevel+1), false)
	g.regenerateStairGuardians(prevLevel)
	g.collectItems()
	return true
}

func (g *game) afterFloorTransition(message string, healDog bool) {
	if g.dog.Freed && g.dog.Alive {
		if healDog {
			heal := g.randRange(2, 3)
			g.dog.HP = min(g.dog.MaxHP, g.dog.HP+heal)
			g.addMessage(fmt.Sprintf("Ghost dog recovers %d health on this new depth. (%d/%d)", heal, g.dog.HP, g.dog.MaxHP))
		}
		g.placeDogNearPlayer()
	}
	g.addMessage(message)
}

func (g *game) regenerateStairGuardians(lvl *level) {
	if !lvl.HasStair {
		return
	}
	want := 2 + g.rng.Intn(2)
	spots := make([]pos, 0, want)
	for radius := 1; radius <= 2 && len(spots) < want; radius++ {
		for y := lvl.Stairs.Y - radius; y <= lvl.Stairs.Y+radius && len(spots) < want; y++ {
			for x := lvl.Stairs.X - radius; x <= lvl.Stairs.X+radius && len(spots) < want; x++ {
				p := pos{X: x, Y: y}
				if distance(p, lvl.Stairs) != radius || !g.inBounds(p) || lvl.Tiles[y][x] == '#' || p == g.player.Pos ||
					(!g.dog.Freed && lvl.DogChain != nil && p == *lvl.DogChain) {
					continue
				}
				if g.monsterAt(lvl, p) != nil || g.itemAt(lvl, p) != nil || (g.dog.Freed && g.dog.Alive && g.dog.Pos == p) {
					continue
				}
				spots = append(spots, p)
			}
		}
	}
	pool := monsterPoolForLevel(lvl.Index)
	if len(spots) == 0 || len(pool) == 0 {
		return
	}
	count := min(want, len(spots))
	for i := 0; i < count; i++ {
		spotIndex := g.rng.Intn(len(spots))
		spawnPos := spots[spotIndex]
		spots = append(spots[:spotIndex], spots[spotIndex+1:]...)
		spec := pool[g.rng.Intn(len(pool))]
		for spec.Boss && len(pool) > 1 {
			spec = pool[g.rng.Intn(len(pool))]
		}
		spawn := scaleMonsterForDepth(spec, lvl.Index)
		spawn.Pos = spawnPos
		lvl.Monsters = append(lvl.Monsters, &spawn)
	}
	g.addMessage("Stairwell guardians stir behind you as you return.")
}
