package main

import "fmt"

func (g *game) secretPlan() ([levelCount]int, int) {
	var counts [levelCount]int
	counts[levelCount-1] = 1 // The escape sanctum is included in the total.
	for remaining := 4 + g.rng.Intn(4); remaining > 0; remaining-- {
		choices := []int{}
		for depth, count := range counts {
			weight := 0
			switch count {
			case 0:
				weight = 50
			case 1:
				weight = 18
			case 2:
				weight = 3
			}
			for i := 0; i < weight; i++ {
				choices = append(choices, depth)
			}
		}
		counts[choices[g.rng.Intn(len(choices))]]++
	}
	nested := -1
	if g.rng.Intn(100) < 40 {
		choices := []int{}
		for depth, count := range counts[:levelCount-1] {
			if count >= 2 {
				choices = append(choices, depth)
			}
		}
		if len(choices) > 0 {
			nested = choices[g.rng.Intn(len(choices))]
		}
	}
	return counts, nested
}

func (g *game) buildDungeon() error {
	counts, nested := g.secretPlan()
	storyIDs := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for i := len(storyIDs) - 1; i > 0; i-- {
		j := g.rng.Intn(i + 1)
		storyIDs[i], storyIDs[j] = storyIDs[j], storyIDs[i]
	}
	nextStory := 0
	g.levels = make([]*level, levelCount)
	for depth := 0; depth < levelCount; depth++ {
		style := layoutKinds[g.rng.Intn(len(layoutKinds))]
		theme := floorThemes[g.rng.Intn(len(floorThemes))]
		var built *level
		for attempt := 0; attempt < 128; attempt++ {
			tiles, start := makeStyledFloorPlan(g.rng, depth, style)
			actualStyle := style
			if attempt >= 96 {
				tiles, start = compactFloorPlan(depth, attempt%2 == 0)
				actualStyle = layoutChambers
			}
			lvl := &level{Index: depth, Tiles: tiles, Start: start, Layout: actualStyle, Theme: theme}
			if depth == levelCount-1 {
				g.addEscapeSanctum(lvl)
				if !lvl.HasEscapeStair {
					continue
				}
			}
			if depth == nested {
				g.addNestedSecretRoom(lvl)
				if lvl.Secret == nil || lvl.Secret.Inner == nil {
					continue
				}
			}
			for len(lvl.secretRooms()) < counts[depth] {
				if !g.placeSecretRoom(lvl, 2+g.rng.Intn(3), 2+g.rng.Intn(3)) && !g.placeSecretRoom(lvl, 2, 3) {
					break
				}
			}
			if len(lvl.secretRooms()) != counts[depth] {
				continue
			}
			built = lvl
			break
		}
		if built == nil {
			return fmt.Errorf("could not place floor %d for seed %d", depth+1, g.seed)
		}
		g.levels[depth] = built
		for _, hidden := range built.secretRooms() {
			if hidden.SideStoryID != 0 {
				continue
			}
			hidden.SideStoryID = storyIDs[nextStory]
			nextStory++
		}
		g.addSecretTreasure(built)
		g.populateLevel(built)
		g.addSpecialRoom(built)
		g.arrangeEncounters(built)
		g.addFloorEvent(built)
	}
	return nil
}
