# Gameplay contracts

This document defines intended gameplay invariants and links them to source and regression tests. Player-facing
usage and mechanics are in the [README](../README.md); repository workflow and shared safeguards are in
[AGENTS.md](../AGENTS.md). Read the relevant sections before changing these systems.

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

- God armor and Artifact weapons are rare exceptions to ordinary loot balance. Generate room relics only in chambers
  after the first in secret-room chains on floors 7–9, with at most one room relic per dungeon. While no room relic
  exists, make one 25% roll per eligible floor, then select a replaceable treasure position among its later chambers;
  additional chambers do not add rolls. Replace ordinary treasure so fountains and side-story scrolls still fit.
  Never roll relics on discovery, revisiting, or loading. See [addLinkedRelic in relics.go](../src/relics.go).
  The boundary cases in [relics_test.go](../src/relics_test.go) check the 25% threshold and prevent extra rolls for
  additional chambers independently of the large rarity sweep.
- Epic monsters occur only on floors 7–9, outside ordinary difficulty budgets and guardian pools. Every floor with a
  generated God armor or Artifact room relic gets one Epic, bypassing the random chance and an Epic on another floor.
  Reserve its position before ordinary actors and features, and place stairs at least six tiles away. Encounters must
  preserve that reserved position and remain bypassable and at least eight tiles from the entrance.
- Without a room relic, make one 12% roll per eligible floor only while no Epic exists anywhere in the dungeon.
  Keep at most one Epic per floor; with one room relic per dungeon, there can be at most two Epics overall. Existing
  encounters, including dead ones, prevent duplicates. Epic relic drops already come from an encounter on their floor
  and must not spawn replacement Epics. Discovery, revisiting, dropping carried gear, and loading do not roll encounters.
- The eight Epic possibilities are Iron Colossus, Grave Titan, Ashen Wyrm, Hollow Reaper, Frost Monarch, Storm Leviathan,
  Dusk Tyrant, and Obsidian Golem. All wake within four tiles and act once every three turns. New ice binding and hexes
  use the existing one-move/next-strike effects and Oracle Crown immunity. Lightning affects only the player and a freed,
  living Ghost Dog within two tiles, only after a damaging melee strike; player spell wards apply. See
  [epic_monsters.go](../src/epic_monsters.go).
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

## Personal records

- Only escape or death submits records. Fewest turns requires an escape; Ghost Dog safe escape also requires him
  freed, alive, and never fallen. Both speed records prefer fewer turns. Kills and defeated difficulty prefer higher
  totals, including on death.
- Compare finished results with the previous records before writing improvements. First qualifying results and strict
  improvements earn a prominent **NEW RECORD!** announcement; equal best scores earn **RECORD TIED!**. Include every
  qualifying category and its score after the final statistics. Ties preserve the original record holder and do not
  rewrite the records file. Suppress the celebration if record storage fails.
- Reading records, saving, and quitting an unfinished run must not submit or celebrate records. Celebrations do not
  change gameplay state, turns, RNG, or the save format.

Keep record comparisons, eligibility, ties, and storage-error coverage in [records_test.go](../src/records_test.go),
and compiled-game save/load, victory, and death coverage in [replay_test.go](../src/replay_test.go).

## Campaign replay

[campaign_test.go](../tests/campaign_test.go) runs the compiled game from a fresh seed using the ordinary commands in
[seed-6.txt](../tests/testdata/seed-6.txt). It checks all ten floor checkpoints, Ghost Dog's rescue, equipment and story
collection, the Lich's death, the hidden escape, kill accounting, and final records. Uninterrupted play and save/load
continuation must preserve identical gameplay state. The route completes the dungeon with some optional content left
behind; it does not require collecting every story. This executable replay and the large seed sweeps run in the full
suite; fast and full commands are documented in [Build](../README.md#build).
