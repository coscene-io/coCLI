# coScene CLI (coCLI)

`cocli` 是刻行时空（coScene）的命令行工具，方便用户在终端和自动化过程中对刻行时空平台的资源进行管理。

`cocli` 所有的命令都可以通过添加 `-h` 参数查看帮助。

详细的图文操作指南和常见操作实例方法请参考 [刻行时空 coCli 文档](https://docs.coscene.cn/docs/category/cocli)。

## 安装

```shell
# 通过 curl 安装
curl -fL https://download.coscene.cn/cocli/install.sh | sh

# 安装具体版本
curl -fL https://download.coscene.cn/cocli/install.sh | sh -s -- v1.x.y
curl -fL https://download.coscene.cn/cocli/install.sh | sh -s -- v1.x.y-rc6
```

## 环境变量配置

当没有配置文件或配置文件为空时，`cocli` 可以通过环境变量进行配置。这对于 Docker 容器、CI/CD 环境或自动化部署场景特别有用。

### 支持的环境变量

| 环境变量       | 描述                 | 必需 | 示例                         |
| -------------- | -------------------- | ---- | ---------------------------- |
| `COS_ENDPOINT` | coScene API 端点地址 | ✅   | `https://openapi.coscene.cn` |
| `COS_TOKEN`    | API 认证令牌         | ✅   | `your-api-token`             |
| `COS_PROJECT`  | 默认项目 slug        | ✅   | `your-project-slug`          |

## 本地安装

### 克隆代码

```shell
git clone https://github.com/coscene-io/cocli.git
```

### 本地快速测试

```shell
go run cmd/cocli/main.go [具体命令]
```

### 本地构建可执行文件

```shell
# 构建可执行文件, 生成的可执行文件在 `./bin` 目录下
make build-binary

# 将可执行文件移动到任意系统路径 PATH 下以便全局使用，当前示例移动到 `/usr/local/bin/` 目录下
mv bin/cocli /usr/local/bin/

# 运行 cocli 命令, 查看帮助文档, 确认安装成功
cocli -h
```
