package main

import (
	"reflect"
	"testing"
)

func TestGenerationVarietyAndPlayableFeaturesAcrossSeeds(t *testing.T) {
	var maps [levelCount]map[string]bool
	var starts [levelCount]map[pos]bool
	var styles [levelCount]map[layoutKind]bool
	var themes [levelCount]map[floorTheme]bool
	for depth := 0; depth < levelCount; depth++ {
		maps[depth] = map[string]bool{}
		starts[depth] = map[pos]bool{}
		styles[depth] = map[layoutKind]bool{}
		themes[depth] = map[floorTheme]bool{}
	}
	counts := map[int]int{}
	nestedFloors := map[int]bool{}
	eventKinds := map[eventKind]bool{}
	roomKinds := map[specialRoomKind]bool{}
	withNesting, withoutNesting := 0, 0
	for seed := int64(1); seed <= 300; seed++ {
		g := newGame(seed)
		if g.generationError != nil {
			t.Fatalf("seed %d: %v", seed, g.generationError)
		}
		total, nested := 0, false
		storyIDs := map[int]bool{}
		for depth, lvl := range g.levels {
			maps[depth][encodeTiles(lvl.Tiles)] = true
			starts[depth][lvl.Start] = true
			styles[depth][lvl.Layout] = true
			themes[depth][lvl.Theme] = true
			if len(tileDistances(lvl.Tiles, lvl.Start)) != floorBudget(depth) || countFloorTiles(lvl.Tiles) != floorBudget(depth) {
				t.Fatalf("seed %d floor %d: disconnected or wrong-sized floor", seed, depth+1)
			}
			if lvl.HasStair && !reachable(lvl.Tiles, lvl.Start, lvl.Stairs) {
				t.Fatal("stairs are unreachable")
			}
			occupied := map[pos]bool{lvl.Start: true}
			if lvl.HasStair {
				occupied[lvl.Stairs] = true
			}
			if lvl.DogChain != nil {
				if !reachable(lvl.Tiles, lvl.Start, *lvl.DogChain) {
					t.Fatal("dog cannot be rescued")
				}
				occupied[*lvl.DogChain] = true
			}
			for _, m := range lvl.Monsters {
				if occupied[m.Pos] || lvl.Tiles[m.Pos.Y][m.Pos.X] == '#' || distance(m.Pos, lvl.Start) < 2 {
					t.Fatalf("seed %d floor %d: monster overlaps a feature or crowds the entrance", seed, depth+1)
				}
				occupied[m.Pos] = true
				if m.Boss && distance(m.Pos, lvl.Start) < 10 {
					t.Fatal("the Lich is too close to the entry")
				}
			}
			for _, it := range lvl.Items {
				if occupied[it.Pos] {
					t.Fatalf("seed %d floor %d: loot overlaps another feature", seed, depth+1)
				}
				occupied[it.Pos] = true
			}
			for _, f := range lvl.Fountains {
				if occupied[f.Pos] {
					t.Fatal("fountain overlaps another feature")
				}
				occupied[f.Pos] = true
			}
			reserved := lvl.secretFootprint(false)
			for _, e := range lvl.Events {
				if occupied[e.Pos] || reserved[e.Pos] || !reachable(lvl.Tiles, lvl.Start, e.Pos) {
					t.Fatal("event overlaps another feature or cannot be reached")
				}
				occupied[e.Pos] = true
				eventKinds[e.Kind] = true
				if e.Kind == eventLever && (e.Shortcut == nil || lvl.Tiles[e.Shortcut.Y][e.Shortcut.X] != '#' || lvl.secretFootprint(true)[*e.Shortcut]) {
					t.Fatal("lever should open an ordinary wall without bypassing a secret room")
				}
			}
			if lvl.Special != nil {
				roomKinds[lvl.Special.Kind] = true
			}
			count := len(lvl.secretRooms())
			counts[count]++
			total += count
			if count > 3 {
				t.Fatal("a floor has more than three secret chambers")
			}
			for _, hidden := range lvl.secretRooms() {
				if hidden.Inner != nil {
					nested = true
					nestedFloors[depth] = true
				}
				if storyIDs[hidden.SideStoryID] || hidden.SideStoryID < 1 || hidden.SideStoryID > len(ghostDogSideStories) {
					t.Fatal("every secret chamber needs a distinct side story")
				}
				storyIDs[hidden.SideStoryID] = true
				fountains, scrolls := 0, 0
				for _, p := range hidden.floorTiles() {
					if lvl.Tiles[p.Y][p.X] != '#' {
						t.Fatal("secret chamber is exposed before discovery")
					}
					if g.fountainAt(lvl, p) != nil {
						fountains++
					}
					if it := g.itemAt(lvl, p); it != nil && it.Kind == itemSideStoryScroll && it.SideStoryID == hidden.SideStoryID {
						scrolls++
					}
				}
				if fountains != 1 || scrolls != 1 {
					t.Fatal("every secret chamber must have its own fountain and side-story scroll")
				}
			}
		}
		if total < 5 || total > 8 {
			t.Fatalf("seed %d has %d secret chambers", seed, total)
		}
		if nested {
			withNesting++
		} else {
			withoutNesting++
		}
	}
	for depth := 0; depth < levelCount; depth++ {
		if len(maps[depth]) < 270 || len(starts[depth]) < 30 || len(styles[depth]) != 4 || len(themes[depth]) != 4 {
			t.Fatalf("floor %d lacks variety: maps=%d starts=%d styles=%d themes=%d", depth+1, len(maps[depth]), len(starts[depth]), len(styles[depth]), len(themes[depth]))
		}
	}
	for count := 0; count <= 3; count++ {
		if counts[count] == 0 {
			t.Fatalf("no floors with %d chambers", count)
		}
	}
	if counts[3]*20 >= levelCount*300 || withNesting == 0 || withoutNesting == 0 || len(nestedFloors) < 5 {
		t.Fatalf("unexpected secret distribution: counts=%v nested runs=%d/%d floors=%v", counts, withNesting, withoutNesting, nestedFloors)
	}
	if len(roomKinds) != 4 || len(eventKinds) != 4 {
		t.Fatalf("missing room/event variety: rooms=%v events=%v", roomKinds, eventKinds)
	}
	t.Logf("300 runs: chamber counts per floor=%v; nested runs=%d; nested floors=%d", counts, withNesting, len(nestedFloors))
}

