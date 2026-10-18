package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestEveryRoomRelicGuaranteesAnEpicOnItsOwnFloor(t *testing.T) {
	for _, treasure := range relicPool() {
		for _, seed := range []int64{63, 108, 79} {
			t.Run(fmt.Sprintf("%s/seed_%d", treasure.Name, seed), func(t *testing.T) {
				g := openArena()
				g.currentLevel, g.rng = 7, newSimpleRNG(seed)
				previous := newMonster(monsterIronColossus)
				previous.Pos = pos{20, 10}
				g.levels[6].Monsters = []*monster{&previous}
				lvl := g.current()
				treasure.Pos = pos{12, 10}
				lvl.Items = []item{treasure}
				g.addEpicEncounter(lvl)
				if len(lvl.Monsters) != 1 || !epicMonster(lvl.Monsters[0].Kind) || g.levels[6].Monsters[0] != &previous {
					t.Fatal("every relic must get its own Epic even when another floor already has one")
				}
				epic := lvl.Monsters[0]
				if distance(epic.Pos, lvl.Start) < 8 || distance(epic.Pos, lvl.Stairs) < 6 || epic.Pos == treasure.Pos {
					t.Fatal("the guaranteed Epic must be clear of loot, entrances, and stairs")
				}
				blocked := cloneTiles(lvl.Tiles)
				blocked[epic.Pos.Y][epic.Pos.X] = '#'
				if len(tileDistances(blocked, lvl.Start)) != countFloorTiles(lvl.Tiles)-1 {
					t.Fatal("the guaranteed Epic must be bypassable")
				}
				for _, dead := range []bool{false, true} {
					if dead {
						epic.HP = 0
					}
					before := g.rng.State
					g.addEpicEncounter(lvl)
					if len(lvl.Monsters) != 1 || lvl.Monsters[0] != epic || g.rng.State != before {
						t.Fatal("an existing encounter, alive or dead, must not generate another Epic or consume RNG")
					}
				}
				g.currentLevel = 8
				before := g.rng.State
				g.addEpicEncounter(g.current())
				if len(g.current().Monsters) != 0 || g.rng.State != before {
					t.Fatal("a floor without a relic must not add a third Epic")
				}
			})
		}
	}
}

func TestNonRelicEpicsKeepTheirTwelvePercentChance(t *testing.T) {
	for _, tc := range []struct {
		seed int64
		roll int
		want int
	}{{116, 0, 1}, {63, 11, 1}, {108, 12, 0}, {79, 99, 0}} {
		t.Run(fmt.Sprintf("roll_%d", tc.roll), func(t *testing.T) {
			g := openArena()
			g.currentLevel, g.rng = 6, newSimpleRNG(tc.seed)
			expected := newSimpleRNG(tc.seed)
			if roll := expected.Intn(100); roll != tc.roll {
				t.Fatalf("seed fixture produced roll %d, want %d", roll, tc.roll)
			}
			g.addEpicEncounter(g.current())
			if len(g.current().Monsters) != tc.want {
				t.Fatalf("roll %d produced %d encounters, want %d", tc.roll, len(g.current().Monsters), tc.want)
			}
			if tc.want == 0 && g.rng.State != expected.State {
				t.Fatal("a rejected random roll must not retry")
			}
		})
	}
}

func TestRelicEpicPositionIsReservedAcrossFloorPopulation(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		g := openArena()
		g.currentLevel, g.rng = 6, newSimpleRNG(seed)
		lvl := g.current()
		lvl.HasStair = false
		treasure := relicPool()[int(seed)%len(relicPool())]
		treasure.Pos = pos{12, 10}
		lvl.Items = []item{treasure}
		if err := g.populateLevel(lvl); err != nil {
			t.Fatal(err)
		}
		epic := lvl.epicEncounter()
		if epic == nil {
			t.Fatal("population must reserve the relic's Epic before placing ordinary actors")
		}
		spot := epic.Pos
		g.addSpecialRoom(lvl)
		g.arrangeEncounters(lvl)
		g.addFloorEvent(lvl)
		g.placeFountains()
		g.addEpicEncounter(lvl)
		ordinary := []monster{}
		seen := map[pos]bool{}
		count := 0
		for _, m := range lvl.Monsters {
			if seen[m.Pos] {
				t.Fatal("population placed two monsters on the same tile")
			}
			seen[m.Pos] = true
			if epicMonster(m.Kind) {
				count++
			} else {
				ordinary = append(ordinary, *m)
			}
		}
		if count != 1 || epic.Pos != spot || distance(spot, lvl.Start) < 8 || distance(spot, lvl.Stairs) < 6 {
			t.Fatal("floor features and encounter layouts must preserve one distant relic Epic")
		}
		if g.itemAt(lvl, spot) != nil || g.fountainAt(lvl, spot) != nil || lvl.eventAt(spot) != nil {
			t.Fatal("floor features must not overlap the reserved Epic")
		}
		if rosterDifficulty(ordinary) != lvl.InitialDifficulty || !fairFloorRoster(ordinary, lvl.Index) {
			t.Fatal("a guaranteed Epic must remain outside the ordinary roster budget")
		}
	}
}

func TestImpossibleRelicEpicPlacementReturnsUsefulError(t *testing.T) {
	g := openArena()
	g.currentLevel = 6
	lvl := g.current()
	lvl.Tiles = blankFloor()
	carveRoom(lvl.Tiles, room{X: 3, Y: 3, W: 5, H: 5})
	lvl.HasStair = false
	treasure := relicPool()[0]
	treasure.Pos = pos{6, 6}
	lvl.Items = []item{treasure}
	if err := g.populateLevel(lvl); err == nil || !strings.Contains(err.Error(), "floor 7 has no safe position") {
		t.Fatalf("a malformed floor must report its missing encounter instead of silently skipping it: %v", err)
	}
}
