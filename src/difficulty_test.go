package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDifficultyPricesDurabilityDamageArmorAndPowers(t *testing.T) {
	plain := newMonster(monsterGoblin)
	for name, tougher := range map[string]monster{
		"health": func() monster { m := plain; m.MaxHP += 5; return m }(),
		"damage": func() monster { m := plain; m.MaxDamage += 2; return m }(),
		"armor":  func() monster { m := plain; m.Kind = monsterRustGuard; return m }(),
		"drain":  func() monster { m := plain; m.Kind = monsterWraith; return m }(),
	} {
		if monsterDifficulty(tougher) <= monsterDifficulty(plain) {
			t.Fatalf("%s should increase difficulty", name)
		}
	}
	slow := newMonster(monsterIronrootBrute)
	fast := slow
	fast.Kind = monsterChapelSentinel
	if monsterDifficulty(slow) >= monsterDifficulty(fast) {
		t.Fatal("resting every other turn should discount the brute's difficulty")
	}
	wounded := plain
	wounded.HP = 1
	if monsterDifficulty(wounded) != monsterDifficulty(plain) {
		t.Fatal("wounding a monster must not reduce the points earned for killing it")
	}
}

func TestBudgetedRostersStayBalancedAndVaryAcrossSeeds(t *testing.T) {
	if testing.Short() {
		t.Skip("1,000-seed roster sweep runs in the full suite")
	}
	var counts [levelCount]map[int]bool
	var totals, health, damage [levelCount]int
	seen, absent := map[monsterKind]bool{}, map[monsterKind]bool{}
	for depth := range counts {
		counts[depth] = map[int]bool{}
	}
	for seed := int64(1); seed <= 1000; seed++ {
		g := &game{rng: newSimpleRNG(seed)}
		previousBudget, previousActual := 0, 0
		present := map[monsterKind]bool{}
		for depth := 0; depth < levelCount; depth++ {
			lvl := &level{Index: depth, Theme: floorThemes[int(seed)%len(floorThemes)]}
			roster, err := g.monstersForLevel(lvl)
			if err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
			low, high := floorDifficultyBounds(depth)
			if lvl.DifficultyBudget < low || lvl.DifficultyBudget > high || lvl.InitialDifficulty < low || lvl.InitialDifficulty > high || !fairFloorRoster(roster, depth) {
				t.Fatalf("seed %d floor %d exceeds its difficulty or stat limits: %+v", seed, depth+1, lvl)
			}
			if lvl.DifficultyBudget <= previousBudget || lvl.InitialDifficulty <= previousActual || lvl.InitialDifficulty != rosterDifficulty(roster) {
				t.Fatalf("seed %d: floor difficulty should strictly increase", seed)
			}
			previousBudget, previousActual = lvl.DifficultyBudget, lvl.InitialDifficulty
			counts[depth][len(roster)] = true
			totals[depth] += lvl.InitialDifficulty
			copies := map[monsterKind]int{}
			bosses, borrowed := 0, 0
			local := map[monsterKind]bool{}
			for _, spec := range monsterPoolForLevel(depth) {
				local[spec.Kind] = true
			}
			for _, m := range roster {
				if m.Boss {
					bosses++
				} else {
					copies[m.Kind]++
					if !local[m.Kind] {
						borrowed += monsterDifficulty(m)
					}
				}
				if copies[m.Kind] > 3 {
					t.Fatal("more than three copies of a monster")
				}
				health[depth] += m.MaxHP
				damage[depth] += m.MinDamage + m.MaxDamage
				present[m.Kind], seen[m.Kind] = true, true
			}
			if borrowed > lvl.DifficultyBudget/4 {
				t.Fatalf("floor %d relies too heavily on earlier monsters", depth+1)
			}
			wantBosses := 0
			if depth == levelCount-1 {
				wantBosses = 1
			}
			if bosses != wantBosses {
				t.Fatal("only the final floor should have exactly one Dread Lich")
			}
		}
		for _, extra := range additionalMonsters {
			if !present[extra.Kind] {
				absent[extra.Kind] = true
			}
		}
	}
	for depth := 0; depth < levelCount; depth++ {
		if len(counts[depth]) < 2 {
			t.Fatalf("floor %d still has a fixed monster count", depth+1)
		}
		if abs(totals[depth]/1000-floorDifficultyBaselines[depth]) > max(2, floorDifficultyBaselines[depth]*3/100) ||
			abs(health[depth]/1000-floorHealthBaselines[depth]) > max(2, floorHealthBaselines[depth]*5/100) ||
			abs(damage[depth]/1000-floorDamageBaselines[depth]) > max(3, floorDamageBaselines[depth]*5/100) {
			t.Fatalf("floor %d drifts from the calibrated averages: points=%d HP=%d damage=%d", depth+1, totals[depth]/1000, health[depth]/1000, damage[depth]/1000)
		}
	}
	for _, extra := range additionalMonsters {
		if !seen[extra.Kind] || !absent[extra.Kind] {
			t.Fatalf("%s must be possible without appearing in every run", extra.Kind)
		}
	}
}

