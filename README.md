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
