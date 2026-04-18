# coScene CLI (coCLI) [![Tests](https://github.com/coscene-io/cocli/actions/workflows/test.yaml/badge.svg?branch=main)](https://github.com/coscene-io/cocli/actions/workflows/test.yaml) [![codecov](https://codecov.io/gh/coscene-io/cocli/graph/badge.svg?branch=main)](https://codecov.io/gh/coscene-io/cocli)

[English](README.md) | 简体中文

`cocli` 是 coScene 的命令行工具。完整用法和产品文档请查看 coScene CLI 文档。

## 安装

中国大陆 / CN：

```bash
curl -fL https://download.coscene.cn/cocli/install.sh | sh
```

安装指定版本：

```bash
curl -fL https://download.coscene.cn/cocli/install.sh | sh -s -- v1.7.0-rc2
```

验证安装：

```bash
cocli --version
```

## 帮助与文档

- 命令帮助：`cocli <command> -h`
- 文档：[coScene CLI 文档](https://docs.coscene.cn/docs/category/cocli/)
- 默认英文版：[README.md](README.md)
- 问题反馈：[GitHub Issues](https://github.com/coscene-io/cocli/issues)

## 开发

本地构建：

```bash
git clone https://github.com/coscene-io/cocli.git
cd cocli
make build-binary
./bin/cocli --version
```

本地运行：

```bash
go run ./cmd/cocli --version
```

## 许可证

[Apache-2.0](LICENSE)
