# Timed dungeon experiment

The preserved helper is helper_test.go.txt. Keeping the template extension prevents it from being discovered by go test ./.... run.ps1 temporarily copies it into src, formats and vets it, compiles an isolated test executable and a normal game executable, then removes only that temporary copy. Both template and binaries remain here.

Run from PowerShell:

    & .\bin\dungeon-trials-20261016-rerun\run.ps1

Parameters: -Seconds 60 -SamplingSeed 20261016. -PrepareOnly builds without starting trials.

The monotonic trial stopwatch starts inside TestDungeonTrials immediately before the first seed draw and dungeon generation, after preparation and compilation. It includes every seed draw, new dungeon, policy decision, legal command, and attempt-log write. All retries share one deadline. Artifacts are finalized after stopping the trial stopwatch. A deadline can be exceeded slightly by a single noninterruptible game action or generation operation; the exact elapsed time is reported. A success must occur before the deadline.

The same heuristic policy as the previous experiment selects healing, protection, attacks, equipment upgrades, nearby events and shortest paths to loot, Ghost Dog, stairs and the Lich. It has complete access to current floor geometry, actor locations, loot and player state. It does not read future RNG state, alter gameplay state directly, rewind, learn between attempts, call an LLM, or estimate win probabilities. It does not seek ordinary hidden rooms. These are policy limitations rather than findings about game balance.

Seeds are sampled without replacement from a reproducible independent math/rand stream, skipping zero and the previously scripted seed 6. Gameplay still uses the unmodified game's own saved RNG. This is Monte Carlo evaluation across sampled dungeons using a deterministic rule policy, not Monte Carlo tree search.

Each run directory contains attempts.jsonl and summary.json. The histogram buckets count deepest floor reached, including any partial deadline attempt, which is also reported separately. Stalled and command-limit trials are policy failures. The game, production files, real saves and player records are not modified. A winner additionally produces winning-commands.txt and a normal save snapshot.

