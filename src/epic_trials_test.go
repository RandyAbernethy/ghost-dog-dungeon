package main

import "testing"

func directionCommand(from, to pos) string {
	for _, cmd := range []string{"w", "a", "s", "d", "q", "e", "z", "x"} {
		dx, dy, _ := moveDelta(cmd)
		if dx == sign(to.X-from.X) && dy == sign(to.Y-from.Y) {
			return cmd
		}
	}
	return "."
}

// These are fixed, deliberately simple arena policies, not human win rates.
// The sweep checks that normal equipment can win with resources and that
// ignoring warnings can lose; it records the tradeoffs for balance review.
func TestEpicCombatPolicyTrials(t *testing.T) {
	if testing.Short() {
		t.Skip("reproducible combat policy sweep")
	}
	type totals struct{ wins, deaths, turns, damage, items int }
	results := map[string]*totals{}
	policies := []string{"greedy", "careful", "relic"}
	for _, policy := range policies {
		result := &totals{}
		results[policy] = result
		for seed := int64(1); seed <= 16; seed++ {
			for _, kind := range epicMonsterKinds {
				g := openArena()
				g.rng = newSimpleRNG(seed)
				g.player.Weapon = makeWeaponLoot(weaponPoolForLevel(8)[0]).Weapon
				for _, a := range []armor{{Name: "Graveward Helm", Slot: slotHead, Defense: 4}, {Name: "Ruinplate Vest", Slot: slotBody, Defense: 5}, {Name: "Stormscale Greaves", Slot: slotLeg, Defense: 4}} {
					g.player.setArmor(a)
				}
				if policy == "relic" {
					g.player.Weapon = artifactWeapons[0]
					g.player.setArmor(godArmorPieces[0])
				}
				g.player.WardingCharms, g.player.FrostCharms = 2, 2
				m := newMonster(kind)
				m.Pos = pos{6, 5}
				g.current().Monsters = []*monster{&m}
				for action := 0; action < 300 && m.HP > 0 && g.player.HP > 0; action++ {
					cmd := directionCommand(g.player.Pos, m.Pos)
					if policy != "greedy" {
						switch {
						case g.epicDangerAt(g.player.Pos):
							if next, ok := g.epicEscapeStep(g.player.Pos, g.epicDangerAt); ok {
								cmd = directionCommand(g.player.Pos, next)
							} else if g.player.FrostCharms > 0 {
								cmd = "t"
							}
						case g.player.HP < 28 && g.player.Potions > 0:
							cmd = "p"
						case g.player.ShieldTurns == 0 && g.player.WardingCharms > 0:
							cmd = "b"
						case m.Rage == 3 && g.player.FrostCharms > 0:
							cmd = "t"
						}
					}
					before := g.player.HP
					if cmd == "p" || cmd == "b" || cmd == "t" {
						result.items++
					}
					g.processCommand(cmd)
					result.damage += max(0, before-g.player.HP)
				}
				result.turns += g.stats.Turns
				if m.HP == 0 && g.player.HP > 0 {
					result.wins++
				}
				if g.player.HP == 0 {
					result.deaths++
				}
			}
		}
		t.Logf("%s: %d/128 wins, %d deaths, %d turns, %d damage, %d items", policy, result.wins, result.deaths, result.turns, result.damage, result.items)
	}
	if results["greedy"].deaths == 0 || results["careful"].wins == 0 || results["relic"].wins == 0 {
		t.Fatal("Epics must punish greed while allowing prepared victories, including ordinary equipment")
	}
}
