#!/usr/bin/env bash
# Start a new Go project from one of the skill's verified assets.
#
#   scripts/new.sh <asset> <dest-dir> <module-path>
#   scripts/new.sh report ~/src/depcheck github.com/jz-wilson/depcheck
#
# Copies assets/<asset> to <dest-dir>, rewrites the module path (go.mod and
# imports), sets `go 1.27`-style minimum, adds .gitattributes, runs go mod tidy,
# gofmt, vet and tests, then git init + first commit. Rename user-facing names
# (kong.Name, help text, testscript command names) afterwards.
# Run from Linux, macOS or WSL (see SKILL.md); <dest-dir> should be on the
# Linux filesystem, not /mnt/c.
set -euo pipefail

usage() { echo "usage: $(basename "$0") <asset> <dest-dir> <module-path>" >&2; exit 2; }
[[ $# -eq 3 ]] || usage
ASSET=$1 DEST=$2 MODULE=$3
SKILL=$(cd "$(dirname "$0")/.." && pwd)
SRC=$SKILL/assets/$ASSET

if [[ ! -d $SRC ]]; then
  echo "new: unknown asset '$ASSET'; one of: $(ls "$SKILL/assets" | tr '\n' ' ')" >&2; exit 2
fi
if [[ $MODULE != *.* ]]; then
  echo "new: module path '$MODULE' has no dot; use github.com/<user>/<tool> (SKILL.md rule 11)" >&2; exit 2
fi
if [[ -e $DEST && -n $(ls -A "$DEST" 2>/dev/null) ]]; then
  echo "new: $DEST exists and is not empty" >&2; exit 1
fi
command -v go >/dev/null || { echo "new: go not found on PATH" >&2; exit 1; }

mkdir -p "$DEST"
cp -r "$SRC/." "$DEST/"
cd "$DEST"

OLD=$(go list -m)
go mod edit -module "$MODULE"
# Rewrite imports of the old module path (internal packages, multi-package assets).
{ grep -rl --include='*.go' "\"$OLD/" . || true; } | xargs -r sed -i "s#\"$OLD/#\"$MODULE/#g"
# Rename the tool: when the asset has exactly one kong.Name, replace that name as a
# whole word everywhere (Go strings, error prefixes, testscript command, Taskfile,
# goreleaser) and its env-var prefix (MODFRESH_ -> CERTCHECK_). The new name is the
# last element of the module path. Multi-binary assets (recipes) are left alone.
NEW_NAME=${MODULE##*/}
mapfile -t names < <(grep -rhoE --include='*.go' 'kong\.Name\("[^"]+"\)' . | sed -E 's/kong\.Name\("([^"]+)"\)/\1/' | sort -u)
if [[ ${#names[@]} -eq 1 && ${names[0]} != "$NEW_NAME" ]]; then
  OLD_NAME=${names[0]}
  OLD_ENV=$(echo "$OLD_NAME" | tr 'a-z-' 'A-Z_')_ NEW_ENV=$(echo "$NEW_NAME" | tr 'a-z-' 'A-Z_')_
  # Go identifiers embed the name capitalized (runModfresh): my-tool -> MyTool.
  camel() { echo "$1" | sed -E 's/(^|[-_])([a-z])/\U\2/g'; }
  OLD_CAMEL=$(camel "$OLD_NAME") NEW_CAMEL=$(camel "$NEW_NAME")
  { grep -rlE --include='*.go' --include='*.txtar' --include='*.yml' --include='*.yaml' --include='.gitignore' \
    "\b$OLD_NAME\b|$OLD_ENV|$OLD_CAMEL" . || true; } |
    xargs -r sed -i -E "s/\b$OLD_NAME\b/$NEW_NAME/g; s/$OLD_ENV/$NEW_ENV/g; s/$OLD_CAMEL/$NEW_CAMEL/g"
  echo "Renamed '$OLD_NAME' -> '$NEW_NAME' (also $OLD_ENV -> $NEW_ENV, $OLD_CAMEL -> $NEW_CAMEL)."
fi
GOVER=$(go env GOVERSION | sed -E 's/^go([0-9]+\.[0-9]+).*/\1/')
go mod edit -go="$GOVER"
[[ -f .gitattributes ]] || printf '* text=auto eol=lf\n' > .gitattributes

go mod tidy
if [[ -n $(gofmt -l .) ]]; then echo "new: gofmt issues: $(gofmt -l .)" >&2; exit 1; fi
go vet ./...
go test ./...
if command -v golangci-lint >/dev/null && [[ -f .golangci.yml ]]; then golangci-lint run ./...; fi

git init -q
git add -A
git commit -qm "Start from go-cli asset '$ASSET'"
echo
echo "Created $DEST ($MODULE) from '$ASSET'. Checks pass."
echo "Next: replace the asset's domain code (see the asset's row in SKILL.md), keep the generic parts."
