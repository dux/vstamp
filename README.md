# vstamp

Keeps the version in `.version` in sync with the git commit count, and can amend, tag and push it before a deploy.

The count on `master` (else `main`) becomes the version: the last two digits are minor and patch, the rest is major.

| Commits | Version |
|---|---|
| 5 | `0.0.5` |
| 380 | `3.8.0` |
| 1111 | `11.1.1` |

## .version

`.version` is a YAML file:

```yaml
version: 3.8.5   # written by vstamp
offset: 14       # optional, added to the commit count
```

* vstamp reads the top-level `version` and `offset` lines and rewrites only the `version` line.
* `version` is an unquoted `X.Y.Z`, `offset` a non-negative integer.
* Other keys and comments are kept as they are, so the file can carry more settings.
* When `.version` is missing, `vstamp count` creates it with a single `version` line.

Reading the version elsewhere:

```ruby
YAML.load_file('.version')['version']
```

```js
/^version:\s*(\S+)/m.exec(fs.readFileSync('.version', 'utf8'))[1]
```

```sh
sed -n 's/^version: *\([^ #]*\).*/\1/p' .version
```

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/dux/vstamp/main/install.sh | sh
```

* `VSTAMP_VERSION=1.2.3` pins a release, `VSTAMP_INSTALL_DIR=/usr/local/bin` changes the target (default `~/.local/bin`).
* Prebuilt binaries for macOS, Linux and Windows (amd64, arm64) are on the [releases page](https://github.com/dux/vstamp/releases), so `mise use ubi:dux/vstamp` works too.
* With Go installed: `go install github.com/dux/vstamp@latest`, or run without installing: `go run github.com/dux/vstamp@latest count`.

## Usage

```
vstamp                 print the version from .version (dev when missing)
vstamp tag             print the last vX.Y.Z tag reachable from HEAD (none, exit 1)
vstamp count [flags]   set the version from the commit count on master (else main)
vstamp --version       print vstamp version
```

| Flag | Effect |
|---|---|
| `--branch <name>` | Count this branch instead of `master`/`main`. |
| `--next` | Count the commit about to be made (+1) without committing, for deploy scripts and pre-commit hooks that commit themselves. |
| `--commit` | Commit `.version` (+ `package.json`) as `release X.Y.Z`. The count includes that commit. |
| `--amend` | Fold `.version` (+ `package.json`) into HEAD, keeping its message. |
| `--tag` | Tag `vX.Y.Z`. Needs `--commit` or `--amend`. |
| `--push` | Push the branch (+ tag) to its upstream. |
| `--release` | Same as `--amend --tag --push`. |
| `--dry-run` | Print the change and every git command, run nothing. |

## Before deploy

```sh
vstamp count --release && ./deploy
```

This stamps `.version`, amends it into the last commit, tags `vX.Y.Z` and runs:

```sh
git push --atomic --force-with-lease --force-if-includes <remote> HEAD:<branch> refs/tags/vX.Y.Z
```

* `--atomic` lands the branch and the tag together or not at all.
* `--force-with-lease` is needed because the amend rewrites HEAD, and it refuses when someone else pushed in the meantime.
* `--force-if-includes` keeps a background `git fetch` from silently approving that overwrite (git 2.30+).
* When the push fails, the local commit and tag stay and vstamp prints the command to retry.

## Safety checks

All checks run before anything is written.

* A `.version` without a valid `version: X.Y.Z` line is an error.
* A count below the current version is an error that names the offset to set, e.g. `count gives 0.1.0, below .version 0.2.3, set "offset: 14" in .version`.
* `--commit` and `--amend` need HEAD on the counted branch, and refuse when other files are staged.
* `--tag` refuses an existing tag, so a release without new commits stops instead of re-tagging.
* `--push` needs an upstream branch.
* `vstamp` and `vstamp count` warn when `.version` is behind the last tag.

## package.json

When `package.json` exists, its top-level `"version"` is set to the same `X.Y.Z`.
Only that value changes, formatting and key order stay as they are.

## Releasing vstamp

```sh
./release.sh --dry-run   # test and build dist/, show the git commands
./release.sh             # build, vstamp count --commit --tag --push, gh release create
```
