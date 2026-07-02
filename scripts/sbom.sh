#!/usr/bin/env sh
set -eu
out=${1:-dist/sbom.txt}
mkdir -p "$(dirname "$out")"
{
  echo '# PES minimal SBOM baseline'
  echo '## Go modules'
  go list -m all
  echo '## Packages'
  go list ./...
} > "$out"