func TestStyledLayoutsHaveDifferentGeometry(t *testing.T) {
	caveEdges, cryptEdges := 0, 0
	for seed := int64(1); seed <= 50; seed++ {
		for _, style := range []layoutKind{layoutCaves, layoutCrypt} {
			tiles, start := makeStyledFloorPlan(newSimpleRNG(seed), 5, style)
			if len(tileDistances(tiles, start)) != floorBudget(5) {
				t.Fatal("styled floor is disconnected")
			}
			edges := 0
			for y := 1; y < mapHeight-1; y++ {
				for x := 1; x < mapWidth-1; x++ {
					if tiles[y][x] == '.' {
						edges += openCardinalNeighbors(tiles, pos{x, y})
					}
				}
			}
			if style == layoutCaves {
				caveEdges += edges
			} else {
				cryptEdges += edges
			}
		}
	}
	if cryptEdges >= caveEdges {
		t.Fatal("crypts should have narrower passages than the caves, rather than merely different labels")
	}
}

func TestThemesPreserveFloorDifficultyAndSeedReproduction(t *testing.T) {
	for depth := 0; depth < levelCount; depth++ {
		for _, theme := range floorThemes {
			for seed := int64(1); seed <= 30; seed++ {
				base := (&game{rng: newSimpleRNG(seed)}).monstersForLevel(depth)
				g := &game{rng: newSimpleRNG(seed)}
				actual := g.themedMonsters(&level{Index: depth, Theme: theme})
				allowed := map[monsterKind]bool{}
				for _, m := range monsterPoolForLevel(depth) {
					allowed[m.Kind] = true
				}
				baseThreat, actualThreat, bosses := 0, 0, 0
				for _, m := range base {
					baseThreat += monsterThreat(m)
				}
				for _, m := range actual {
					actualThreat += monsterThreat(m)
					if m.Boss {
						bosses++
					}
					if !allowed[m.Kind] || m != scaleMonsterForDepth(newMonster(m.Kind), depth) {
						t.Fatal("theme changed monster stats or exceeded the floor's pool")
					}
				}
				if len(actual) != len(base) || actualThreat > baseThreat*110/100 || actualThreat < baseThreat*90/100 || (depth == levelCount-1 && bosses != 1) || (depth < levelCount-1 && bosses != 0) {
					t.Fatal("theme changed the encounter budget or boss count")
				}
			}
		}
	}
	for _, seed := range []int64{-7, 1, 23, 128, 9001} {
		a, b := newGame(seed), newGame(seed)
		if a.generationError != nil || b.generationError != nil || !reflect.DeepEqual(a.levels, b.levels) || a.rng.State != b.rng.State {
			t.Fatal("a seed must reproduce layout, secrets, themes, encounters, rewards, and events")
		}
	}
}
