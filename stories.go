package main

import (
	"fmt"
	"io"
	"sort"
)

var ghostDogStory = [...]string{
	"A small dog named Ash lived in the chapel yard, where the bell was his favorite song.",
	"Ash chose Mara, the bellkeeper's daughter, and followed her everywhere through the village.",
	"One winter, a stranger in a crown of bone came asking for the chapel's buried names.",
	"When the villagers refused, the stranger opened a stair beneath the chapel and called the dead upward.",
	"Ash barked until Mara woke. She led the children through a back passage while he drew the dead away.",
	"The stranger was the Dread Lich. He struck Ash with a curse meant to silence him forever.",
	"Mara chained Ash beside the stair, hoping the silver links would keep the Lich's curse from spreading.",
	"The curse killed Ash, but his loyalty held his spirit fast. He guarded the stair and waited for a kind voice.",
	"The Lich fed on Ash's bound spirit to keep his own heart beating. Only the Lich's death can end that bond.",
	"The final page is in the Lich's keeping: when he falls, Ash is free to choose his own way home.",
}

type sideStory struct {
	Title string
	Text  string
}

var ghostDogSideStories = [...]sideStory{
	{"The Stolen Glove", "Mara could never find her left glove on cold mornings. Ash carried it to the chapel steps and sat on it until she came outside. She called him a thief, then stayed to watch the sunrise with him."},
	{"The Chainmaker", "The smith who forged Ash's silver chain filed every link smooth. He thought a kind chain could make captivity gentle. After Mara left, he kept the rough filings in a jar and could not bring himself to throw them away."},
	{"A Light in the Rain", "When the chapel roof leaked, Ash followed each falling drop with his nose. Mara moved a lantern from puddle to puddle while the children laughed. For one stormy evening, the chapel had a hundred little moons."},
	{"The Watchman's Bread", "An old watchman always broke his supper in two. Ash waited for the smaller half without begging. Years later, the empty chair beside the chapel gate still had crumbs tucked into its cracks."},
	{"The Arrow and the Bell", "A village archer once aimed at a crow on the chapel bell. Ash barked and spoiled the shot. The archer grumbled until he saw the nest above it; the next day he brought Ash a strip of smoked meat."},
	{"The Bell That Would Not Ring", "Mara lost her voice during a winter fever. Ash slept beneath the silent bell rope for three days. When she finally rang it again, he howled so loudly that nobody noticed how weak the bell sounded."},
	{"Roots Beneath the Chapel", "Ash dug at the great tree until he uncovered a child's wooden soldier. Mara returned it to a gray-haired farmer who had lost it fifty years before. He cried into Ash's fur and called it dust in his eyes."},
	{"Warmth for a Stranger", "A stray hound appeared at the chapel gate during the first snow. Ash pushed his own blanket toward it and lay across the doorway. By morning there were two dogs under the blanket, and Mara had to step over both."},
	{"The Door in a Dream", "Long after Ash was bound below, Mara dreamed of a door opening onto the chapel yard. A pale dog waited on the other side. She never saw who opened it, but she left a clean bowl beside the waking door."},
	{"The Unfinished Song", "Mara wrote a small song for Ash, but she never found its final line. In the margin she left these words: When the bell rings and the road is open, let him choose where home will be."},
	{"The Room Behind the Room", "Mara hid her keepsakes in a little room behind the chapel pantry. Ash found the loose stone and nudged it open with his nose. Inside lay a ribbon, a wooden bell, and a drawing of a small dog. She tied the ribbon to his collar and left the door open whenever they were alone."},
}

func hasStoryID(ids []int, id int) bool {
	for _, known := range ids {
		if known == id {
			return true
		}
	}
	return false
}

func (g *game) addSecretSideStory(lvl *level) {
	for _, hidden := range lvl.secretRooms() {
		g.addRoomSideStory(lvl, hidden)
	}
}

func (g *game) addRoomSideStory(lvl *level, hidden *secretRoom) {
	id := hidden.SideStoryID
	if id == 0 {
		id = lvl.Index + 1
	}
	if id < 1 || id > len(ghostDogSideStories) {
		return
	}
	if hasStoryID(g.player.SideStories, id) {
		return
	}
	for _, it := range lvl.Items {
		if it.Kind == itemSideStoryScroll && it.SideStoryID == id {
			return
		}
	}
	for _, p := range hidden.floorTiles() {
		if g.itemAt(lvl, p) == nil && g.fountainAt(lvl, p) == nil && (!lvl.HasEscapeStair || p != lvl.EscapeStairs) {
			lvl.Items = append(lvl.Items, item{Pos: p, Kind: itemSideStoryScroll, Name: "Side-story scroll: " + ghostDogSideStories[id-1].Title, Glyph: '?', SideStoryID: id})
			return
		}
	}
}

func (g *game) renderBook(w io.Writer) {
	fmt.Fprintln(w, "\n=== Book of Ghost Dog ===")
	fmt.Fprintln(w, "The book keeps every story scroll you have found. Reading takes no turn.")
	chapters := append([]int(nil), g.player.StoryChapters...)
	sort.Ints(chapters)
	fmt.Fprintln(w, "\nAsh's story:")
	seen := map[int]bool{}
	for _, id := range chapters {
		if id >= 1 && id <= len(ghostDogStory) && !seen[id] {
			fmt.Fprintf(w, "Chapter %d/10: %s\n", id, ghostDogStory[id-1])
			seen[id] = true
		}
	}
	if len(seen) == 0 {
		fmt.Fprintln(w, "No chapters found yet.")
	}
	fmt.Fprintln(w, "\nSide stories:")
	seen = map[int]bool{}
	for _, id := range g.player.SideStories {
		if id >= 1 && id <= len(ghostDogSideStories) && !seen[id] {
			story := ghostDogSideStories[id-1]
			fmt.Fprintf(w, "%s\n%s\n\n", story.Title, story.Text)
			seen[id] = true
		}
	}
	if len(seen) == 0 {
		fmt.Fprintln(w, "No side stories found yet.")
	}
}
