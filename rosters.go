package main

import "fmt"

type monsterCandidate struct {
	Spec   monster
	Weight int
	Local  bool
}

func encounterPool(depth int, theme floorTheme) []monsterCandidate {
	var pool []monsterCandidate
	add := func(spec monster, local bool) {
		if spec.Boss {
			return
		}
		for i := range pool {
			if pool[i].Spec.Kind == spec.Kind {
				if local {
					pool[i].Weight++
					pool[i].Local = true
				}
				return
			}
		}
		weight := 2
		if _, extra := additionalMonsterSpec(spec.Kind); extra {
			weight = 1
		}
		if spec.Kind == newMonsterKindsByFloor[depth] {
			weight += 2
		}
		if !local {
			weight = 1
		}
		if themeFavors(theme, spec) {
			weight *= 2
		}
		pool = append(pool, monsterCandidate{scaleMonsterForDepth(spec, depth), weight, local})
	}
	for _, spec := range monsterPoolForLevel(depth) {
		add(spec, true)
	}
	if depth > 0 {
		for _, spec := range monsterPoolForLevel(max(0, depth-2)) {
			add(spec, false)
		}
	}
	return pool
}

func (g *game) weightedMonster(pool []monsterCandidate) monsterCandidate {
	total := 0
	for _, c := range pool {
		total += c.Weight
	}
	roll := g.rng.Intn(total)
	for _, c := range pool {
		if roll < c.Weight {
			return c
		}
		roll -= c.Weight
	}
	return pool[len(pool)-1]
}

func (g *game) monstersForLevel(lvl *level) ([]monster, error) {
	depth := lvl.Index
	low, high := floorDifficultyBounds(depth)
	lvl.DifficultyBudget = low + g.rng.Intn(high-low+1)
	var reserved []monster
	if depth == levelCount-1 {
		reserved = []monster{scaleMonsterForDepth(newMonster(monsterBoss), depth)}
	}
	baseCount := depth + 1
	minCount := max(1, baseCount-max(1, baseCount/4))
	maxCount := baseCount + max(1, depth/3+1)
	if depth == 0 {
		maxCount = 2
	}
	roster, err := g.rosterForBudget(encounterPool(depth, lvl.Theme), reserved, lvl.DifficultyBudget,
		low, high, minCount, maxCount, baseCount, func(ms []monster) bool { return fairFloorRoster(ms, depth) })
	if err != nil {
		return nil, fmt.Errorf("floor %d encounter: %w", depth+1, err)
	}
	lvl.InitialDifficulty = rosterDifficulty(roster)
	return roster, nil
}

func (g *game) rosterForBudget(pool []monsterCandidate, reserved []monster, target, low, high, minCount, maxCount, baseCount int, fair func([]monster) bool) ([]monster, error) {
	var best []monster
	bestError := high + target
	counts := []int{baseCount, baseCount, baseCount, baseCount, baseCount, baseCount,
		baseCount - 1, baseCount - 1, baseCount + 1, baseCount + 1, baseCount - 2, baseCount + 2}
	for attempt := 0; attempt < 256; attempt++ {
		count := max(minCount, min(maxCount, counts[g.rng.Intn(len(counts))]))
		roster := append([]monster(nil), reserved...)
		copies := map[monsterKind]int{}
		borrowed := 0
		pickPool := pool
		if count > baseCount {
			average := (target - rosterDifficulty(reserved)) / max(1, count-len(reserved))
			var smaller []monsterCandidate
			for _, c := range pool {
				if monsterDifficulty(c.Spec) <= average*5/4 {
					smaller = append(smaller, c)
				}
			}
			if len(smaller) > 0 {
				pickPool = smaller
			}
		}
		for len(roster) < count {
			c := g.weightedMonster(pickPool)
			copies[c.Spec.Kind]++
			if !c.Local {
				borrowed += monsterDifficulty(c.Spec)
			}
			roster = append(roster, c.Spec)
		}
		total := rosterDifficulty(roster)
		valid := total >= low && total <= high && borrowed <= target/4
		for _, n := range copies {
			if n > 3 {
				valid = false
			}
		}
		if !valid || !fair(roster) {
			continue
		}
		error := abs(target - total)
		if error < bestError {
			best, bestError = roster, error
		}
		if error <= max(2, target/50) {
			return roster, nil
		}
	}
	if best != nil {
		return best, nil
	}
	return fallbackRoster(pool, reserved, target, low, high, minCount, maxCount, fair)
}

// A bounded knapsack supplies a roster if random sampling misses a usable combination.
// Fallback picks melee creatures without disabling powers, with at most three of a kind.
func fallbackRoster(pool []monsterCandidate, reserved []monster, target, low, high, minCount, maxCount int, fair func([]monster) bool) ([]monster, error) {
	type key struct{ Count, Points int }
	start := key{len(reserved), rosterDifficulty(reserved)}
	paths := map[key][]monster{start: append([]monster{}, reserved...)}
	borrowedKinds := map[monsterKind]bool{}
	for _, c := range pool {
		borrowedKinds[c.Spec.Kind] = !c.Local
	}
	for _, c := range pool {
		if rangedMonster(c.Spec.Kind) || controllingMonster(c.Spec.Kind) {
			continue
		}
		next := make(map[key][]monster, len(paths))
		// Explicit numeric order prevents map iteration from choosing the fallback.
		for count := start.Count; count <= maxCount; count++ {
			for points := start.Points; points <= high; points++ {
				k := key{count, points}
				roster, ok := paths[k]
				if !ok {
					continue
				}
				for n := 0; n <= 3 && count+n <= maxCount; n++ {
					total := points + n*monsterDifficulty(c.Spec)
					if total > high {
						break
					}
					newKey := key{count + n, total}
					if _, exists := next[newKey]; exists {
						continue
					}
					added := append([]monster{}, roster...)
					for i := 0; i < n; i++ {
						added = append(added, c.Spec)
					}
					borrowed := 0
					for _, m := range added {
						if borrowedKinds[m.Kind] {
							borrowed += monsterDifficulty(m)
						}
					}
					if borrowed > target/4 {
						continue
					}
					next[newKey] = added
				}
			}
		}
		paths = next
	}
	var best []monster
	bestError := high + target
	for count := minCount; count <= maxCount; count++ {
		for points := low; points <= high; points++ {
			if roster, ok := paths[key{count, points}]; ok && fair(roster) && abs(points-target) < bestError {
				best, bestError = roster, abs(points-target)
			}
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no roster fits difficulty range %d-%d", low, high)
	}
	return best, nil
}

func (g *game) stairGuardianRoster(depth, slots int) ([]monster, error) {
	maxCount := min(3, slots)
	minCount := min(2, maxCount)
	mean := floorDifficultyBaselines[depth] / (depth + 1)
	base := mean * maxCount
	if maxCount == 3 {
		base = mean * 5 / 2
	}
	radius := max(3, base/10)
	target := base - radius + g.rng.Intn(radius*2+1)
	return g.rosterForBudget(encounterPool(depth, ""), nil, target, base-radius, base+radius,
		minCount, maxCount, minCount, func(roster []monster) bool {
			ranged, controls := 0, 0
			for _, m := range roster {
				if rangedMonster(m.Kind) {
					ranged++
				}
				if controllingMonster(m.Kind) {
					controls++
				}
			}
			return ranged <= 1 && controls <= 1
		})
}
