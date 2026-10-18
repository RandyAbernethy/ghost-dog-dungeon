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
- During the last rest turn, an Epic within two tiles may prepare its named signature. Store the exact warning tiles
  without drawing RNG; on its next active turn, strike those tiles instead of moving or making a normal attack.
  Never retarget after the warning. Walls block its line, and every threatened actor must initially have an open
  neighboring escape tile. Signature damage uses the monster's saved damage range; magical signatures respect spell
  wards. Existing melee powers remain tied to melee hits. Ice Bloom and Eclipse apply existing binding/hex immunity.
- Map and inspect show the pending attack, including when the player or Ash occupies a warning tile. Reading,
  rendering, failed actions and saves cannot advance it. Freezing/stunning delays it; fleeing or death clears it.
  Ash spends his action dodging if threatened, avoids marked movement destinations, and never dodges while inactive.
  Blink excludes warning tiles for both companions. Pending warnings survive save/load exactly.
- Gear powers require equipped pieces and must not revive Ghost Dog. God protection is powerful but not invulnerability.
- Defeated Epics guarantee one additional elite treasure at their final position, independently of linked rooms:
  God armor, an Artifact weapon, a Starfire Orb, a Sun Orb, Phoenix Ash, or Ghost Dog Recall. Avoid duplicate equipment.
  The shared RNG and death-accounting rules apply to every kill source. Save/load preserves the dropped or collected
  reward without rerolling or recreating it.

Preserve placement, rarity, equipment-power, kill-source, and persistence coverage in
[relics_test.go](../src/relics_test.go), [epic_monsters_test.go](../src/epic_monsters_test.go), and
[epic_loot_test.go](../src/epic_loot_test.go).
Signature cadence, dodge space, cancellation, companion movement, and persistence are covered in
[epic_strikes_test.go](../src/epic_strikes_test.go) and [adventure_cli_test.go](../src/adventure_cli_test.go).

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

## Optional objectives and adventure narrative

- Generate at most one optional objective in a new dungeon: one 35% roll, then a suitable existing 3-by-3 open area on
  floors 3–8. Keep the whole area clear of features, actors, secret footprints and special rooms; do not carve terrain,
  add enemies or change ordinary budgets. Choose its kind and consumable reward through the saved RNG during generation.
  Loading never adds objectives. Preserve reproducibility and the rarity/placement sweep in
  [objectives_test.go](../src/objectives_test.go).
- Starting the spirit vigil spends one turn; the following three successful turns provide protection after enemies
  act. Player or freed, living Ash must remain within three tiles on that floor. Adjacent enemies remove one resolve
  per turn from its initial three. Losing all resolve, abandoning the area, or leaving the floor permanently fails it.
  Death or escape cannot advance a vigil. Breaking each distinct ritual sigil requires standing on it and spending
  one interaction turn; broken sigils, partial progress and terminal states persist.
- Completing an objective awards its saved consumable directly once through ordinary collection. No interaction,
  revisit, rendering or loading can reroll or duplicate it. Reading rules and interacting with spent markers are free.
  Arrival prompts follow existing proximity boundaries. Record successful completion in the adventure state.
- Ash's personality moments require him freed and alive, are one-time by ID, and change only narrative state and the
  current turn's log. Ambient moments require no living enemy within six tiles of either companion, at most one moment
  on that floor, and a fifteen-turn cooldown. Contextual rescue, fountain, keepsake and revival moments remain tied to
  those actions. Use no extra RNG or turns; no moment revives or buffs him.
- Escape callbacks use only collected side-story IDs, in catalog order, and reflect Ash's final survival. The finished
  run chronicle uses visited/revealed terrain, collected stories, recorded rescue floor, Epic deaths, objective successes,
  final equipment and existing fall statistics. Only death or escape shows it, before final statistics and celebrations.
  Rendering never records new history. Missing history in older format-2 saves stays absent; new fields are additive.

Keep narrative, older-save, free-action, one-time reward and executable coverage in
[adventure_test.go](../src/adventure_test.go), [objectives_test.go](../src/objectives_test.go), and
[adventure_cli_test.go](../src/adventure_cli_test.go).

## Hidden crypt

- Every new dungeon receives one extra crypt, saved separately from the ten main levels. Generate it once with the
  saved RNG after ordinary generation. Select two distinct concealed wall doors on one main floor 4–7, away from
  secret-room footprints and existing features. Existing format-2 saves without a crypt stay unchanged on load.
- Searching beside the entrance reveals it and warns about all travel/magic/sight restrictions before commitment.
  Walking through spends one turn, permanently seals that entrance, clears active wards, and moves only a freed,
  living companion. Leaving the parent floor refills its fountains and abandons any started spirit vigil. The far
  exit is discovered by searching beside it, then stepping through returns to the other parent-floor door. Ordinary
  stairs, repeated searches, and suppressed Blink Stones cannot return the player early. No rewards or actors reset.
- Fill the map with two connected winding mazes and small rooms. A solid central divider has exactly one crossing,
  occupied by the Under Lich; even eight-direction movement cannot bypass him. He holds that tile under fear,
  attacks every other turn, and uses normal undead/guard, damage, freezing, companion and death rules. He is not the
  campaign boss: no escape unlock, chapter ten or automatic companion revival on his death.
- Include only undead, with the Under Lich included in the floor-eight total difficulty bounds. Use floor-eight
  equipment and magic pools, without duplicating chapters or adding relics/Epics. Offer equipment on the entrance
  side. Keep the crypt free of fountains, ordinary stairways, events, optional objectives and secret rooms.
- Crypt sight has a Chebyshev radius of three and stops at walls. Render unseen tiles as spaces. Inspection, loot
  lists, codex discovery, and targeted consumables obey sight; Sun Orbs keep their existing floor-wide effect.
  Unseen ordinary monsters can approach, but cannot cast or attack through walls. Reads remain free of gameplay
  mutations apart from existing codex bookkeeping.
- A Warding Charm or Blink Stone with available stock is consumed and spends one action turn with no magic effect
  or extra RNG. Active wards protect neither companion inside the crypt. Passive equipment traits still work.
- Generate 1–4 distinct traps from the eight-kind pool, with fixed saved damage, away from actors, loot, the entry
  and exit approaches, and the guardian. An adjacent search makes one independent 50% roll per undiscovered,
  unspent trap. Known or spent traps do not reroll. A discovered trap can be disarmed with one interaction turn;
  discovery alone does not make stepping on it safe. Only player movement triggers traps, once each, before the
  enemy turn. Repulsion never places the player on an occupied tile or triggers another trap recursively.
- Pit holds cost one attempted move and survive equipment changes; Oracle Crown does not negate a physical pit.
  Magical ice/hex effects retain their ordinary immunity rules. Gas causes three two-damage ticks including the
  trigger action, after enemies; it persists through the exit and saving, and only successful actions advance it.
  Trap effects can kill the player normally. The adventure chronicle records crypt entry, escape and the guardian's
  defeat from the saved state; leaving the crypt does not submit finished-run records.

Preserve connectivity, mandatory-crossing, difficulty, rarity, placement and reproduction sweeps in
[crypt_test.go](../src/crypt_test.go), action/discovery/immunity coverage in [traps_test.go](../src/traps_test.go),
and complete generated-maze traversal through the compiled executable and save/load in
[crypt_cli_test.go](../src/crypt_cli_test.go).

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
