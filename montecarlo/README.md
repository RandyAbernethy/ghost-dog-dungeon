# Monte Carlo dungeon trials

Run the automated player from the repository root with Go 1.22+ and PowerShell:

```powershell
./montecarlo/run.ps1 -Seconds 60 -SamplingSeed 20261016
```

Preparation and compilation happen first. One monotonic timer then covers every seed draw, dungeon generation,
policy decision, game command and attempt-log write until an escape with a freed, living Ghost Dog or the deadline.
The runner stops at the first success. It does not restart the timer after a failure. Summary and winner artifacts
are finalized after stopping the trial timer. A single game action or generation operation can cross the deadline
slightly; the exact elapsed time is reported, and a success must occur before the deadline.

Use `-PrepareOnly` to format, vet, compile and check policy regressions without starting any trials. These checks
also run before each timed experiment. Normal `go test ./...` runs do not
discover this experiment. The runner uses a Go build overlay to compile the helper with private game state;
it never adds or changes a file in `src/`. The `.go.txt` extension keeps the template outside Go package discovery.
All experiment code and data live under `montecarlo/`:

- `helper_test.go.txt` and `run.ps1`: the reusable policy and runner.
- `data/`: dated results, reports, charts, and preserved helper snapshots; see the [data index](data/README.md).
- `bin/`: generated executables, ignored by Git.
- `tmp/`: build overlays, Go cache, and isolated temporary files, ignored by Git.

The repository's root `bin/` holds game binaries. Experiment results remain available to Git for review and preservation.

## What is being sampled

This is Monte Carlo evaluation of a fixed decision policy across sampled dungeons. A separate, reproducible
`math/rand` stream samples positive 63-bit seeds without replacement, excluding the previously scripted seed 6.
The unmodified game's saved RNG still controls all gameplay outcomes. Using the same sampling seed and policy
reproduces the sequence of dungeons and decisions; a wall-clock cutoff can change the number of attempts completed.

The helper has full access to the current floor's geometry, monsters, loot, player equipment and companion state.
It uses deterministic rules for healing, equipment, combat, nearby events and shortest-path movement.
Policy `rule-policy-v2-shrine-eligibility` corrects the previous policy's shrine rule: a shrine is useful when the
player is injured or a freed, living Ghost Dog is injured. A fallen or unfreed dog's missing health does not trigger
shrine use. The helper otherwise retains the previous decision rules. It does not estimate action win probabilities,
simulate alternative future actions, learn between attempts or call an LLM. It cannot inspect future RNG results,
change gameplay state directly or rewind failed attempts. It does not seek ordinary secret chambers.

The policy is deliberately simple: it collects reachable loot before descending, takes any available straight
melee or reach attack, equips improvements using fixed scores, and chooses the first friendly-ghost gift.
A poor result measures this policy on the sampled dungeons; it does not establish a human win rate or prove
the game is too difficult. This is not Monte Carlo tree search.

## Outputs

Each dated directory under `data/` preserves the exact helper and runner used, a manifest with Go version and production
source hashes, and:

- `attempts.jsonl`: seed, deepest floor, turns, companion state, termination reason and elapsed time for each attempt.
- `summary.json`: total elapsed time, failures, any interrupted attempt, histogram and winning attempt.
- On success, `winning-commands.txt` and a normal save snapshot.

Histogram buckets count the deepest floor reached by each started attempt, including an interrupted deadline
attempt. Completed failures and deadline interruptions are reported separately. Repeated commands that spend no
turns end a trial as `stalled`; a 2,500-command cap ends it as `command_limit`. These are player-policy failures.

The built test executable inherits the repository's isolated test home. Saves and records never use the player's
real home. The runner also builds `ghostdog-verification` for replaying a winner through the ordinary executable.

