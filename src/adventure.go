package main

import (
	"fmt"
	"io"
	"strings"
)

// Optional fields keep format-2 saves readable without inventing past events.
type adventureState struct {
	RescueFloor     int      `json:"rescue_floor,omitempty"`
	AshMoments      []string `json:"ash_moments,omitempty"`
	LastMomentTurn  int      `json:"last_moment_turn,omitempty"`
	LastMomentFloor int      `json:"last_moment_floor,omitempty"`
	EpicsDefeated   []string `json:"epics_defeated,omitempty"`
	ObjectivesDone  []string `json:"objectives_done,omitempty"`
}

func (g *game) ashMoment(id, message string) {
	if !g.dog.Freed || !g.dog.Alive {
		return
	}
	for _, seen := range g.adventure.AshMoments {
		if seen == id {
			return
		}
	}
	g.adventure.AshMoments = append(g.adventure.AshMoments, id)
	g.adventure.LastMomentTurn = g.stats.Turns
	g.adventure.LastMomentFloor = g.currentLevel + 1
	g.addMessage(message)
}

func (g *game) finishAdventureTurn() {
	g.advancePoison()
	g.advanceObjective()
	if g.won || g.player.HP <= 0 || !g.dog.Freed || !g.dog.Alive ||
		g.adventure.LastMomentFloor == g.currentLevel+1 || g.stats.Turns < g.adventure.LastMomentTurn+15 {
		return
	}
	for _, m := range g.current().Monsters {
		if m.HP > 0 && (distance(m.Pos, g.player.Pos) <= 6 || distance(m.Pos, g.dog.Pos) <= 6) {
			return
		}
	}
	switch {
	case g.player.HP*4 <= g.player.MaxHP:
		g.ashMoment("injured", "Ash presses against your knee. His outline is faint, but he refuses to leave your side.")
	case g.dog.HP*3 <= g.dog.MaxHP:
		g.ashMoment("flickering", "Ash's outline flickers. Even now, he turns back to make sure you are following.")
	case g.current().Theme == themeChapel:
		g.ashMoment("chapel", "Ash lifts his ears toward a bell you cannot hear. For a moment, his tail remembers how to wag.")
	case g.current().Theme == themeKennel:
		g.ashMoment("kennels", "Ash noses an empty sleeping place, then returns to sit beside you. This is not home anymore.")
	case g.current().Theme == themeOvergrown:
		g.ashMoment("roots", "Ash chases a drifting seed through the roots, then looks back as if hoping you saw.")
	}
}

func (g *game) collectedSideStory(title string) bool {
	for i, story := range ghostDogSideStories {
		if story.Title == title && hasStoryID(g.player.SideStories, i+1) {
			return true
		}
	}
	return false
}

