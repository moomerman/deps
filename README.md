# deps

Language-agnostic dependency manager for GitHub repositories. Downloads source code directly into a local `.deps/` directory and tracks exact versions in a `.deps.lock` file.

## Install

Download a pre-built binary from the [releases page](https://github.com/moomerman/deps/releases/latest):

```
# macOS Apple Silicon
curl -L https://github.com/moomerman/deps/releases/latest/download/deps-darwin-arm64.tar.gz | tar xz

# macOS Intel
curl -L https://github.com/moomerman/deps/releases/latest/download/deps-darwin-amd64.tar.gz | tar xz

# Linux x64
curl -L https://github.com/moomerman/deps/releases/latest/download/deps-linux-amd64.tar.gz | tar xz

# Linux ARM64
curl -L https://github.com/moomerman/deps/releases/latest/download/deps-linux-arm64.tar.gz | tar xz
```

Windows: download and extract the [Windows release](https://github.com/moomerman/deps/releases/latest/download/deps-windows-amd64.zip).

Or build from source:

```
git clone https://github.com/moomerman/deps.git
cd deps
CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o deps *.go
```

## Usage

```
deps get github.com/user/repo              # add dependency (latest default branch)
deps get github.com/user/repo@v1.2.3       # add dependency (specific tag)
deps get github.com/user/repo@main         # add dependency (specific branch)
deps get github.com/user/repo@abc123...    # add dependency (specific commit)

deps check                                  # check status and available updates
deps install                                # install dependencies from lock file
deps update                                 # update all dependencies
deps update github.com/user/repo           # update a specific dependency

deps version
deps help
```

## Project structure

```
your-project/
├── .deps.lock          # commit this — pinned versions for reproducible builds
├── .deps/              # gitignore this — downloaded source code
│   └── github.com/
│       └── user/repo/
│           └── ...
└── ...
```

Add `.deps/` to your `.gitignore`. Keep `.deps.lock` in version control.

## Lock file format

```json
{
  "dependencies": {
    "github.com/user/repo": {
      "ref": "v1.2.3",
      "sha": "75ccf94d605a05fe24817fc2f166f6f2959d5cea",
      "hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    }
  }
}
```

| Field  | Description |
|--------|-------------|
| `ref`  | The branch, tag, or SHA you specified |
| `sha`  | The resolved commit SHA that was downloaded |
| `hash` | SHA-256 of the downloaded tarball, verified on install |

Lock files created before v1.1.0 won't have `hash` — it will be populated automatically on the next `deps install`.

## Importing dependencies

How you reference `.deps/` depends on your language:

```odin
// Odin
import sapp ".deps/github.com/floooh/sokol-odin/sokol/app"
```

```c
// C/C++
#include ".deps/github.com/user/repo/header.h"
```

```python
# Python
import sys
sys.path.append('.deps/github.com/user/repo')
```

## GitHub Action

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: moomerman/deps@v1
    with:
      command: install    # or: check, update
      working-directory: .
```

## Limitations

- GitHub public repositories only (private repo support planned)
- No transitive dependency resolution
- No semantic version ranges

## License

MIT — see [LICENSE](LICENSE).