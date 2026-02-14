# GOVM - Go Version Manager

[中文文档](./README.zh.md)

A lightweight, cross-platform Go version manager written in pure Go. Easily install, switch between, and manage multiple Go versions without system-wide configuration.

## Features

- **Simple & Lightweight**: Single binary, no dependencies
- **Cross-Platform**: Works on Windows, Linux, macOS
- **Multiple Versions**: Install and manage multiple Go versions simultaneously
- **Fast Switching**: Switch between installed versions instantly
- **Download Caching**: Downloaded files are cached for quick reinstallation
- **Real-time Progress**: Visual progress bar during downloads
- **Checksum Verification**: Automatic SHA256 verification with detailed feedback
- **Pure Go**: No shell scripts, fully written in Go
- **Easy Configuration**: Single environment variable `GOROOT=~/.govm/go`

## Installation

### Download Binary

Download the latest release from [codefloe Releases](https://codefloe.com/apps/govm/releases) and add it to your PATH.

### Build from Source

```bash
git clone https://codefloe.com/apps/govm.git
cd govm
go build -o govm ./cmd
```

## Quick Start

### List Available Versions

```bash
# List all available Go versions
govm list

# List only stable versions
govm list --stable
# or
govm list -s
```

### Install a Go Version

```bash
# Install Go 1.25.6
govm use 1.25.6

# Install with a custom mirror
govm use 1.25.6 -s https://mirrors.aliyun.com/golang/
# or
govm use 1.25.6 --site https://golang.google.cn/dl/
```

Supported mirror sites:
- `https://go.dev/dl/` (default)
- `https://golang.google.cn/dl/`
- `https://mirrors.aliyun.com/golang/`
- `https://mirrors.hust.edu.cn/golang/`
- `https://mirrors.nju.edu.cn/golang/`

### Remove a Go Version

```bash
# Remove Go 1.25.6
govm remove 1.25.6

# Using flag syntax
govm remove -v 1.25.6
```

## Configuration

Add the following to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
export GOROOT=~/.govm/go/
export PATH=$GOROOT/bin:$PATH
```

Then reload your shell:

```bash
source ~/.bashrc  # or source ~/.zshrc
```

Verify the installation:

```bash
go version
```

## Directory Structure

```
~/.govm/
├── go/              # Current active Go version
│   ├── bin/
│   ├── lib/
│   ├── src/
│   └── ...
├── versions/             # All installed Go versions
│   ├── 1.25.6/
│   ├── 1.24.11/
│   └── ...
├── downloads/            # Downloaded Go distributions (cache)
│   ├── go1.25.6.tar.gz
│   ├── go1.24.11.zip
│   └── ...
├── local.json            # Configuration file
├── versions.json         # Remote version list cache
└── govm.log              # Log file
```

## Usage Examples

### Install Multiple Versions

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

### View Installed Versions

```bash
$ govm list
1.23.0          # green = installed
1.24.11         # green = installed
1.25.6          # green = installed
...
```

### Quick Version Switching

```bash
# Switch to version 1.24.11 (already installed, skips download)
$ govm use 1.24.11
✓ file found in downloads, skipping download [file]="go1.24.11.linux-amd64.tar.gz"
✓ version installed and set as current [version]="1.24.11"
```

### Remove a Version

```bash
$ govm remove 1.23.0
✓ removed version directory [version]="1.23.0"
✓ removed file from downloads [file]="go1.23.0.linux-amd64.tar.gz"
✓ version removed [version]="1.23.0"
```

This removes:
- The version from `versions/1.23.0/`
- The downloaded file from `downloads/`
- Clears `go/` if it was the active version

## How It Works

### First Installation (govm use 1.25.6)

1. Check if file exists in `downloads/`
2. If not, download from the specified mirror
3. Verify SHA256 checksum
4. Extract to `versions/1.25.6/`
5. Copy to `go/`
6. Update `local.json`

### Subsequent Use (govm use 1.25.6)

1. File already in `downloads/`, skip download
2. Extract to `versions/1.25.6/`
3. Copy from `versions/1.25.6/` to `go/`
4. Update `local.json`

## System Requirements

- **OS**: Windows, Linux, macOS
- **Architecture**: x86_64, arm64 (depending on Go's availability)
- **Disk Space**: ~200-300 MB per Go version
- **Memory**: Minimal (< 50 MB for govm itself)

## Project Structure

```
govm/
├── cmd/                        # CLI entry point and command definitions
│   ├── main.go                 # Entry point, root command
│   ├── list.go                 # List command
│   ├── use.go                  # Use command
│   └── remove.go               # Remove command
├── govm/                       # Core business logic
│   ├── types.go                # Version, VersionFile, LocalData types
│   ├── manager.go              # Manager: Init, Install, Uninstall, Sync
│   └── storage.go              # Local file I/O (JSON read/write)
├── internal/
│   ├── archive/                # Archive extraction (tar.gz, zip)
│   ├── download/               # File download with progress, SHA256 verification
│   └── fsutil/                 # File system utilities (CopyDir, HomeDir, etc.)
├── logger/                     # Structured logging (console + file)
├── go.mod
└── go.sum
```

## Troubleshooting

### Issue: Command not found

**Solution**: Ensure `govm` binary is in your PATH:
```bash
export PATH=$PATH:/path/to/govm
```

### Issue: Go version not found

**Solution**: The version might not be available. Check with:
```bash
govm list
```

### Issue: SHA256 verification failed

**Solution**: The downloaded file might be corrupted. Try again with a different mirror:
```bash
govm use 1.25.6 -s https://golang.google.cn/dl/
```

### Issue: GOROOT not set correctly

**Solution**: Verify your shell profile configuration:
```bash
echo $GOROOT
# Should output: /home/<user>/.govm/go

echo $PATH
# Should contain: /home/<user>/.govm/go/bin
```

## FAQ

**Q: Can I use multiple Go versions simultaneously?**

A: Yes! Each version is installed in `~/.govm/versions/{version}/`. The `go/` directory holds a copy of the active version.

**Q: Will govm interfere with my system Go installation?**

A: No. govm only manages versions in `~/.govm/`. Your system Go (if any) is unaffected.

**Q: How much disk space do I need?**

A: Each Go version is ~150-200 MB. Plan for ~260 MB per version (including cached downloads).

**Q: Can I use govm on Windows?**

A: Yes! govm is cross-platform and works on Windows, Linux, and macOS.

**Q: How do I uninstall govm?**

A: Simply delete the `~/.govm/` directory and remove the govm binary from your PATH.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

## Support

For issues and feature requests, please visit the [codefloe Issues](https://codefloe.com/apps/govm/issues) page.
