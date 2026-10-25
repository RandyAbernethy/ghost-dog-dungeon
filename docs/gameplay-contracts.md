# Gameplay contracts

> **Spoiler warning:** These contracts reveal hidden content, generation rules, and ending behavior.

This document defines intended gameplay invariants and links them to source and regression tests. Setup, controls,
and general play are in the [README](../README.md); detailed player-facing mechanics are in the
[gameplay guide (spoilers)](gameplay-guide.md). Repository workflow and shared safeguards are in
[AGENTS.md](../AGENTS.md). Read the relevant sections before changing these systems.

## Opening screen

- Show the ASCII puppy portrait once per game launch, after the version and dungeon seed and before personal records
  and the initial map. Show it for both new and loaded games, not on subsequent map redraws or in `--help`/`--version`.
- Embed the reusable [puppy.txt](../src/assets/puppy.txt) asset in the executable; startup must not read artwork from
  the working directory. Use printable ASCII and line feeds so the art works in both raw-key and command-and-Enter mode.
- On interactive terminals, show version, seed, artwork and personal records before the prompt
  "Press enter to begin the descent! >". Wait for Enter, then clear the welcome screen and render gameplay.
  Share the existing input reader, preserving subsequent input. Ctrl-C/Ctrl-D or EOF exits without gameplay.
  Redirected scripts skip the pause and screen clearing. Opening and confirmation spend no turn, draw no RNG,
  and do not change saved game state.
- New games start with the chapel-yard cellar introduction: the wind through the bell tower evokes a lonely hound,
  and the cellar door slams shut before the player explores the catacombs. Keep this in the initial action log, free
  of turns and extra RNG. Render each paragraph at word boundaries within 80 columns, including the bullet prefix;
  indent continuation lines by two spaces. Keep stored paragraphs intact. Loading retains the saved log instead of
  inserting a new introduction.

Preserve portrait and startup coverage in [opening_test.go](../src/opening_test.go), executable new-game/load checks in
[cli_test.go](../tests/cli_test.go), and raw-key coverage in [terminal_linux_test.go](../src/terminal_linux_test.go).

## Save format

- Favor clean, maintainable code over backward compatibility. Break save compatibility only when doing so makes
  the codebase clearer or easier to work on; avoid unnecessary breaks and compatibility debt.
- Keep explicit save-format versioning and bump the version for incompatible schema changes. The game supports
  only the current format, currently format 4. Reject older or unversioned saves and unknown future versions with
  a useful error asking the player to start a new game. Do not migrate, repair, or overwrite rejected snapshots.
- Do not retain obsolete fields, compatibility fallbacks, or legacy-format loading code in the game. If support for
  older saves becomes necessary, build a separate savefile updater that converts them to the current format before
  loading. The updater is future work; no implementation or scaffolding is needed now.
- Restore the complete current-format dungeon, player, companion, crypt, rewards, challenge progress and RNG exactly.
  Carried equipment is initialized for a new game and maintained by pickup, equip and drop actions; reading screens
  and loading must not rebuild it. Runtime save-directory and records-file settings remain separate.
- Keep personal records across save-format changes; records have their own file and schema.

Preserve snapshot, rejection and command coverage in [save_test.go](../src/save_test.go), startup rejection in
[cli_test.go](../tests/cli_test.go), and deterministic continuation in [campaign_test.go](../tests/campaign_test.go).

## Inventory and compact presentation

- Each carried consumable in the inventory list displays its direct use command beside its count and map symbol.

- Inventory and equipment open the same transient menu. Arrows and Enter select in raw mode; numbered selection works
  in both modes. From the list, `e` equips and `d` drops the highlighted item without opening details.
  Enter still opens details, which show descriptions, equipment comparisons and powers before Use, Equip or Drop.
- Equipment pickup messages show the item name, armor slot where applicable, and stat differences from equipped gear.
  Start the comparison sentence on its own line with a two-space indent, wrapping at 80 visible columns.
  Preserve the stored message and apply this layout during rendering.
  Omit repeated absolute attributes, power descriptions and inventory-key reminders; full details remain in inventory.
- Each armor entry in the inventory list shows its guard, spell ward and strike damage, including zero values,
  without requiring selection of its details.
- Use the same stable equipment/consumable numbering as advanced commands, even when zero-stock consumables are hidden.
  Confirmed actions go through `processCommand`; selecting, reading, cancelling and invalid selections remain free.
  Confirmed actions close the menu; menus are never serialized or restored by loading.
