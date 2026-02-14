# GOVM - Go 版本管理器

[English](./README.md)

一个轻量级的、跨平台的 Go 版本管理器，用纯 Go 编写。轻松安装、切换和管理多个 Go 版本，无需系统级配置。

## 特性

- **简单轻量**：单个二进制文件，无任何依赖
- **跨平台支持**：支持 Windows、Linux、macOS
- **多版本管理**：同时安装和管理多个 Go 版本
- **快速切换**：在已安装的版本间瞬间切换
- **下载缓存**：下载的文件缓存，快速重新安装
- **实时进度条**：下载时显示进度条
- **校验和验证**：自动 SHA256 校验，详细的反馈信息
- **纯 Go 编写**：无 shell 脚本，完全用 Go 实现
- **简化配置**：只需设置一个环境变量 `GOROOT=~/.govm/go`

## 安装

### 下载二进制文件

从 [codefloe Releases](https://codefloe.com/apps/govm/releases) 下载最新版本，并添加到 PATH。

### 从源码编译

```bash
git clone https://codefloe.com/apps/govm.git
cd govm
go build -o govm ./cmd
```

## 快速开始

### 列出可用版本

```bash
# 列出所有可用的 Go 版本
govm list

# 仅列出稳定版本
govm list --stable
# 或
govm list -s
```

### 安装 Go 版本

```bash
# 安装 Go 1.25.6
govm use 1.25.6

# 使用自定义镜像安装
govm use 1.25.6 -s https://mirrors.aliyun.com/golang/
# 或
govm use 1.25.6 --site https://golang.google.cn/dl/
```

支持的镜像站点：
- `https://go.dev/dl/` （默认）
- `https://golang.google.cn/dl/`
- `https://mirrors.aliyun.com/golang/`
- `https://mirrors.hust.edu.cn/golang/`
- `https://mirrors.nju.edu.cn/golang/`

### 删除 Go 版本

```bash
# 删除 Go 1.25.6
govm remove 1.25.6

# 使用 flag 语法
govm remove -v 1.25.6
```

## 配置

在 shell 配置文件中添加（`~/.bashrc`、`~/.zshrc` 等）：

```bash
export GOROOT=~/.govm/go/
export PATH=$GOROOT/bin:$PATH
```

重新加载 shell：

```bash
source ~/.bashrc  # 或 source ~/.zshrc
```

验证安装：

```bash
go version
```

## 目录结构

```
~/.govm/
├── go/              # 当前活跃的 Go 版本
│   ├── bin/
│   ├── lib/
│   ├── src/
│   └── ...
├── versions/             # 所有已安装的 Go 版本
│   ├── 1.25.6/
│   ├── 1.24.11/
│   └── ...
├── downloads/            # 已下载的 Go 发行版（缓存）
│   ├── go1.25.6.tar.gz
│   ├── go1.24.11.zip
│   └── ...
├── local.json            # 配置文件
├── versions.json         # 远程版本列表缓存
└── govm.log              # 日志文件
```

## 使用示例

### 安装多个版本

```bash
$ govm use 1.25.6
✓ downloading [file]="go1.25.6.linux-amd64.tar.gz"
[==============================] 150.0 MB / 150.0 MB (100.0%)
✓ SHA256 verification passed: go1.25.6.linux-amd64.tar.gz
✓ version installed and set as current [version]="1.25.6"

$ govm use 1.24.11
✓ downloading [file]="go1.24.11.linux-amd64.tar.gz"
[==============================] 145.0 MB / 145.0 MB (100.0%)
✓ SHA256 verification passed: go1.24.11.linux-amd64.tar.gz
✓ version installed and set as current [version]="1.24.11"
```

### 查看已安装版本

```bash
$ govm list
1.23.0          # 绿色 = 已安装
1.24.11         # 绿色 = 已安装
1.25.6          # 绿色 = 已安装
...
```

### 快速版本切换

```bash
# 切换到已安装的 1.24.11（已缓存，跳过下载）
$ govm use 1.24.11
✓ file found in downloads, skipping download [file]="go1.24.11.linux-amd64.tar.gz"
✓ version installed and set as current [version]="1.24.11"
```

### 删除版本

```bash
$ govm remove 1.23.0
✓ removed version directory [version]="1.23.0"
✓ removed file from downloads [file]="go1.23.0.linux-amd64.tar.gz"
✓ version removed [version]="1.23.0"
```

这会删除：
- `versions/1.23.0/` 目录
- `downloads/` 中的下载文件
- 如果是当前版本，清空 `go/` 目录

## 工作原理

### 首次安装 (govm use 1.25.6)

1. 检查 `downloads/` 中是否存在该文件
2. 如果不存在，从指定的镜像下载
3. 验证 SHA256 校验和
4. 解压到 `versions/1.25.6/`
5. 复制到 `go/`
6. 更新 `local.json`

### 后续使用 (govm use 1.25.6)

1. 文件已在 `downloads/` 中，跳过下载
2. 解压到 `versions/1.25.6/`
3. 从 `versions/1.25.6/` 复制到 `go/`
4. 更新 `local.json`

## 系统要求

- **操作系统**：Windows、Linux、macOS
- **架构**：x86_64、arm64（取决于 Go 的可用性）
- **磁盘空间**：每个 Go 版本约 200-300 MB
- **内存**：最小（govm 本身 < 50 MB）

## 项目结构

```
govm/
├── cmd/                        # CLI 入口和命令定义
│   ├── main.go                 # 入口文件，根命令
│   ├── list.go                 # list 命令
│   ├── use.go                  # use 命令
│   └── remove.go               # remove 命令
├── govm/                       # 核心业务逻辑
│   ├── types.go                # Version、VersionFile、LocalData 类型定义
│   ├── manager.go              # Manager：Init、Install、Uninstall、Sync
│   └── storage.go              # 本地文件读写（JSON 序列化）
├── internal/
│   ├── archive/                # 归档解压（tar.gz、zip）
│   ├── download/               # 文件下载（带进度条）、SHA256 校验
│   └── fsutil/                 # 文件系统工具（CopyDir、HomeDir 等）
├── logger/                     # 结构化日志（控制台 + 文件）
├── go.mod
└── go.sum
```

## 故障排除

### 问题：命令未找到

**解决方案**：确保 `govm` 二进制文件在 PATH 中：
```bash
export PATH=$PATH:/path/to/govm
```

### 问题：版本未找到

**解决方案**：该版本可能不可用。检查可用版本：
```bash
govm list
```

### 问题：SHA256 校验失败

**解决方案**：下载的文件可能已损坏。尝试使用其他镜像：
```bash
govm use 1.25.6 -s https://golang.google.cn/dl/
```

### 问题：GOROOT 配置不正确

**解决方案**：验证 shell 配置文件：
```bash
echo $GOROOT
# 应输出：/home/<user>/.govm/go

echo $PATH
# 应包含：/home/<user>/.govm/go/bin
```

## 常见问题

**Q：我能同时使用多个 Go 版本吗？**

A：可以！每个版本都安装在 `~/.govm/versions/{version}/` 中。`go/` 目录存放当前活跃版本的副本。

**Q：govm 会与我的系统 Go 安装冲突吗？**

A：不会。govm 仅管理 `~/.govm/` 中的版本。系统 Go（如果有的话）不受影响。

**Q：我需要多少磁盘空间？**

A：每个 Go 版本约 150-200 MB。计划每个版本约 260 MB（包括缓存下载）。

**Q：govm 在 Windows 上可以使用吗？**

A：可以！govm 是跨平台的，支持 Windows、Linux 和 macOS。

**Q：如何卸载 govm？**

A：只需删除 `~/.govm/` 目录，并从 PATH 中移除 govm 二进制文件。

## 贡献

欢迎贡献！请随时提交 Pull Request。

## 许可证

该项目采用 Apache License 2.0 许可证 - 详见 LICENSE 文件。

## 支持

如有问题和功能请求，请访问 [codefloe Issues](https://codefloe.com/apps/govm/issues) 页面。
