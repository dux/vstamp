#!/bin/sh
# Install vstamp from GitHub releases:
#   curl -fsSL https://raw.githubusercontent.com/dux/vstamp/main/install.sh | sh
# VSTAMP_VERSION=1.2.3 pins a release, VSTAMP_INSTALL_DIR overrides ~/.local/bin.
set -eu

repo=dux/vstamp
dir=${VSTAMP_INSTALL_DIR:-$HOME/.local/bin}

fail() {
  echo "vstamp: $*" >&2
  exit 1
}

case $(uname -s) in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) fail "unsupported OS $(uname -s), download from https://github.com/$repo/releases" ;;
esac
case $(uname -m) in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "unsupported CPU $(uname -m), download from https://github.com/$repo/releases" ;;
esac

if [ -n "${VSTAMP_VERSION:-}" ]; then
  base=https://github.com/$repo/releases/download/v${VSTAMP_VERSION#v}
else
  base=https://github.com/$repo/releases/latest/download
fi

bin=vstamp-$os-$arch
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL -o "$tmp/$bin" "$base/$bin" || fail "download failed: $base/$bin"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" || fail "download failed: $base/checksums.txt"

expected=$(awk -v f="$bin" '$2 == f { print $1 }' "$tmp/checksums.txt")
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp/$bin" | awk '{ print $1 }')
else
  actual=$(shasum -a 256 "$tmp/$bin" | awk '{ print $1 }')
fi
[ -n "$expected" ] && [ "$expected" = "$actual" ] || fail "checksum mismatch for $bin"

mkdir -p "$dir"
chmod +x "$tmp/$bin"
mv "$tmp/$bin" "$dir/vstamp"
echo "installed vstamp $("$dir/vstamp" --version) to $dir/vstamp"
case ":$PATH:" in
  *":$dir:"*) ;;
  *) echo "add $dir to your PATH" ;;
esac
