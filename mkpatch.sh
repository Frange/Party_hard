#!/bin/bash
# Update patch: ONLY what changed from v1.0.0 (the version the testers got) to the one built now (not the full game).
#
#   ./mkpatch.sh                  -> PartyHardy-Parche-Mac-1.0.0-a-<ver>.zip and PartyHardy-Parche-Windows-1.0.0-a-<ver>.zip
#
# Owner's policy (2026-11-12): there are NO patches between intermediate versions. Each release ships the full zips and
# ONE pair of patches, always from 1.0.0 to the latest version; the older patches (1.0.0-a-1.1.0, 1.1.0-a-1.3.0...) are
# deleted from this folder when the new one is written. Any base other than 1.0.0 is refused.
#
# Run it AFTER ./build.sh: it compares releases/manifest-<base>-<mac|win>.json (written by build.sh for every
# version) with PartyHardy-Mac.zip / PartyHardy-Windows.zip and writes the patches in this folder. Each patch
# carries the new and changed files (same folders as the game), REMOVIDOS.txt (to delete), LEEME_ACTUALIZAR.txt
# (ES/EN: how to apply it and the CHANGELOG between both versions) and VERIFICAR_BASE.txt (hashes of the base).
# It never carries progress.json, settings.json, editor_prefs.json, imports/ or intros/*.mp3.
# The tool is tools/mkpatch (its own Go module; tests: go -C tools/mkpatch test .).
set -e
cd "$(dirname "$0")"
PATCH_BASE=1.0.0
if [ $# -gt 1 ] || { [ $# -eq 1 ] && [ "$1" != "$PATCH_BASE" ]; }; then
  echo "uso: ./mkpatch.sh   (sólo hay UN parche, desde $PATCH_BASE hasta la última versión; no se hacen parches entre versiones)" >&2
  exit 1
fi
VERSION=$(sed -n 's/^const gameVersion = "\(.*\)"$/\1/p' version.go)
[ -n "$VERSION" ] || { echo "no gameVersion in version.go" >&2; exit 1; }
for z in PartyHardy-Mac.zip PartyHardy-Windows.zip; do
  [ -f "$z" ] || { echo "falta $z: ejecuta antes ./build.sh" >&2; exit 1; }
done
go -C tools/mkpatch run . -from "$PATCH_BASE" -to "$VERSION" \
  -mac "$PWD/PartyHardy-Mac.zip" -win "$PWD/PartyHardy-Windows.zip" -dir "$PWD/releases" -out "$PWD"
# only the patch to the latest version stays: the older ones (and any between versions) go
for old in PartyHardy-Parche-*.zip; do
  [ -e "$old" ] || continue
  case "$old" in
    PartyHardy-Parche-Mac-$PATCH_BASE-a-$VERSION.zip|PartyHardy-Parche-Windows-$PATCH_BASE-a-$VERSION.zip) ;;
    *) rm -f "$old" && echo "borrado parche antiguo: $old" ;;
  esac
done
