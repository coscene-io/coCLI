# coCLI

Go CLI (`cocli`) for managing coScene resources from the terminal.

## Branch Model

**All changes go through `dev` first.** Never target `main` directly.

- Feature/fix branches → PR to `dev`
- Release: `dev` → PR to `main` (creates RC tag, then release tag after verification)
- Hotfixes: branch from `main`, PR to `dev` first, then cherry-pick to `main` if urgent

When creating a PR with `gh pr create`, always use `--base dev`:

```bash
gh pr create --base dev --title "..." --body "..."
```

## Build & Test

```bash
make build-binary          # build to ./bin/cocli
go test ./...              # run all tests
go test ./pkg/cmd/...      # test CLI commands only
```

## Conventions

- Module: `github.com/coscene-io/cocli`
- Use `cmd_utils.ProfileManager(cmd, getProvider, cfgPath)` for profile resolution (supports --profile flag and COS_* env)
- Error handling: `log.Fatalf` for fatal CLI errors
- Output formats: support `-o table|json|yaml` via `printer.Printer()`