func (g *game) renderEnding(w io.Writer) {
	fmt.Fprintln(w, "Beside the door, you see a clean bowl filled with fresh water.")
	together := g.dog.Freed && g.dog.Alive
	memories := []struct {
		title string
		line  string
	}{
		{"The Stolen Glove", "You remember Mara's stolen glove, and hope there will be small, foolish mornings again."},
		{"The Chainmaker", "You leave the chains below. Kindness will have to find a better shape than silver links."},
		{"A Light in the Rain", "Rain catches the light outside. For a moment, you can picture Mara's hundred little moons."},
		{"The Watchman's Bread", "At the next supper, you will remember to leave a share for a hungry traveler."},
		{"The Arrow and the Bell", "A bird calls above the doorway. You think of the nest Ash once saved beneath the bell."},
		{"The Bell That Would Not Ring", "Somewhere beyond the door, you imagine a weak bell ringing, and a dog answering louder than grief."},
		{"Roots Beneath the Chapel", "You carry the story of a wooden soldier back into the world. Some lost things can still be returned."},
		{"Warmth for a Stranger", "If a stranger waits beyond this door, you decide there will be room for one more beside the fire."},
		{"The Room Behind the Room", "You remember the little room behind Mara's pantry. This time, you leave the door open."},
	}
	for _, memory := range memories {
		if g.collectedSideStory(memory.title) {
			fmt.Fprintln(w, memory.line)
		}
	}
	if g.collectedSideStory("The Door in a Dream") {
		if together {
			fmt.Fprintln(w, "You recognize the bowl from Mara's dream. Ash crosses the threshold as though someone has been waiting all this time.")
		} else {
			fmt.Fprintln(w, "You recognize the bowl from Mara's dream. You leave it beside the open door, keeping a place for Ash.")
		}
	}
	if g.collectedSideStory("The Unfinished Song") {
		fmt.Fprintln(w, "You open Mara's unfinished song. At last, you find its final line:")
		if together {
			fmt.Fprintln(w, "When the bell rings and the road is open, home is the hand that opens the door.")
		} else {
			fmt.Fprintln(w, "When the bell rings and the road is open, let the light remain for those still finding home.")
		}
	}
	if together && g.stats.GhostDogFalls > 0 {
		fmt.Fprintln(w, "Ash fell in the dark, and you brought him back. This time, he follows you into the light.")
	}
}

func (g *game) renderChronicle(w io.Writer) {
	fmt.Fprintln(w, "\n=== Your Adventure ===")
	deepest := g.currentLevel + 1
	for i, lvl := range g.levels {
		if lvl.Visited {
			deepest = max(deepest, i+1)
		}
	}
	fmt.Fprintf(w, "Your journey reached floor %d of the dungeon.\n", deepest)
	secrets := 0
	for _, lvl := range g.levels {
		for _, hidden := range lvl.secretRooms() {
			if hidden.Revealed {
				secrets++
			}
		}
	}
	fmt.Fprintf(w, "Secrets uncovered: %d. Your book holds %d chapters and %d side stories.\n", secrets, len(g.player.StoryChapters), len(g.player.SideStories))
	if g.adventure.RescueFloor > 0 {
		fmt.Fprintf(w, "You freed Ash from his silver chain on floor %d.\n", g.adventure.RescueFloor)
	}
	if g.stats.GhostDogFalls > 0 {
		if g.stats.GhostDogFalls == 1 {
			fmt.Fprint(w, "Ash fell once.")
		} else {
			fmt.Fprintf(w, "Ash fell %d times.", g.stats.GhostDogFalls)
		}
		if g.dog.Freed && g.dog.Alive {
			fmt.Fprint(w, " You brought him back.")
		}
		fmt.Fprintln(w)
	}
	if len(g.adventure.EpicsDefeated) > 0 {
		fmt.Fprintln(w, "You defeated "+strings.Join(g.adventure.EpicsDefeated, " and ")+".")
	}
	for _, objective := range g.adventure.ObjectivesDone {
		fmt.Fprintln(w, "You completed "+objective+".")
	}
	fmt.Fprintln(w, "Your final weapon was "+g.player.Weapon.Name+".")
	if g.crypt != nil && g.crypt.Floor.Visited {
		if g.crypt.Completed {
			fmt.Fprintln(w, "You crossed "+cryptName+" and found its far secret door.")
		} else {
			fmt.Fprintln(w, "You dared to enter "+cryptName+".")
		}
		for _, m := range g.crypt.Floor.Monsters {
			if m.Kind == monsterUnderLich && m.HP == 0 {
				fmt.Fprintln(w, "You defeated the Under Lich, guardian of the crypt.")
			}
		}
	}
	if g.won && g.dog.Freed && g.dog.Alive {
		fmt.Fprintln(w, "Together, you and Ash found the way home.")
	} else if g.won {
		fmt.Fprintln(w, "You found the way out. Ash's place beside you remains empty.")
	} else {
		fmt.Fprintln(w, "Your road ended here. The dungeon will remember.")
	}
}
