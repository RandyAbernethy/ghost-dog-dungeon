package main

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBookShowsOnlyCollectedSideStoriesThroughRoomDiscoveryAndLoading(t *testing.T) {
	g := nestedGameForTest(t, filepath.Join(t.TempDir(), "book-discoveries.json"))
	lvl := g.current()
	inner := lvl.Secret.Inner
	var outerScroll, innerScroll item
	for _, it := range lvl.Items {
		if it.Kind != itemSideStoryScroll {
			continue
		}
		if it.SideStoryID == lvl.Secret.SideStoryID {
			outerScroll = it
		} else if it.SideStoryID == inner.SideStoryID {
			innerScroll = it
		}
	}
	if outerScroll.SideStoryID == 0 || innerScroll.SideStoryID == 0 {
		t.Fatal("the generated rooms need separate side-story scrolls")
	}
	checkBook := func(g *game, discovered ...int) {
		t.Helper()
		known := map[int]bool{}
		for _, id := range discovered {
			known[id] = true
		}
		before := append([]int(nil), g.player.SideStories...)
		beforeStats := g.stats
		var out bytes.Buffer
		g.renderBook(&out)
		for id, story := range ghostDogSideStories {
			if strings.Contains(out.String(), story.Title) != known[id+1] || strings.Contains(out.String(), story.Text) != known[id+1] {
				t.Fatalf("side story %q visibility should match its collected scroll", story.Title)
			}
		}
		if !reflect.DeepEqual(before, g.player.SideStories) || beforeStats != g.stats {
			t.Fatal("reading the book must not discover stories or spend a turn")
		}
	}
	checkBook(g)
	g.revealSecretRoom(lvl)
	checkBook(g)
	g.player.Pos = outerScroll.Pos
	g.collectItems()
	checkBook(g, outerScroll.SideStoryID)
	revealRoom(lvl, inner)
	checkBook(g, outerScroll.SideStoryID)
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	checkBook(loaded, outerScroll.SideStoryID)
	loaded.player.Pos = innerScroll.Pos
	loaded.collectItems()
	checkBook(loaded, outerScroll.SideStoryID, innerScroll.SideStoryID)
	if err := loaded.save(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadGame(loaded.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	checkBook(reloaded, outerScroll.SideStoryID, innerScroll.SideStoryID)
}
