package main

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDungeonSecretCountAndReachability(t *testing.T) {
	for seed := int64(-1); seed <= 200; seed++ {
		g := newGame(seed)
		if g.generationError != nil {
			t.Fatalf("seed %d: %v", seed, g.generationError)
		}
		total := 0
		for depth, lvl := range g.levels {
			if lvl.primarySecret() == nil {
				continue
			}
			total += len(lvl.secretRooms())
			if len(lvl.secretRooms()) > 3 {
				t.Fatalf("seed %d floor %d has too many chambers", seed, depth+1)
			}
			g.currentLevel = depth
			occupied := map[pos]bool{}
			for _, hidden := range lvl.secretRooms() {
				for _, p := range append([]pos{hidden.Door}, hidden.Tiles...) {
					if occupied[p] {
						t.Fatalf("seed %d floor %d: secret chambers overlap", seed, depth+1)
					}
					occupied[p] = true
				}
				if hidden.Revealed || lvl.Tiles[hidden.Door.Y][hidden.Door.X] != '#' {
					t.Fatalf("seed %d floor %d starts with a discovered room", seed, depth+1)
				}
				for _, p := range hidden.Tiles {
					if lvl.Tiles[p.Y][p.X] != '#' {
						t.Fatalf("seed %d floor %d exposes hidden tiles", seed, depth+1)
					}
				}
				if next := hidden.Next; next != nil {
					assertLinkedSecretGeometry(t, lvl, hidden, next, seed)
				}
			}
			for _, hidden := range lvl.secretRooms() {
				approach, ok := floorNextToDoor(g, hidden.Door)
				if !ok || !reachable(lvl.Tiles, lvl.Start, approach) {
					t.Fatalf("seed %d floor %d has an unreachable secret door", seed, depth+1)
				}
				revealRoom(lvl, hidden)
				for _, p := range hidden.Tiles {
					if !reachable(lvl.Tiles, lvl.Start, p) {
						t.Fatalf("seed %d floor %d has unreachable secret-room contents", seed, depth+1)
					}
				}
			}
			for _, parent := range lvl.secretRooms() {
				if parent.Next == nil {
					continue
				}
				blocked := cloneTiles(lvl.Tiles)
				for _, p := range parent.Tiles {
					blocked[p.Y][p.X] = '#'
				}
				if reachable(blocked, lvl.Start, parent.Next.Tiles[0]) {
					t.Fatalf("seed %d floor %d: second chamber has a route bypassing the first", seed, depth+1)
				}
			}
		}
		if total < 5 || total > 8 {
			t.Fatalf("seed %d: %d secret rooms, want 5-8", seed, total)
		}
	}
}

func linkedGameForTest(t *testing.T, path string) *game {
	t.Helper()
	for seed := int64(1); seed <= 100; seed++ {
		g := newGameWithSaveDir(seed, filepath.Dir(path))
		if g.generationError != nil {
			t.Fatal(g.generationError)
		}
		for depth, lvl := range g.levels {
			if lvl.primarySecret() != nil && lvl.primarySecret().Next != nil {
				g.currentLevel = depth
				g.player.Pos = lvl.Start
				return g
			}
		}
	}
	t.Fatal("expected occasional nested chambers across seeds")
	return nil
}

