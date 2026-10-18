package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func objectiveArena(kind objectiveKind) *game {
	g := openArena()
	g.current().Objective = &objectiveRoom{
		Kind: kind, Pos: pos{5, 5}, Area: room{4, 4, 3, 3}, Integrity: 3,
		Reward: item{Kind: itemWardingCharm, Name: "warding charm", Glyph: '*'},
	}
	if kind == objectiveSigils {
		g.current().Objective.Sigils = []ritualSigil{{Pos: pos{5, 5}}, {Pos: pos{6, 5}}, {Pos: pos{6, 6}}}
	}
	return g
}

func saveAndReloadAdventure(t *testing.T, g *game) *game {
	t.Helper()
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.levels, loaded.levels) || !reflect.DeepEqual(g.adventure, loaded.adventure) || g.rng.State != loaded.rng.State || !reflect.DeepEqual(g.player, loaded.player) {
		t.Fatal("saving must preserve objective, reward, story and combat state exactly")
	}
	return loaded
}

func TestSpiritVigilTurnsSaveAndOneTimeReward(t *testing.T) {
	g := objectiveArena(objectiveSpirit)
	rng := g.rng.State
	g.processCommand("y")
	o := g.current().Objective
	if !o.Started || o.Progress != 0 || g.stats.Turns != 1 {
		t.Fatal("starting the vigil must spend one turn without counting it toward protection")
	}
	g.processCommand("y")
	g.processCommand("v") // No fountain: failed action.
	var out bytes.Buffer
	if err := g.loop(strings.NewReader("m\nbook\nmap\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	g.quit = false
	if o.Progress != 0 || g.stats.Turns != 1 || g.rng.State != rng {
		t.Fatal("reading and failed actions must not protect the spirit or spend RNG")
	}
	g.processCommand(".")
	g = saveAndReloadAdventure(t, g)
	g.processCommand(".")
	g.processCommand(".")
	o = g.current().Objective
	if !o.Completed || o.Progress != 3 || g.stats.Turns != 4 || g.player.WardingCharms != 1 || len(g.adventure.ObjectivesDone) != 1 || g.rng.State != rng {
		t.Fatal("three protection turns must award the saved reward once")
	}
	g = saveAndReloadAdventure(t, g)
	g.processCommand("y")
	g.player.Pos = g.current().Stairs
	g.processCommand(">")
	g.processCommand("<")
	g.processCommand(".")
	if !g.levels[0].Objective.Completed || g.player.WardingCharms != 1 || len(g.adventure.ObjectivesDone) != 1 {
		t.Fatal("revisits, interactions and saves must never renew an objective reward")
	}
}

func TestSpiritCanFailOrBeGuardedByLivingAsh(t *testing.T) {
	for _, tc := range []struct {
		name  string
		freed bool
		alive bool
		fail  bool
	}{{"active Ash", true, true, false}, {"chained Ash", false, true, true}, {"fallen Ash", true, false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			g := objectiveArena(objectiveSpirit)
			g.processCommand("y")
			g.player.Pos = pos{12, 5}
			g.dog.Freed, g.dog.Alive, g.dog.Pos = tc.freed, tc.alive, pos{5, 6}
			// Evaluate after enemy actions while Ash is still guarding the room.
			g.stats.Turns++
			g.finishAdventureTurn()
			o := g.current().Objective
			if o.Failed != tc.fail || (tc.fail && g.player.WardingCharms != 0) || (!tc.fail && o.Progress != 1) {
				t.Fatal("only a freed, living companion may keep the vigil in the player's absence")
			}
		})
	}
	g := objectiveArena(objectiveSpirit)
	g.processCommand("y")
	m := newMonster(monsterSkeleton)
	m.Pos, m.MinDamage, m.MaxDamage = pos{6, 5}, 0, 0
	g.current().Monsters = []*monster{&m}
	for i := 0; i < 3; i++ {
		g.processCommand(".")
	}
	if !g.current().Objective.Failed || g.current().Objective.Completed || g.player.WardingCharms != 0 {
		t.Fatal("three turns with an adjacent enemy must extinguish the spirit without a reward")
	}
	g = objectiveArena(objectiveSpirit)
	g.processCommand("y")
	g.player.Pos = g.current().Stairs
	g.processCommand(">")
	if !g.levels[0].Objective.Failed {
		t.Fatal("leaving the floor must abandon an active vigil")
	}
	g = objectiveArena(objectiveSpirit)
	g.processCommand("y")
	g.player.HP = 0
	g.stats.Turns += 3
	g.finishAdventureTurn()
	if g.current().Objective.Completed || g.player.WardingCharms != 0 {
		t.Fatal("a dead player cannot finish the vigil")
	}
}

func TestRitualSigilsRequireDistinctSuccessfulInteractions(t *testing.T) {
	g := objectiveArena(objectiveSigils)
	rng := g.rng.State
	g.processCommand("y")
	g.processCommand("y")
	if g.stats.Turns != 1 || g.current().Objective.Progress != 1 {
		t.Fatal("a broken sigil must not advance the ritual twice or spend another turn")
	}
	g = saveAndReloadAdventure(t, g)
	g.processCommand("d")
	g.processCommand("y")
	g.processCommand("s")
	g.processCommand("y")
	if !g.current().Objective.Completed || g.current().Objective.Progress != 3 || g.stats.Turns != 5 || g.player.WardingCharms != 1 || g.rng.State != rng {
		t.Fatal("breaking all three sigils must spend three interaction turns and award one fixed reward")
	}
	g = saveAndReloadAdventure(t, g)
	g.processCommand("y")
	if g.stats.Turns != 5 || g.player.WardingCharms != 1 || len(g.adventure.ObjectivesDone) != 1 {
		t.Fatal("a completed ritual cannot be repeated after loading")
	}
}

func TestObjectiveMarkerTakesPriorityOverNearbyFreeEvent(t *testing.T) {
	g := objectiveArena(objectiveSigils)
	g.current().Events = []floorEvent{{Kind: eventGhost, Pos: pos{4, 5}}}
	g.processCommand("y")
	if g.current().Objective.Progress != 1 || g.stats.Turns != 1 || g.current().Events[0].Used {
		t.Fatal("a nearby gift offer must not prevent breaking the sigil underfoot")
	}
}

func TestObjectiveArrivalPromptsAndRenderingAreQuiet(t *testing.T) {
	g := objectiveArena(objectiveSigils)
	g.player.Pos = pos{5, 8}
	g.processCommand("w")
	g.processCommand("w")
	if strings.Count(strings.Join(g.messages, "\n"), "Three ritual sigils") != 1 {
		t.Fatal("entering the objective must show its instructions")
	}
	before := *g.current().Objective
	beforeSigils := append([]ritualSigil(nil), before.Sigils...)
	log := append([]string(nil), g.messages...)
	var out bytes.Buffer
	g.render(&out)
	g.renderInspect(&out)
	if !reflect.DeepEqual(before, *g.current().Objective) || !reflect.DeepEqual(beforeSigils, g.current().Objective.Sigils) || !reflect.DeepEqual(log, g.messages) {
		t.Fatal("display and inspection must not mutate objective or log state")
	}
	g.processCommand("a")
	if strings.Contains(strings.Join(g.messages, "\n"), "Three ritual sigils") {
		t.Fatal("moving within the room must not repeat arrival prompts")
	}
}

func TestObjectiveGenerationIsRareReachableAndReproducible(t *testing.T) {
	seeds := int64(200)
	if testing.Short() {
		seeds = 25
	}
	counts := map[objectiveKind]int{}
	withRoom := 0
	for seed := int64(1); seed <= seeds; seed++ {
		g := newGame(seed)
		rooms := 0
		for depth, lvl := range g.levels {
			o := lvl.Objective
			if o == nil {
				continue
			}
			rooms++
			counts[o.Kind]++
			if depth < 2 || depth > 7 || o.Started || o.Completed || o.Failed || o.Progress != 0 {
				t.Fatalf("seed %d: invalid initial objective state or floor", seed)
			}
			reachable := tileDistances(lvl.Tiles, lvl.Start)
			secret := lvl.secretFootprint(true)
			for y := o.Area.Y; y < o.Area.Y+o.Area.H; y++ {
				for x := o.Area.X; x < o.Area.X+o.Area.W; x++ {
					p := pos{x, y}
					if _, ok := reachable[p]; !ok || secret[p] || g.monsterAt(lvl, p) != nil || g.itemAt(lvl, p) != nil || g.fountainAt(lvl, p) != nil || lvl.eventAt(p) != nil || p == lvl.Start || p == lvl.Stairs || p == lvl.UpStairs {
						t.Fatalf("seed %d: objective overlaps a feature or cannot be reached at %+v", seed, p)
					}
				}
			}
			repeat := newGame(seed)
			if !reflect.DeepEqual(o, repeat.levels[depth].Objective) || g.rng.State != repeat.rng.State {
				t.Fatalf("seed %d: room and reward must reproduce exactly", seed)
			}
		}
		if rooms > 1 {
			t.Fatal("a dungeon may have at most one optional objective")
		}
		if rooms == 1 {
			withRoom++
		}
	}
	if withRoom < int(seeds)/5 || withRoom > int(seeds)/2 || len(counts) != 2 {
		t.Fatalf("objectives should be uncommon and both kinds possible: %d/%d, %v", withRoom, seeds, counts)
	}
}
