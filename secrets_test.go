package main

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDungeonGuaranteesSecretFloorsAndNestedRoom(t *testing.T) {
	for seed := int64(-1); seed <= 200; seed++ {
		g := newGame(seed)
		floors, nested := 0, 0
		for depth, lvl := range g.levels {
			if lvl.Secret == nil {
				continue
			}
			floors++
			g.currentLevel = depth
			for _, hidden := range lvl.Secret.rooms() {
				if hidden.Revealed || lvl.Tiles[hidden.Door.Y][hidden.Door.X] != '#' {
					t.Fatalf("seed %d floor %d starts with a discovered room", seed, depth+1)
				}
				for _, p := range hidden.Tiles {
					if lvl.Tiles[p.Y][p.X] != '#' {
						t.Fatalf("seed %d floor %d exposes hidden tiles", seed, depth+1)
					}
				}
				if inner := hidden.Inner; inner != nil {
					nested++
					inside := map[pos]bool{}
					for _, p := range hidden.Tiles {
						inside[p] = true
					}
					for _, p := range append(append([]pos{inner.Door}, inner.Tiles...), inner.Walls...) {
						if !inside[p] {
							t.Fatalf("seed %d floor %d puts its inner room outside its parent", seed, depth+1)
						}
					}
				}
			}
			for _, hidden := range lvl.Secret.rooms() {
				approach, ok := floorNextToDoor(g, hidden.Door)
				if !ok || !reachable(lvl.Tiles, lvl.Start, approach) {
					t.Fatalf("seed %d floor %d has an unreachable secret door", seed, depth+1)
				}
				revealRoom(lvl, hidden)
				for _, p := range hidden.floorTiles() {
					if !reachable(lvl.Tiles, lvl.Start, p) {
						t.Fatalf("seed %d floor %d has unreachable secret-room contents", seed, depth+1)
					}
				}
			}
		}
		if floors < (levelCount+1)/2 || nested < 1 {
			t.Fatalf("seed %d: %d secret floors and %d nested rooms", seed, floors, nested)
		}
	}
}

func TestNestedRoomDiscoveryRewardsAndFountainRenewal(t *testing.T) {
	g := newGame(14)
	lvl := g.current()
	lvl.Monsters = nil
	outer, inner := lvl.Secret, lvl.Secret.Inner
	approach, ok := floorNextToDoor(g, outer.Door)
	if !ok {
		t.Fatal("outer door has no approach")
	}
	g.player.Pos = approach
	g.processCommand("search")
	if !outer.Revealed || inner.Revealed || g.stats.Turns != 1 {
		t.Fatal("the first search should discover only the outer room and spend one turn")
	}
	for _, p := range append(append([]pos{inner.Door}, inner.Tiles...), inner.Walls...) {
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
	for _, p := range inner.Walls {
		if p != inner.Door && lvl.Tiles[p.Y][p.X] != '#' {
			t.Fatal("the inner chamber's surrounding walls should remain intact")
		}
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

func TestNestedRoomSavePreservesSeparateDiscoveries(t *testing.T) {
	for _, revealInner := range []bool{false, true} {
		name := "outer only"
		if revealInner {
			name = "both rooms"
		}
		t.Run(name, func(t *testing.T) {
			g := newGameWithSaveFile(14, filepath.Join(t.TempDir(), "nested.json"))
			lvl := g.current()
			g.revealSecretRoom(lvl)
			if revealInner {
				revealRoom(lvl, lvl.Secret.Inner)
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
			if !reflect.DeepEqual(lvl.Secret, loaded.current().Secret) || !reflect.DeepEqual(lvl.Tiles, loaded.current().Tiles) || !reflect.DeepEqual(lvl.Items, loaded.current().Items) || !reflect.DeepEqual(lvl.Fountains, loaded.current().Fountains) {
				t.Fatal("save/load must preserve each door, hidden tiles, rewards, and spent fountains")
			}
			if !revealInner {
				approach, ok := floorNextToDoor(loaded, loaded.current().Secret.Inner.Door)
				if !ok {
					t.Fatal("loading should preserve access to the inner door")
				}
				loaded.player.Pos = approach
				loaded.search()
				if !loaded.current().Secret.Inner.Revealed {
					t.Fatal("the inner chamber should remain discoverable after loading")
				}
			}
		})
	}
}

// Walk through the actual movement commands to check that the enclosed room's
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