- Keep the map's survival summary short: player and companion health, weapon, protection and current effects. Show only
  nonzero equipment spell ward and strike bonuses in the map header; inventory retains zero-valued statistics. Show only
  visible entities within four tiles in the contextual legend. Inventory and records retain the full counts and statistics; help retains
  a complete feature/item reference and the discovered codex provides enemy symbols. Prompts describe the actual input mode.
- Include the original, unrescued Ghost Dog chain location as `d chained Ghost Dog` in the nearby legend when visible
  within four tiles. Stop listing that original location after rescue; keep the existing companion legend behavior.
- Wrap status, menu, objective, hint and warning prose to the terminal width (up to 80 columns). Windows
  and redirected streams fall back to 60; `COLUMNS` overrides it, with a minimum of the 38-column map. Never truncate
  the complete turn log. Gameplay log prose wraps between words at 80 visible columns with a two-character bullet
  or continuation indent; ignore ANSI colors when counting columns. Preserve full snapshot-path announcements on
  one line for copying. Store complete original messages and wrap only during rendering, preserving order and RNG.
  Size queries belong in the OS terminal files and consume no gameplay randomness.

Preserve command-loop, compiled save/load, cancellation, width and raw-menu coverage in
[inventory_menu_test.go](../src/inventory_menu_test.go) and Linux PTY coverage in
[terminal_linux_test.go](../src/terminal_linux_test.go).

## Map symbols

- Give each monster species, physical item category, trap kind, and other feature its own printable ASCII symbol.
  Fire, Ghost Dog recall, chapter and side-story scrolls share `?`; Sun and Starfire orbs share `=`; Warding and Frost
  Charms share `%`; healing potions and Phoenix Ash share `!`. Weapons use `)`, armor `[`, and stones `*`.
  No different physical categories may share a symbol.
- Floor legends and inspection show only physical categories (scroll, orb, charm, potion, weapon, armor, stone), never
  exact names, rarity, statistics, effects, chapter numbers, story text or titles. Monster-drop announcements also
  conceal these identities. Pickup identifies the exact item; carried inventory and explicit friendly-ghost offers
  retain full details. Dropped items again use category-only floor descriptions; no identification memory or new save
  fields are needed. Loading and inspection reveal nothing
  extra and cannot change gameplay state or RNG. Help lists each physical symbol once.
- Reserve `F` and `f` for fresh and spent fountains, `D`/`d` for Ghost Dog, and `X` for Epic attack warnings.
  The eight Epics use `1`–`8`; Frost Monarch uses `5`. Shrines retain `^`; each discovered trap has its own symbol.
- Maintain the shared catalog in [glyphs.go](../src/glyphs.go). Constructors, map, codex, inventory, inspection,
  and legends must agree. Derive symbols by entity kind instead of serializing duplicate glyph fields.
  Displaying current symbols must not spend turns, draw RNG, or alter the dungeon or rewards.

