# deps GitHub Action

A GitHub Action to install and manage dependencies using the `deps` tool - a language-agnostic dependency manager for GitHub repositories.

## Usage

### Install Dependencies

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: moomerman/deps@v1
    with:
      command: install
```

### Check Dependencies

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: moomerman/deps@v1
    with:
      command: check
```

### Update Dependencies

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: moomerman/deps@v1
    with:
      command: update
```

### Working Directory

Run the action in a specific directory:

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: moomerman/deps@v1
    with:
      command: install
      working-directory: ./my-project
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `command` | Command to run (`install`, `check`, `update`) | No | `install` |
| `working-directory` | Directory to run deps in | No | `.` |

## Example Workflows

### Basic CI with Dependency Installation

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install dependencies
        uses: moomerman/deps@v1
        with:
          command: install
      
      - name: Build project
        run: |
          # Your build commands here
          make build
```

### Multi-language Project

```yaml
name: Build

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install dependencies
        uses: moomerman/deps@v1
      
      - name: Build Odin project
        run: |
          odin build src -out:app
      
      - name: Build C project
        run: |
          gcc -I.deps/github.com/vendor/lib src/main.c -o app
```

### Dependency Update Bot

```yaml
name: Update Dependencies

on:
  schedule:
    - cron: '0 0 * * 1' # Weekly on Monday

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Update dependencies
        uses: moomerman/deps@v1
        with:
          command: update
      
      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v5
        with:
          title: "Update dependencies"
          body: "Automated dependency update"
          branch: deps-update
```

## About

This action uses the `deps` tool, a fast and simple dependency manager that works with any programming language by downloading source code directly from GitHub repositories.

For more information about the `deps` tool, see [the main repository](https://github.com/moomerman/deps).