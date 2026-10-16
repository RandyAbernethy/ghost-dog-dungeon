# Recorded experiments

These files preserve the experiments and diagnostics performed while developing the helper:

- [First timed run](independent-run-20261016-115625/report.md): the original one-minute budget included preparation.
- [Full-minute trial run](run-20261016-120520926/report.md): policy v1 ran for 60 seconds after preparation, with 3,905
  failed attempts and one interrupted attempt. The directory includes raw logs, a histogram, and source hashes.
- [Stall diagnosis](stall-diagnosis/diagnostics.json): all five stalled v1 runs repeatedly attempted an unusable shrine.
- [Shrine repair validation](shrine-repair-validation/results.json): policy v2 advanced past all five original stall points.
- `dungeon-trials-20261016-rerun/`: the earlier preparation helper and script.
- `campaign-candidate.txt`: the command output from the original seed-6 route finder, before these timed experiments.

Archived helper and runner snapshots retain their original contents, including historical paths. Use the current
[runner](../run.ps1) for new experiments. Diagnostic binaries now live under `../bin/`; their build overlays live
under `../tmp/`. Raw results and saved experiment source are kept here, separate from generated binaries and caches.