The complete player reference is in [Map symbols](gameplay-guide.md#map-symbols). Preserve category uniqueness,
rendering, trap visibility, and compiled-executable save/load coverage in [glyphs_test.go](../src/glyphs_test.go).

## Generation and secret rooms

- Bound generation retries and provide a connected fallback. Reserve features before placing competing actors or items.
- Keep ordinary terrain connected, stairs and rewards reachable, unrelated secret rooms separate, and undiscovered
  chambers concealed.
- Generate 5–8 secret chambers per new game, counting linked chambers and the final escape sanctum. Ordinary floors
  may have zero, one, two, or rarely three chambers. Linked chambers are occasional and have no fixed floor.
- Secret chambers must never enclose another chamber. Link disjoint rooms through a hidden door reachable only
  through the preceding chamber, with separate discovery after its parent is opened.
- Every secret chamber has its own fountain and side-story scroll. The Book of Ghost Dog shows only collected stories.
- Derive candidate passage clues from existing door geometry. A clue requires an open approach within three tiles;
  intervening walls and unopened parent chambers block it. Successful movement or area arrival with eligible clues
  has one 20% chance (roll below 20 out of 100) to add one subtle directional hint to the current turn log. Hints must
  not instruct the player to search. Do not reveal hints automatically in map context text or inspection, including
  beside the crypt entrance or exit. Help retains the search control; search behavior itself is unchanged.
- Use a copy of the saved RNG salted with the saved turn for narrative hint rolls, preserving combat RNG and seeded
  campaign outcomes. Pick at most one candidate after a successful chance roll. Reading, redraws, failed actions, waiting,
  saving and loading cannot generate or retry hints. The saved current-turn log retains any hint already offered; no
  new save fields are needed. Killing the Lich opens the mandatory escape sanctum automatically.

Preserve the seed sweeps and assertions in [generation_test.go](../src/generation_test.go),
[secrets_test.go](../src/secrets_test.go), [balance_test.go](../src/balance_test.go), and
[replay_test.go](../src/replay_test.go). They cover connectivity, feature overlap, hidden-room access, room-count
distribution, geometric variety, rewards, and seed reproduction.
Clue accessibility, parent ordering, wall occlusion and persistence are covered in
[passage_clues_test.go](../src/passage_clues_test.go).

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

## Dread Lich and companion chains

- The Dread Lich has 60 base HP and 7–11 base damage: the normal depth scaling produces 78 HP and 10–14 damage on
  floor ten. Keep the existing ordinary-floor budgets, roster bounds, death accounting and escape rewards.
- On each active Lich turn, make one saved-RNG roll below 10 out of 100 when Ghost Dog is freed, alive, unchained,
  within five tiles, and visible to the Lich through clear terrain. Successful chaining replaces the Lich's attack;
  failed rolls allow his normal action. Dead, stunned, frozen or fleeing Liches and ineligible targets do not roll.
  Reading, failed actions, saving and rendering cannot give the Lich another turn.
- Store temporary chains separately from the historical `Freed` flag, with the main-floor anchor and dog position.
  While chained, Ash cannot attack, dodge, guard objectives, follow stairs, or blink with the player. Display `d`
  rather than `D`, and explain the release action in the log, status, inventory, inspection and nearby hint.
- Walking onto chained Ash takes one ordinary movement turn, breaks the chain, and lets him act again. Prioritize
  rescue over a reach attack past him. Preserve his health and original rescue history; avoid overlapping actors.
  Only present companions can be targeted or receive passive healing. Ash stays anchored when the player leaves;
  matching coordinates on another floor, blink and recall cannot release or move him.
- Living Ash remains chained when the Lich dies, until the player reaches him. Dispersion clears the physical chain
  and counts a fall once; existing explicit revival rules still apply. Leaving Ash chained behind is an escape alone
  in victory text, ending memories, the chronicle, and safe-companion record eligibility.
- Saves persist chains and their anchor exactly. Do not repair, reroll, revive or rebalance anything on load.
  Reject invalid chained-dog anchors with useful errors.

Preserve the chance-boundary sweep, rescue, inactivity, travel, targeting, records and executable save/load checks in
[lich_test.go](../src/lich_test.go), the generated boss-stat assertions and balance sweeps in
[difficulty_test.go](../src/difficulty_test.go), and existing companion combat and campaign replay coverage.

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
  Dusk Tyrant, and Obsidian Golem. All wake within four tiles and normally act once every three turns. Alternate normal
  actions (cadence 1, 7, 13...) and signature actions (4, 10, 16...), warning signatures on 3, 9, 15... . Normal attacks
  reach two tiles through clear terrain and use half the saved signature damage bounds, rounded down with a minimum of
  one, before ordinary defenses. Adjacent attacks retain melee powers; distance-two pressure has no melee power.
  Iron Colossus/Obsidian Golem ranged pressure is physical; other ranged pressure respects player spell wards.
  Out-of-range normal actions pursue rather than hitting. Existing ice binding and hexes
  use the existing one-move/next-strike effects and Oracle Crown immunity. Lightning affects only the player and a freed,
  living Ghost Dog within two tiles, only after a damaging melee strike; player spell wards apply. See
  [epic_monsters.go](../src/epic_monsters.go).
- During the last rest turn before a signature action, an Epic within two tiles may prepare its named signature. Store the exact warning tiles
  without drawing RNG; on its next active turn, strike those tiles instead of moving or making a normal attack.
  Never retarget after the warning. Walls block its line, and every threatened actor must initially have an open
  neighboring escape tile. Signature damage uses the monster's saved damage range; magical signatures respect spell
  wards. Existing melee powers remain tied to melee hits. Ice Bloom and Eclipse apply existing binding/hex immunity.
- Every separate damaging hit on a surviving Epic adds one rage, capped at four, through the common damage handler.
  Player, companion, consumables and passive damage all count; fully blocked hits and corpses do not. At full rage,
  the next eligible enemy turn with no pending warning may prepare a retaliation, replacing that turn's normal cadence.
  Retaliation marks a fixed 3-by-3 area around each companion within four tiles and clear line of sight, excludes walls
  and the Epic's own tile, and gives two complete player actions before resolving the ordinary signature damage/effects.
  Preparation resets rage; later hits can build the next meter. Rage does not decay or reroll on travel or loading.
- Require a legal two-move escape from the combined warning tiles for each threatened actor before preparing retaliation.
  Account for occupied tiles, player movement holds and reach attacks that would replace movement. A chained Ash cannot
  escape and prevents a retaliation covering him. If no fair pattern is available, retain rage and follow normal cadence.
  Freezing/stunning pauses the retaliation countdown; fear cancels it; death clears warnings, rage and countdown.
  Ash starts a two-step retreat immediately when necessary. Retaliation replaces other attacks while pending and
  resumes the saved normal cadence afterward. Save format 4 persists rage, countdown and exact tiles without renewal.
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
Normal pressure, rage sources, lethal mistakes, two-action escapes, defenses and compiled continuation are covered in
[epic_pressure_test.go](../src/epic_pressure_test.go).
The deterministic arena policies in [epic_trials_test.go](../src/epic_trials_test.go) compare reckless, prepared ordinary
equipment, and relic-equipped combat over 16 seeds per Epic. They log wins, deaths, turns, damage and consumable use;
these are fixed-policy checks, not human win-rate estimates. Keep at least one ordinary-equipment prepared victory
and a lethal result from reckless play. Generation, ordinary budgets and equipment pools remain independently covered.

## Events and movement effects

- The gift-giving event is displayed as "ghost of a friendly villager" in prompts, help and map legends.

- Fountain arrival prompts say only "A healing fountain flows nearby. Press v to drink."
- Fountains refill on leaving their floor. Event gifts and shrine blessings are one-use and stay used after revisiting
  or loading.
- Ash's keepsakes restore 8 HP, capped at maximum health, to a freed, living Ghost Dog within two tiles of the keepsake
  and grant one warding charm. Activation spends one turn; saving or revisiting cannot renew either reward.
  The arrival prompt identifies only the faded collar and interaction key, without revealing rewards:
  "Ash's keepsake, a faded collar, lies nearby; press y (interact)."
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
  act. Player or freed, living, unchained Ash must remain within three tiles on that floor. Adjacent enemies remove one
  resolve per turn from its initial three. Losing all resolve, abandoning the area, or leaving the floor permanently fails it.
  Death or escape cannot advance a vigil. Breaking each distinct ritual sigil requires standing on it and spending
  one interaction turn; broken sigils, partial progress and terminal states persist.
- Completing an objective awards its saved consumable directly once through ordinary collection. No interaction,
  revisit, rendering or loading can reroll or duplicate it. Reading rules and interacting with spent markers are free.
  Arrival prompts follow existing proximity boundaries. Record successful completion in the adventure state.
- Ash's personality moments require him freed and alive, are one-time by ID, and change only narrative state and the
  current turn's log. Ambient moments require no living enemy within six tiles of either companion, at most one moment
  on that floor, and a fifteen-turn cooldown. Contextual rescue, fountain, keepsake and revival moments remain tied to
  those actions. Use no extra RNG or turns; no moment revives or buffs him.
- Escape prints the victory message and bowl, followed by at most one randomly selected reminiscence. Only collected
  side stories and a freed, living Ash's return after a recorded fall are eligible; dream/song variants reflect his
  presence at the escape. Keep every candidate on one output line. If none is eligible, print only the bowl.
  Select using a copy of the saved RNG so repeated rendering and save/load retain the same choice without advancing
  gameplay RNG, spending turns, or adding save fields. Keep Your Adventure, challenges, records, celebrations, and
  the final dungeon seed in place.
- The finished run chronicle uses visited/revealed terrain, collected stories, recorded rescue floor, Epic deaths,
  objective successes, final equipment and existing fall statistics. Only death or escape shows it, before final
  statistics and celebrations.
  Rendering never records new history.

Keep narrative, save/load, free-action, one-time reward and executable coverage in
[adventure_test.go](../src/adventure_test.go), [objectives_test.go](../src/objectives_test.go), and
[adventure_cli_test.go](../src/adventure_cli_test.go).

## Hidden crypt

- Every new dungeon receives one extra crypt, saved separately from the ten main levels. Generate it once with the
  saved RNG after ordinary generation. Select two distinct concealed wall doors on one main floor 4–7, away from
  secret-room footprints and existing features.
- Searching beside the entrance reveals it and warns that it seals behind the player. Fountain and magic
  restrictions are left for players to discover; no recurring crypt-rules banner is shown. Entry announces only
  three-tile sight and fading wards.
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
  or extra RNG. Each attempt explains that the power of the crypt consumes the item’s magic without effect. Active wards protect neither companion inside the crypt. Passive equipment traits still work.
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
  freed, alive, unchained with the player, and never fallen. Both speed records prefer fewer turns. Kills and defeated
  difficulty prefer higher totals, including on death.
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