func TestLinkedRoomDiscoveryRewardsAndFountainRenewal(t *testing.T) {
	g := linkedGameForTest(t, "")
	lvl := g.current()
	lvl.Monsters = nil
	outer, inner := lvl.primarySecret(), lvl.primarySecret().Next
	approach, ok := floorNextToDoor(g, outer.Door)
	if !ok {
		t.Fatal("outer door has no approach")
	}
	g.player.Pos = approach
	g.processCommand("search")
	if !outer.Revealed || inner.Revealed || g.stats.Turns != 1 {
		t.Fatal("the first search should discover only the outer room and spend one turn")
	}
	for _, p := range append([]pos{inner.Door}, inner.Tiles...) {
		if lvl.Tiles[p.Y][p.X] != '#' {
			t.Fatal("opening the outer room must leave the inner chamber concealed")
		}
	}
	var scroll item
	var fountainPos pos
	for _, it := range lvl.Items {
		if it.Kind == itemSideStoryScroll && it.SideStoryID == inner.SideStoryID {
			scroll = it
		}
	}
	for _, p := range inner.Tiles {
		if g.fountainAt(lvl, p) != nil {
			fountainPos = p
		}
	}
	if scroll.SideStoryID == 0 || fountainPos == (pos{}) {
		t.Fatal("the inner room needs its own side story and fountain")
	}
	var display bytes.Buffer
	g.renderBook(&display)
	if strings.Contains(display.String(), ghostDogSideStories[inner.SideStoryID-1].Title) {
		t.Fatal("an undiscovered inner scroll must not appear in the book")
	}
	approach, ok = floorNextToDoor(g, inner.Door)
	if !ok || reachable(lvl.Tiles, lvl.Start, scroll.Pos) || reachable(lvl.Tiles, lvl.Start, fountainPos) {
		t.Fatal("the inner door should be approachable while its rewards stay inaccessible")
	}
	walkToSecretTestTile(t, g, approach)
	for i := range lvl.Fountains {
		lvl.Fountains[i].Used = true
	}
	if !strings.Contains(g.contextHint(), "try search") {
		t.Fatal("the inner door should have the usual suspicious-wall hint")
	}
	g.processCommand("search")
	if !inner.Revealed || lvl.Tiles[inner.Door.Y][inner.Door.X] != '.' {
		t.Fatal("a second search from the outer room should open the inner door")
	}
	walkToSecretTestTile(t, g, scroll.Pos)
	display.Reset()
	g.renderBook(&display)
	if !hasStoryID(g.player.SideStories, inner.SideStoryID) || !strings.Contains(display.String(), ghostDogSideStories[inner.SideStoryID-1].Text) {
		t.Fatal("finding the inner scroll should add its tale to the book")
	}
	walkToSecretTestTile(t, g, fountainPos)
	g.fountainAt(lvl, fountainPos).Used = false
	g.player.HP = 10
	g.dog.Freed, g.dog.HP, g.dog.Pos = true, 1, approach
	if !g.drinkFromFountain() || g.player.HP < 14 || g.dog.HP < 9 {
		t.Fatal("the inner fountain should heal the player and Ghost Dog")
	}
	g.player.Pos = lvl.Stairs
	if !g.tryDescend() || !g.tryAscend() || g.fountainAt(lvl, fountainPos).Used || !inner.Revealed {
		t.Fatal("returning should renew the inner fountain and preserve its discovery")
	}
}

func TestLinkedRoomSavePreservesSeparateDiscoveries(t *testing.T) {
	for _, revealInner := range []bool{false, true} {
		name := "outer only"
		if revealInner {
			name = "both rooms"
		}
		t.Run(name, func(t *testing.T) {
			g := linkedGameForTest(t, filepath.Join(t.TempDir(), "nested.json"))
			lvl := g.current()
			g.revealSecretRoom(lvl)
			if revealInner {
				revealRoom(lvl, lvl.primarySecret().Next)
			}
			for i := range lvl.Fountains {
				lvl.Fountains[i].Used = true
			}
			if err := g.save(); err != nil {
				t.Fatal(err)
			}
			loaded, err := loadGame(g.saveFile)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(lvl.primarySecret(), loaded.current().primarySecret()) || !reflect.DeepEqual(lvl.Tiles, loaded.current().Tiles) || !reflect.DeepEqual(lvl.Items, loaded.current().Items) || !reflect.DeepEqual(lvl.Fountains, loaded.current().Fountains) {
				t.Fatal("save/load must preserve each door, hidden tiles, rewards, and spent fountains")
			}
			if !revealInner {
				approach, ok := floorNextToDoor(loaded, loaded.current().primarySecret().Next.Door)
				if !ok {
					t.Fatal("loading should preserve access to the inner door")
				}
				loaded.player.Pos = approach
				loaded.search()
				if !loaded.current().primarySecret().Next.Revealed {
					t.Fatal("the inner chamber should remain discoverable after loading")
				}
			}
		})
	}
}

