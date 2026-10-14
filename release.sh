#!/usr/bin/env bash
# Release vstamp: test, build all binaries, stamp + tag + push with vstamp
# itself, then publish a GitHub release. --dry-run stops after the build.
set -euo pipefail
cd "$(dirname "$0")"

dry=""
case "${1:-}" in
  "") ;;
  --dry-run) dry=1 ;;
  -h | --help) echo "Usage: ./release.sh [--dry-run]"; exit 0 ;;
  *) echo "unknown argument: $1" >&2; exit 1 ;;
esac

fail() { echo "release: $*" >&2; exit 1; }

[[ $(git symbolic-ref -q --short HEAD) == main ]] || fail "switch to main first"
[[ -z $(git status --porcelain) ]] || fail "commit all changes first"
[[ -n $dry ]] || gh auth status >/dev/null 2>&1 || fail "run gh auth login first"

go vet ./...
go test ./...

# first line of the dry run is "<old> -> <new> (dry run)"
version=$(go run . count --commit --dry-run | awk 'NR == 1 { print $3 }')
[[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "could not compute next version"

rm -rf dist
mkdir dist
for target in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  os=${target%/*}
  arch=${target#*/}
  ext=""
  [[ $os == windows ]] && ext=.exe
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
    -ldflags "-s -w -X main.version=$version" -o "dist/vstamp-$os-$arch$ext" .
done
(cd dist && shasum -a 256 vstamp-* > checksums.txt)
echo "built dist/ for $version"

if [[ -n $dry ]]; then
  go run . count --commit --tag --push --dry-run
  exit 0
fi

go run . count --commit --tag --push
[[ $(cat .version) == "$version" ]] || fail ".version is $(cat .version), binaries were built as $version"

gh release create "v$version" dist/* --title "vstamp $version" --generate-notes --latest
