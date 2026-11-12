#!/bin/bash
# Update patches: ONLY what changed from an older version to the one built now (not the full game).
#
#   ./mkpatch.sh 1.0.0            -> PartyHardy-Parche-Mac-1.0.0-a-<ver>.zip and PartyHardy-Parche-Windows-1.0.0-a-<ver>.zip
#   ./mkpatch.sh 1.0.0 1.0.1      -> one pair of patches per base version
#
# Run it AFTER ./build.sh: it compares releases/manifest-<base>-<mac|win>.json (written by build.sh for every
# version) with PartyHardy-Mac.zip / PartyHardy-Windows.zip and writes the patches in this folder. Each patch
# carries the new and changed files (same folders as the game), REMOVIDOS.txt (to delete), LEEME_ACTUALIZAR.txt
# (ES/EN: how to apply it and the CHANGELOG between both versions) and VERIFICAR_BASE.txt (hashes of the base).
# It never carries progress.json, settings.json, editor_prefs.json, imports/ or intros/*.mp3.
# The tool is tools/mkpatch (its own Go module; tests: go -C tools/mkpatch test .).
set -e
cd "$(dirname "$0")"
[ $# -ge 1 ] || { echo "uso: ./mkpatch.sh <versión base> [<otra base> ...]   (p. ej. ./mkpatch.sh 1.0.0)" >&2; exit 1; }
VERSION=$(sed -n 's/^const gameVersion = "\(.*\)"$/\1/p' version.go)
[ -n "$VERSION" ] || { echo "no gameVersion in version.go" >&2; exit 1; }
for z in PartyHardy-Mac.zip PartyHardy-Windows.zip; do
  [ -f "$z" ] || { echo "falta $z: ejecuta antes ./build.sh" >&2; exit 1; }
done
for base in "$@"; do
  go -C tools/mkpatch run . -from "$base" -to "$VERSION" \
    -mac "$PWD/PartyHardy-Mac.zip" -win "$PWD/PartyHardy-Windows.zip" -dir "$PWD/releases" -out "$PWD"
done
