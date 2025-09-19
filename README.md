# deps

A fast, language-agnostic dependency manager for GitHub projects.

## Overview

`deps` is a simple command-line tool for managing dependencies from GitHub repositories. Unlike language-specific package managers, `deps` works with any programming language by downloading source code directly to a local `.deps/` directory.

Originally created for the [Odin programming language](https://odin-lang.org/) which lacks a built-in package manager, `deps` is designed to be language-agnostic and can manage dependencies for any project.

## Features

- **Language Agnostic**: Works with any programming language
- **GitHub Integration**: Fetches dependencies directly from GitHub repositories
- **Version Pinning**: Support for branches, tags, and specific commit SHAs
- **Lock File**: Ensures reproducible builds with `.deps.lock`
- **Clean Structure**: Dependencies are organized in `.deps/github.com/user/repo/`
- **Update Management**: Check for and install updates selectively
- **Fast Downloads**: Uses GitHub's tarball API for efficient transfers

## Installation

### Pre-built Binaries (Recommended)

Download the latest release for your platform from the [releases page](https://github.com/moomerman/deps/releases/latest).

**Linux/macOS:**
```bash
# Linux x64
curl -L https://github.com/moomerman/deps/releases/latest/download/deps-linux-amd64.tar.gz | tar xz

# Linux ARM64
curl -L https://github.com/moomerman/deps/releases/latest/download/deps-linux-arm64.tar.gz | tar xz

# macOS Intel
curl -L https://github.com/moomerman/deps/releases/latest/download/deps-darwin-amd64.tar.gz | tar xz

# macOS Apple Silicon
curl -L https://github.com/moomerman/deps/releases/latest/download/deps-darwin-arm64.tar.gz | tar xz
```

**Windows:**
Download and extract the [Windows release](https://github.com/moomerman/deps/releases/latest/download/deps-windows-amd64.zip).

### GitHub Action (CI/CD)

Use in your GitHub workflows:

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: moomerman/deps@v1
    with:
      command: install
```

### Docker

```bash
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/moomerman/deps-action:latest install
```

### Build from Source

```bash
git clone https://github.com/moomerman/deps.git
cd deps
# Production build (recommended)
CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o deps *.go

# Or use just (if installed)  
just build-prod
```

## Quick Start

1. **Add a dependency:**
   ```bash
   deps get github.com/gin-gonic/gin@v1.10.0
   ```

2. **Check status:**
   ```bash
   deps check
   ```

3. **Install missing dependencies:**
   ```bash
   deps install
   ```

## Usage

### Adding Dependencies

Add the latest version from the default branch:
```bash
deps get github.com/user/repo
```

Add a specific version (tag):
```bash
deps get github.com/user/repo@v1.2.3
```

Add from a specific branch:
```bash
deps get github.com/user/repo@main
```

Add a specific commit:
```bash
deps get github.com/user/repo@abc123def456...
```

### Managing Dependencies

**Check dependency status:**
```bash
deps check
```

**Install missing dependencies from lockfile:**
```bash
deps install
```

**Update all dependencies:**
```bash
deps update
```

**Update a specific dependency:**
```bash
deps update github.com/user/repo
```

**Show version:**
```bash
deps version
```

**Show help:**
```bash
deps help
```

## GitHub Action Usage

### Basic CI Workflow

```yaml
name: CI
on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: moomerman/deps@v1
        with:
          command: install
      - name: Build
        run: |
          # Your build commands
          odin build src -out:app
```

### Available Commands

- `install` - Install dependencies from `.deps.lock` (default)
- `check` - Check dependency status
- `update` - Update all dependencies

### Action Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `command` | Command to run | `install` |
| `working-directory` | Directory to run in | `.` |

## Project Structure

After adding dependencies, your project will look like:

```
your-project/
├── .deps.lock                    # Lock file (commit to version control)
├── .deps/                        # Dependencies directory (add to .gitignore)
│   └── github.com/
│       ├── gin-gonic/gin/
│       │   ├── gin.go           # Source files
│       │   ├── auth.go
│       │   └── ...
│       └── user/repo/
│           └── ...
└── your-source-files...
```

### .gitignore Setup

Add this to your `.gitignore`:
```gitignore
# Dependencies
.deps/
```

**Keep the `.deps.lock` file** - it should be committed to ensure reproducible builds.

## Language Integration

Since `deps` is language-agnostic, integration depends on your language's import/include system:

### Odin
```odin
import sapp ".deps/github.com/floooh/sokol-odin/sokol/app"
import sgfx ".deps/github.com/floooh/sokol-odin/sokol/gfx"
```

### C/C++
```c
#include ".deps/github.com/user/repo/header.h"
```

### Go (for non-module projects)
```go
import ".deps/github.com/user/repo"
```

### Python
```python
import sys
sys.path.append('.deps/github.com/user/repo')
import module
```

## Lock File Format

The `.deps.lock` file tracks exact versions for reproducible builds:

```json
{
  "dependencies": {
    "github.com/floooh/sokol-odin": {
      "ref": "main",
      "sha": "2fbaae3c245b2f65c961ef4a38482c81f6bbae6c"
    },
    "github.com/gin-gonic/gin": {
      "ref": "v1.10.0",
      "sha": "75ccf94d605a05fe24817fc2f166f6f2959d5cea"
    }
  }
}
```

- `ref`: What you specified (e.g., `v1.10.0`, `main`, `feature-branch`)
- `sha`: The exact commit SHA that was downloaded

The lockfile format is intentionally minimal - containing only the essential information needed for reproducible builds.

## How It Works

1. **Resolution**: `deps` uses the GitHub API to resolve branches/tags to specific commit SHAs
2. **Download**: Downloads the repository tarball from GitHub's CDN  
3. **Extraction**: Extracts source files directly to `.deps/github.com/owner/repo/` (no hash subdirectories)
4. **Tracking**: Records exact commit SHAs in a minimal `.deps.lock` file for reproducibility
5. **Verification**: Simple directory existence checks (trusts the lockfile for exact versions)

## Limitations

- **GitHub Only**: Currently supports GitHub repositories only
- **Public Repos**: Only public repositories (private repo support planned)
- **No Transitive Dependencies**: Dependencies of dependencies must be managed manually
- **No Semantic Versioning**: No automatic resolution of version ranges

## Roadmap

- [ ] Support for private repositories
- [ ] GitLab and Bitbucket support
- [ ] Transitive dependency resolution

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Why deps?

Traditional package managers are tightly coupled to specific languages and ecosystems. `deps` takes a simpler approach: just download the source code and let your language's import system handle the rest.

This approach works particularly well for:
- Languages without established package managers
- Projects that need direct access to source code
- Cross-language projects
- Simple dependency management without complex resolution

Built with ❤️ for the [Odin](https://odin-lang.org/) community and anyone who needs simple, reliable dependency management.
