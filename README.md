# coScene CLI (coCLI)

`cocli` 是刻行时空（coScene）的命令行工具，方便用户在终端和自动化过程中对刻行时空平台的资源进行管理。

---

## 安装

```bash
curl -fL https://download.coscene.cn/cocli/install.sh | sh
```

验证安装：

```bash
cocli version
```

安装指定版本：

```bash
curl -fL https://download.coscene.cn/cocli/install.sh | sh -s -- v1.4.2
```

---

## 快速开始

### 1. 登录认证

```bash
cocli login add
```

根据提示输入 API endpoint 和 token，或使用已有的 profile。

### 2. 列出项目

```bash
cocli project list
```

### 3. 上传文件到 record

```bash
cocli record upload <record-id> ./data/ -p <project-slug>
```

### 4. 下载 record 的所有文件

```bash
cocli record download <record-id> ./output/ -p <project-slug>
```

---

## 核心场景

### 数据上传工作流

```bash
# 1. 列出可用 projects
cocli project list

# 2. 创建新 record
cocli record create -t <record-title> [-p <project>]

# 3. 上传数据到 record（支持目录、glob 模式）
cocli record upload <record-id> ./data/ [-p <project>]

# 4. 验证上传
cocli record file list <record-id> [-p <project>]
```

### 数据下载工作流

```bash
# 下载整个 record
cocli record download <record-id> ./output/ [-p <project>]

# 或选择性下载
cocli record file download <record-id> ./output/ --dir logs/ [-p <project>]
```

### 项目级文件管理

```bash
# 上传资源文件到 project
cocli project file upload <project> ./shared-data/

# 列出和下载
cocli project file list <project>
cocli project file download <project> ./output/
```

---

## Shell 补全

启用 shell 补全可以自动完成命令、flag 和参数，大幅提升使用体验。

### Bash

```bash
cocli completion bash | sudo tee /etc/bash_completion.d/cocli
source ~/.bashrc
```

### Zsh

```bash
cocli completion zsh > "${fpath[1]}/_cocli"
# 或
cocli completion zsh > ~/.zsh/completions/_cocli
```

重新加载：

```bash
autoload -U compinit && compinit
```

### Fish

```bash
cocli completion fish > ~/.config/fish/completions/cocli.fish
```

---

## 高级功能

### 环境变量配置（适用于 CI/CD）

对于 Docker 容器或 CI/CD 环境，可以通过环境变量配置，无需交互式登录：

| 环境变量       | 描述             | 必需 |
| -------------- | ---------------- | ---- |
| `COS_ENDPOINT` | API 端点地址     | ✅   |
| `COS_TOKEN`    | 认证令牌         | ✅   |
| `COS_PROJECT`  | 默认项目 slug    | ✅   |

示例：

```bash
export COS_ENDPOINT=https://openapi.coscene.cn
export COS_TOKEN=your-api-token
export COS_PROJECT=your-project-slug

cocli record list
```

### Glob 模式上传

使用 glob 模式选择性上传文件：

```bash
# 上传目录（保留目录名）
cocli project file upload <project> data/

# 只上传目录内容（不含目录名）
cocli project file upload <project> "data/*"

# 上传特定类型文件
cocli project file upload <project> "logs/*.log"
```

---

## 开发

### 本地构建

```bash
git clone https://github.com/coscene-io/cocli.git
cd cocli
make build-binary
./bin/cocli version
```

### 快速测试

```bash
go run cmd/cocli/main.go [command]
```

---

## 帮助与文档

- 所有命令支持 `-h` 查看帮助：`cocli <command> -h`
- 详细文档：[coScene CLI 文档](https://docs.coscene.cn/docs/category/cocli)
- 问题反馈：[GitHub Issues](https://github.com/coscene-io/cocli/issues)
