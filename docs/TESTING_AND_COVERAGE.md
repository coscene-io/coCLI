# Testing guide (internal)

This doc is for developers working on cocli: how to run tests, check coverage locally, and add tests the same way the codebase does today.

---

## Run tests

```bash
# All tests
go test ./...

# Short mode (skip tests that call testutil.SkipIfShort)
go test -short ./...

# With race detector (same as CI)
go test -race ./...
```

Make targets (from repo root): `make test`, `make shorttest`, `make testrace`.

---

## Run coverage locally

CI runs coverage and uploads to Codecov. To get the same numbers and report locally:

```bash
go test -race -coverprofile=coverage.txt -covermode=atomic ./...
```

Then:

```bash
# Per-package and total
go tool cover -func=coverage.txt
go tool cover -func=coverage.txt | grep total

# Line-by-line in browser
go tool cover -html=coverage.txt
```

`coverage.txt` is gitignored. Optional: `make cover` builds a combined HTML report under the make `TMP` dir (cross-package coverage view).

---

## Codecov (CI)

- Config: `.codecov.yml`
- Workflow: `.github/workflows/test.yaml` runs the same `go test -race -coverprofile=coverage.txt ...` and uploads `coverage.txt`.
- Only production code is counted; `*_test.go`, `*_mock.go`, `internal/testutil`, `internal/apimocks` are ignored.

---

## How to write tests

### Where tests live

- **CLI commands**: `pkg/cmd/<command>/<command>_test.go` (e.g. `record_test.go`, `login_test.go`). Use package `*_test` (e.g. `record_test`) so you only use exported APIs.
- **API layer**: `api/*_test.go` with gomock.
- **Helpers / small units**: next to the code or in the same package, `*_test.go`.

### Patterns we use

- **Config in tests**: Commands take `getProvider func(string) config.Provider`. In tests either:
  - `config.Provide` for real config (e.g. empty file), or
  - `getProvider := func(string) config.Provider { return mockProvider }` for a fake (e.g. `internal/apimocks.NewMockProvider(t)`).
- **Output**: Use `iostreams.Test(nil, &buf, &buf)` and assert on `buf.String()` so we don’t depend on real stdout/stderr.
- **Temp files / dirs**: `testutil.TempDir(t)`, `testutil.CreateTempFile(...)`, etc. in `internal/testutil`.
- **Long or env-dependent tests**: Call `testutil.SkipIfShort(t)` or `testutil.RequireEnv(t, "VAR")` so `go test -short` stays fast.

### Adding tests to improve coverage

1. Run coverage (see above) and open the HTML to see uncovered lines.
2. Add tests where it matters: new commands, config/auth paths, and new logic. Prefer the same patterns (injected `getProvider`, `io`, mocks) so tests stay fast and stable.
3. Re-run tests and coverage to confirm:
   ```bash
   go test ./pkg/cmd/... ./internal/... -count=1 -short
   go test -race -coverprofile=coverage.txt -covermode=atomic ./...
   go tool cover -func=coverage.txt | grep total
   ```
