# TODO

Planned features for the next release(s). See below for details.

---

## 1. `deps remove` command

Add a `remove` command to cleanly remove a dependency:

```
deps remove github.com/user/repo
```

Should:
- Remove the entry from `.deps.lock`
- Delete the directory from `.deps/github.com/user/repo`
- Remove any alias symlink (see below) that points to it

---

## 2. Aliases with symlinks

Allow an optional alias when adding a dependency:

```
deps get github.com/raysan5/raylib raylib
```

This would create a symlink in `.deps/`:

```
.deps/raylib -> .deps/github.com/raysan5/raylib
```

So you can import using the short name:

```
import ".deps/raylib"
```

instead of:

```
import ".deps/github.com/raysan5/raylib"
```

Things to consider:
- Store the alias in `.deps.lock` (e.g. `"alias": "raylib"`)
- Recreate symlinks on `deps install`
- Handle alias conflicts (two deps can't share an alias)
- Windows support — skip symlink creation on Windows (aliases would be a Unix-only feature)
- `deps remove` should accept either the full URL or the alias

---

## 3. Self-update check

On startup, check if a newer version of `deps` itself is available and show a soft notice:

```
Note: deps v1.2.0 is available (you have v1.1.0). See https://github.com/moomerman/deps/releases
```

Things to consider:
- Query the GitHub releases API for `moomerman/deps` (single lightweight call)
- Cache the result so we don't check on every invocation (e.g. check at most once per day, store timestamp in `~/.cache/deps/` or similar)
- Never block or slow down the main command — fail silently if the network is unavailable
- Respect `NO_COLOR` / CI environments — maybe skip the check entirely in non-interactive contexts
- Could run the check in a goroutine so it doesn't add latency

---

## 4. Configurable install directory

Allow users to configure the install directory instead of hardcoding `.deps/`.

Store the install directory in `.deps.lock` so that `deps install` knows where to put things:

```json
{
  "dir": "vendor",
  "dependencies": { ... }
}
```

Set it via `deps init --dir vendor` or `deps get --dir vendor` (first dep sets it).

Useful for projects that prefer `vendor/`, `third_party/`, `lib/`, etc.

Things to consider:
- Needs a sensible default (`.deps/` as today, used when `dir` is absent from lock file)
- The lock file path should probably stay as `.deps.lock` regardless
- Alias symlinks (see above) should respect the configured directory
- Document the `.gitignore` implications
- May need a `deps init` command to set the dir before adding any dependencies