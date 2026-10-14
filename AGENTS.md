# vstamp

Go CLI (stdlib only, Go 1.25) that sets `version: X.Y.Z` in the YAML `.version` from the git commit count and can amend, tag and push it.
User docs are in `README.md`.

## Layout

* `main.go` - CLI parsing, `show` / `tag` / `count` commands, safety checks, git step plan
* `config.go` - `.version` parse (top-level `version` / `offset` only) and in-place rewrite of the `version` line
* `version.go` - `Version` parse/compare, `FromCount`, `MinCountAbove`
* `git.go` - git wrappers: branch choice (master, main), count, last tag, staged files, upstream
* `manifest.go` - in-place `package.json` top-level `"version"` rewrite
* `*_test.go` - unit tests plus integration tests in temp repos with a bare remote
* `release.sh` - test, cross-compile to `dist/`, `count --commit --tag --push`, `gh release create`
* `install.sh` - POSIX installer for release binaries with checksum check

## Rules

* Every check runs before `.version` or `package.json` is written.
* Tests isolate git with `GIT_CONFIG_GLOBAL=/dev/null`; keep new integration tests on the `newRepo` / `withRemote` helpers.

## Commands

```sh
go vet ./...
go test ./...
./release.sh --dry-run
```
