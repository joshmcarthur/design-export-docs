#!/usr/bin/env bash
# Build release archives for every supported platform.
#
#   ./release.sh <version> [outdir]      e.g. ./release.sh 0.1.0
#
# Writes <outdir>/design-export-docs_<version>_<os>_<arch>.{tar.gz,zip} and SHA256SUMS (default outdir: release/).
# Binaries are static (CGO off), so they run on any machine of the target OS and CPU.
set -euo pipefail

version="${1:?usage: release.sh <version> [outdir]}"
out="${2:-release}"
cd "$(dirname "$0")"

rm -rf "$out"
mkdir -p "$out"

for target in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64; do
  os="${target%/*}"
  arch="${target#*/}"
  name="design-export-docs_${version}_${os}_${arch}"
  bin="design-export-docs"
  [ "$os" = "windows" ] && bin="design-export-docs.exe"

  mkdir -p "$out/$name"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "-s -w -X main.version=${version}" -o "$out/$name/$bin" .
  cp LICENSE README.md "$out/$name/"

  if [ "$os" = "windows" ]; then
    (cd "$out" && zip -qr "$name.zip" "$name")
  else
    tar -C "$out" -czf "$out/$name.tar.gz" "$name"
  fi
  rm -rf "${out:?}/$name"
done

(cd "$out" && shasum -a 256 -- *.tar.gz *.zip > SHA256SUMS)
count=$(find "$out" -maxdepth 1 \( -name '*.tar.gz' -o -name '*.zip' \) | wc -l | tr -d ' ')
echo "built $count archives in $out/"