func TestFallbackRostersAndStairGuardiansFitTheirBudgets(t *testing.T) {
	for depth := 0; depth < levelCount; depth++ {
		low, high := floorDifficultyBounds(depth)
		var reserved []monster
		if depth == levelCount-1 {
			reserved = []monster{scaleMonsterForDepth(newMonster(monsterBoss), depth)}
		}
		for _, theme := range floorThemes {
			roster, err := fallbackRoster(encounterPool(depth, theme), reserved, (low+high)/2, low, high, max(1, depth-1), depth+3,
				func(ms []monster) bool { return fairFloorRoster(ms, depth) })
			if err != nil || !fairFloorRoster(roster, depth) || rosterDifficulty(roster) < low || rosterDifficulty(roster) > high {
				t.Fatalf("floor %d fallback is unbalanced: err=%v roster=%+v", depth+1, err, roster)
			}
		}
		for slots := 1; slots <= 3; slots++ {
			for seed := int64(1); seed <= 200; seed++ {
				g := &game{rng: newSimpleRNG(seed)}
				roster, err := g.stairGuardianRoster(depth, slots)
				if err != nil || len(roster) < min(2, slots) || len(roster) > slots {
					t.Fatalf("floor %d guardians slots=%d: %v", depth+1, slots, err)
				}
				for _, m := range roster {
					if m.Boss {
						t.Fatal("stair guardians must not include another Lich")
					}
				}
			}
		}
	}
}

func TestDifficultyAndRosterSurviveSavingWithoutReroll(t *testing.T) {
	g := newGameWithSaveDir(42, t.TempDir())
	g.stats.DifficultyDefeated = 75
	g.current().Monsters[0].HP = 1
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.levels, loaded.levels) || g.rng.State != loaded.rng.State || g.stats != loaded.stats {
		t.Fatal("loading must preserve budgets, roster, wounds, score, and RNG")
	}
}

func TestDifficultyScoreCountsEachDeathAndPersistsBestFinishedRun(t *testing.T) {
	g := openArena()
	g.stats = runStats{}
	m := newMonster(monsterHollowWarden)
	m.Pos = pos{6, 5}
	g.current().Monsters = []*monster{&m}
	want := monsterDifficulty(m)
	g.damageMonster(&m, m.MaxHP, "hits")
	g.damageMonster(&m, 99, "burns")
	if g.stats.MonstersKilled != 1 || g.stats.DifficultyDefeated != want {
		t.Fatal("each slain monster must award its original difficulty once")
	}
	g.recordsFile = filepath.Join(t.TempDir(), "records.json")
	if _, err := g.recordChallenges(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(g.recordsFile); !os.IsNotExist(err) {
		t.Fatal("unfinished runs must not submit a score")
	}
	g.player.HP = 0
	if _, err := g.recordChallenges(); err != nil {
		t.Fatal(err)
	}
	g.stats.MonstersKilled = 2
	g.stats.DifficultyDefeated = want - 1
	if _, err := g.recordChallenges(); err != nil {
		t.Fatal(err)
	}
	records, err := readChallengeRecords(g.recordsFile)
	if err != nil || records.MostDifficulty == nil || records.MostDifficulty.Difficulty != want || records.MostKills.Kills != 2 {
		t.Fatalf("weighted and raw kill records should keep their own best run: %+v err=%v", records, err)
	}
	var out bytes.Buffer
	g.renderChallenges(&out)
	if !strings.Contains(out.String(), fmt.Sprintf("Difficulty defeated: %d points.", g.stats.DifficultyDefeated)) || !strings.Contains(out.String(), "Most difficulty defeated record:") {
		t.Fatal("challenge screen should show the run's score and saved record")
	}
}
