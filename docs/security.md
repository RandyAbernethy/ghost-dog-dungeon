# Security scans

Build with Go 1.26.9 or newer, as required by [go.mod](../go.mod). CI checks Go 1.26.9 and 1.27.2. These releases
include the [Go security fixes](https://go.dev/doc/devel/release) needed by the October 18, 2026 Trivy database.
The game uses only the Go standard library; its compiled version matters even without external dependencies.

Scan source dependencies, configuration, and secrets with a current Trivy database:

```sh
trivy fs --scanners vuln,misconfig,secret --detection-priority comprehensive \
  --severity HIGH,CRITICAL --exit-code 1 \
  --skip-dirs .git --skip-dirs bin --skip-dirs montecarlo/bin \
  --skip-dirs montecarlo/tmp --skip-dirs tests/.tmp .
```

Scan the actual binaries separately. Choose an unused output directory to preserve existing executables:

```sh
make all BIN_DIR=bin/security-review
trivy rootfs --scanners vuln --severity HIGH,CRITICAL --exit-code 1 bin/security-review
```

Trivy's `rootfs` mode inspects the compiler version embedded in Go binaries. A source-only scan with the default
detection settings can miss standard-library findings; `comprehensive` includes the minimum version in `go.mod`.
See [Trivy's Go coverage](https://trivy.dev/docs/latest/coverage/language/golang/). These scans check known dependencies,
configuration, and secrets; they do not audit the application's own Go logic.

Rebuild older executables before distributing them. No vulnerability exclusions are needed for the patched builds.
