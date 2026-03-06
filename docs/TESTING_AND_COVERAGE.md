# Testing guide (internal)

This doc is for developers working on cocli: how to run tests, check coverage locally, and add tests the same way the codebase does today.

---

## Run tests

```bash
# Unit tests (CI runs these)
go test ./...

# With race detector (same as CI)
go test -race ./...

# Integration tests (manual, requires valid cocli profile)
go test -tags=integration ./test/integration/ -v
```

---

## Run coverage locally

CI runs coverage and uploads to Codecov. To get the same numbers locally:

```bash
go test -race -coverprofile=coverage.txt -covermode=atomic ./...
```

Then:

```bash
# Per-package and total
go tool cover -func=coverage.txt | grep total

# Line-by-line in browser
go tool cover -html=coverage.txt
```

`coverage.txt` is gitignored.

---

## Codecov (CI)

- Config: `.codecov.yml`
- Workflow: `.github/workflows/test.yaml` uploads `coverage.txt`.
- `project` status uses `target: auto` with `threshold: 1%` (coverage must not drop more than 1% from base).
- `patch` status is off (new code is not required to have coverage).
- Ignored paths: `*_test.go`, `*_mock.go`, `internal/testutil`, `internal/apimocks`, `scripts`, `vendor`.

---

## Test structure

### Unit tests (CI, `go test ./...`)

| Directory | What's tested | Pattern |
|-----------|--------------|---------|
| `api/*_test.go` | API client methods (Get, Create, List, etc.) | Function-field mocks (see `api/record_mock_test.go`) |
| `api/api_utils/*_test.go` | Retry interceptor, auth | Mock `connect.UnaryFunc` |
| `pkg/cmd/<cmd>/*_test.go` | Command structure, flags, subcommands | `iostreams.Test` + `config.Provide` |
| `internal/name/*_test.go` | Resource name parsing (regex) | Table-driven, zero deps |
| `internal/fs/*_test.go` | File traversal, SHA256 | `t.TempDir()` |
| `internal/config/*_test.go` | Profile/ProfileManager validation | Direct struct construction |
| `internal/utils/*_test.go` | Connect error helpers | `connect.NewError` |
| `pkg/cmd_utils/upload_utils/*_test.go` | Heap, file opts, glob | Table-driven, `t.TempDir()` |

### Integration tests (manual, `go test -tags=integration`)

Located in `test/integration/`. Uses real cocli profile (`~/.cocli.yaml`) to call the live API.

| File | Purpose |
|------|---------|
| `helpers_test.go` | Shared `liveProfileManager` and `liveContext` |
| `status_code_test.go` | Verify error status codes (NOT_FOUND, INVALID_ARGUMENT) |
| `api_smoke_test.go` | Smoke test stable read-only APIs (ListProjects, GetProject, SearchRecords, ListActions) |

Override config: `COCLI_CONFIG=/path/to/config.yaml go test -tags=integration ./test/integration/ -v`

---

## How to write tests

### Mock pattern for API clients

Mock definitions live in `*_mock_test.go` files (e.g. `api/record_mock_test.go`). Each mock embeds the generated Connect client interface and exposes configurable `func` fields:

```go
type mockRecordServiceClient struct {
    openv1alpha1connect.RecordServiceClient
    getRecordFunc func(...) (...)
}
```

Tests configure the func field per case. Unconfigured methods return `CodeUnimplemented`.

### Command tests

Commands take `getProvider func(string) config.Provider`. In tests either:
- `config.Provide` with an empty temp file for structure tests, or
- `apimocks.NewMockProvider(t)` for behavior tests.

Use `iostreams.Test(nil, &buf, &buf)` to capture output.

### Adding new tests

1. Run `go tool cover -html=coverage.txt` to find uncovered lines.
2. Follow existing patterns: table-driven for pure functions, function-field mocks for API clients, `iostreams.Test` for commands.
3. Verify:
   ```bash
   go test ./... -count=1
   go test -race -coverprofile=coverage.txt -covermode=atomic ./...
   go tool cover -func=coverage.txt | grep total
   ```