// Walk through the actual movement commands to check that the linked room's
// entrance and rewards can be reached in play, rather than by teleporting.
func walkToSecretTestTile(t *testing.T, g *game, target pos) {
	t.Helper()
	start := g.player.Pos
	previous := map[pos]pos{start: start}
	commands := map[pos]string{}
	queue := []pos{start}
	directions := []struct {
		delta pos
		cmd   string
	}{{pos{1, 0}, "d"}, {pos{-1, 0}, "a"}, {pos{0, 1}, "s"}, {pos{0, -1}, "w"}}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p == target {
			break
		}
		for _, dir := range directions {
			next := pos{X: p.X + dir.delta.X, Y: p.Y + dir.delta.Y}
			if _, seen := previous[next]; seen || !g.inBounds(next) || g.current().Tiles[next.Y][next.X] == '#' {
				continue
			}
			previous[next], commands[next] = p, dir.cmd
			queue = append(queue, next)
		}
	}
	if _, found := previous[target]; !found {
		t.Fatalf("cannot reach %+v from %+v", target, start)
	}
	var path []string
	for p := target; p != start; p = previous[p] {
		path = append(path, commands[p])
	}
	for i := len(path) - 1; i >= 0; i-- {
		g.processCommand(path[i])
	}
	if g.player.Pos != target {
		t.Fatalf("movement stopped at %+v before reaching %+v", g.player.Pos, target)
	}
}

func assertLinkedSecretGeometry(t *testing.T, lvl *level, parent, next *secretRoom, seed int64) {
	t.Helper()
	parentTiles := map[pos]bool{}
	x0, x1, y0, y1 := mapWidth, 0, mapHeight, 0
	for _, p := range parent.Tiles {
		parentTiles[p] = true
		x0, x1, y0, y1 = min(x0, p.X), max(x1, p.X), min(y0, p.Y), max(y1, p.Y)
	}
	if len(parentTiles) != (x1-x0+1)*(y1-y0+1) {
		t.Fatalf("seed %d floor %d: first secret chamber has an enclosed hole", seed, lvl.Index+1)
	}
	touchesParent := false
	for _, p := range append([]pos{next.Door}, next.Tiles...) {
		if parentTiles[p] || (p.X >= x0 && p.X <= x1 && p.Y >= y0 && p.Y <= y1) {
			t.Fatalf("seed %d floor %d: second chamber overlaps or is enclosed by the first", seed, lvl.Index+1)
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				q := pos{X: p.X + dx, Y: p.Y + dy}
				if q.X < 0 || q.X >= mapWidth || q.Y < 0 || q.Y >= mapHeight || lvl.Tiles[q.Y][q.X] != '#' {
					t.Fatalf("seed %d floor %d: second chamber can be approached from an ordinary corridor", seed, lvl.Index+1)
				}
				if parentTiles[q] {
					if p != next.Door {
						t.Fatalf("seed %d floor %d: chambers touch without a separating wall", seed, lvl.Index+1)
					}
					if dx*dx+dy*dy == 1 {
						touchesParent = true
					}
				}
			}
		}
	}
	if !touchesParent {
		t.Fatalf("seed %d floor %d: second door does not connect to the first chamber", seed, lvl.Index+1)
	}
	for _, p := range next.Tiles {
		for _, dir := range []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			q := pos{X: p.X + dir.X, Y: p.Y + dir.Y}
			if q == next.Door {
				return
			}
		}
	}
	t.Fatalf("seed %d floor %d: second door does not connect to its chamber", seed, lvl.Index+1)
}
