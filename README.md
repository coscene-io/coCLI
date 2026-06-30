# coScene CLI (coCLI) [![Tests](https://github.com/coscene-io/cocli/actions/workflows/test.yaml/badge.svg?branch=main)](https://github.com/coscene-io/cocli/actions/workflows/test.yaml) [![codecov](https://codecov.io/gh/coscene-io/cocli/graph/badge.svg?branch=main)](https://codecov.io/gh/coscene-io/cocli)

English | [简体中文](README.zh-CN.md)

`cocli` is the command-line interface for coScene. For full usage and product documentation, use the coScene CLI docs.

## Install

Global / IO:

```bash
curl -fL https://download.coscene.io/cocli/install.sh | sh
```

Install a specific version:

```bash
curl -fL https://download.coscene.io/cocli/install.sh | sh -s -- v1.7.0-rc2
```

Verify installation:

```bash
cocli --version
```

## Profiles

`cocli` stores named login profiles in its config file and uses the configured
`current-profile` by default (switch with `cocli login switch <name>`).

To target a different profile for a single command without changing the config,
use the global `--profile` flag:

```bash
cocli record list --profile staging
```

You can also drive a one-off profile entirely from environment variables by
setting the complete `COS_*` set (`COS_ENDPOINT`, `COS_TOKEN`, `COS_PROJECT`;
optional `COS_PROJECTID`). A partial set is ignored.

Resolution precedence (highest first):

```
--profile NAME  >  complete COS_* env  >  config current-profile
```

`--profile` and `COS_*` overrides are applied in memory only and are never
written back to the config file, so different profiles can be used concurrently.
The `--profile` flag has no effect on `cocli login` subcommands, which always
operate on the on-disk config.

## Help and Docs

- Command help: `cocli <command> -h`
- Docs: [coScene CLI Docs](https://docs.coscene.io/docs/category/cocli/)
- China mainland version: [README.zh-CN.md](README.zh-CN.md)
- Issues: [GitHub Issues](https://github.com/coscene-io/cocli/issues)

## Development

Build locally:

```bash
git clone https://github.com/coscene-io/cocli.git
cd cocli
make build-binary
./bin/cocli --version
```

Run locally:

```bash
go run ./cmd/cocli --version
```

## License

[Apache-2.0](LICENSE)
