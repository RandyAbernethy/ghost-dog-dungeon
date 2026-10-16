# Gameplay contracts

These specialized rules supplement [AGENTS.md](../AGENTS.md). Read the relevant sections before changing these
systems. Shared rules for RNG, turn costs, rendering, saves, companions, and death accounting remain in AGENTS.md.

## Generation and secret rooms

- Bound generation retries and provide a connected fallback. Reserve features before placing competing actors or items.
- Keep ordinary terrain connected, stairs and rewards reachable, unrelated secret rooms separate, and undiscovered
  chambers concealed.
- Generate 5–8 secret chambers per new game, counting linked chambers and the final escape sanctum. Ordinary floors
  may have zero, one, two, or rarely three chambers. Linked chambers are occasional and have no fixed floor.
- Secret chambers must never enclose another chamber. Link disjoint rooms through a hidden door reachable only
  through the preceding chamber, with separate discovery after its parent is opened.
- Every secret chamber has its own fountain and side-story scroll. The Book of Ghost Dog shows only collected stories.

Preserve the seed sweeps and assertions in [generation_test.go](../src/generation_test.go),
[secrets_test.go](../src/secrets_test.go), [balance_test.go](../src/balance_test.go), and
[replay_test.go](../src/replay_test.go). They cover connectivity, feature overlap, hidden-room access, room-count
distribution, geometric variety, rewards, and seed reproduction.

## Ordinary balance and placement

- Ordinary monster counts vary within calibrated floor difficulty budgets. Keep ordinary budgets and encounter
  totals strictly increasing with depth, and cap ranged attackers, disabling powers, and duplicate species.
- Preserve the calibrated health, damage, and difficulty bounds in [difficulty.go](../src/difficulty.go) and the
  averages and tolerances asserted in [difficulty_test.go](../src/difficulty_test.go), unless the request changes balance.
- Ordinary equipment variety must preserve average damage, reach, and armor trait averages, as asserted in
  [equipment_variety_test.go](../src/equipment_variety_test.go).
- New monster types and equipment may be absent from a run. Keep entry tiles clear, avoid crowding newly arrived
  players, and reserve one distant guard position for the Dread Lich.

Keep existing seed coverage for roster budgets, population variety, and placement. Use the calibrated constants and
test assertions as the balance baseline rather than an unspecified previous version.

## Relics and Epic encounters

- God armor and Artifact weapons are rare exceptions to ordinary loot balance. Generate room relics only in later
  chambers of secret-room chains on floors 7–9, with a 25% chance per eligible chamber and at most one room relic
  per dungeon. Replace ordinary treasure so fountains and side-story scrolls still fit. Never roll relics on discovery,
  revisiting, or loading.
- Optional Epic monsters occur only on floors 7–9, with at most one per dungeon, outside ordinary difficulty budgets
  and guardian pools. Their initial position must be bypassable and distant from stairs; they act once every three turns.
- Gear powers require equipped pieces and must not revive Ghost Dog. God protection is powerful but not invulnerability.
- Defeated Epics guarantee one additional elite treasure at their final position, independently of linked rooms:
  God armor, an Artifact weapon, a Starfire Orb, a Sun Orb, Phoenix Ash, or Ghost Dog Recall. Avoid duplicate equipment.
  The shared RNG and death-accounting rules apply to every kill source. Save/load preserves the dropped or collected
  reward without rerolling or recreating it.

Preserve placement, rarity, equipment-power, kill-source, and persistence coverage in
[relics_test.go](../src/relics_test.go), [epic_monsters_test.go](../src/epic_monsters_test.go), and
[epic_loot_test.go](../src/epic_loot_test.go).

## Events and movement effects

- Fountains refill on leaving their floor. Event gifts and shrine blessings are one-use and stay used after revisiting
  or loading.
- Blink Stones exclude all secret chambers, including discovered and additional rooms, and move active companions together.
- Track whether a movement hold comes from webs, ice, silver chains, or rift magic; the log and status display must
  name the actual effect. Preserve the one-movement-turn cost and Oracle Crown immunity, including save/load and equipment.

Use [events_test.go](../src/events_test.go), [ghost_gifts_test.go](../src/ghost_gifts_test.go),
[prompts_test.go](../src/prompts_test.go), [blink_test.go](../src/blink_test.go), and
[conditions_test.go](../src/conditions_test.go) for event, arrival, movement, and persistence regressions. Interactive
gift offers live in [ghost_gifts.go](../src/ghost_gifts.go).
