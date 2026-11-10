#!/bin/bash
# Builds the game and the editor for Windows and macOS (Intel + Apple Silicon) into ./dist
# Requires Go. Run from the project folder. Icons: icon/ (regenerate with: go run ./tools/mkicon).
set -e
cd "$(dirname "$0")"
export CGO_ENABLED=0
OUT=dist
# the version of the game: the only source is the constant of version.go (CHANGELOG.md must start with the same one)
VERSION=$(sed -n 's/^const gameVersion = "\(.*\)"$/\1/p' version.go)
[ -n "$VERSION" ] || { echo "no gameVersion in version.go" >&2; exit 1; }
CLVER=$(sed -n 's/^## \[v\([^]]*\)\].*/\1/p' CHANGELOG.md 2>/dev/null | head -1)
[ "$CLVER" = "$VERSION" ] || { echo "version.go says $VERSION but the first CHANGELOG.md entry says '$CLVER'" >&2; exit 1; }
rm -f PartyHardy-Mac.zip PartyHardy-Windows.zip
rm -rf $OUT && mkdir -p $OUT/mac $OUT/win $OUT/tmp

mkapp() { # name binary dest
  local name="$1" bin="$2" dest="$3" id="$4"
  rm -rf "$dest/$name.app" && mkdir -p "$dest/$name.app/Contents/MacOS"
  cp "$bin" "$dest/$name.app/Contents/MacOS/$name" && chmod +x "$dest/$name.app/Contents/MacOS/$name"
  mkdir -p "$dest/$name.app/Contents/Resources" && cp icon/AppIcon.icns "$dest/$name.app/Contents/Resources/AppIcon.icns"
  cat > "$dest/$name.app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PLIST-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleName</key><string>$name</string>
<key>CFBundleDisplayName</key><string>$name</string>
<key>CFBundleIdentifier</key><string>$id</string>
<key>CFBundleExecutable</key><string>$name</string>
<key>CFBundleIconFile</key><string>AppIcon</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleVersion</key><string>$VERSION</string>
<key>CFBundleShortVersionString</key><string>$VERSION</string>
<key>LSMinimumSystemVersion</key><string>11.0</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
PLIST
  codesign --force --deep -s - "$dest/$name.app" 2>/dev/null || true
}

for tags in "" "editor"; do
  suffix=""; [ -n "$tags" ] && suffix="Editor"
  GOOS=windows GOARCH=amd64 go build -trimpath -tags "$tags" -ldflags "-H=windowsgui -s -w" -o "$OUT/win/PartyHardy$suffix.exe" .
  GOOS=darwin GOARCH=amd64 go build -trimpath -tags "$tags" -ldflags "-s -w" -o "$OUT/tmp/ph$suffix-amd64" .
  GOOS=darwin GOARCH=arm64 go build -trimpath -tags "$tags" -ldflags "-s -w" -o "$OUT/tmp/ph$suffix-arm64" .
  lipo -create "$OUT/tmp/ph$suffix-amd64" "$OUT/tmp/ph$suffix-arm64" -output "$OUT/tmp/ph$suffix"
  mkapp "PartyHardy$suffix" "$OUT/tmp/ph$suffix" "$OUT/mac" "local.partyhardy.clone$suffix"
done
rm -rf $OUT/tmp

# package: executables + assets + docs, zipped per platform (assets are the ones in ./assets)
if [ -d assets ]; then
  for d in mac win; do
    cp -R assets $OUT/$d/assets
    # personal data stays out of the packages: your own save and editor preferences
    rm -f $OUT/$d/assets/progress.json* $OUT/$d/assets/editor_prefs.json* $OUT/$d/assets/settings.json* $OUT/$d/assets/.held_objects_fixed
    cp LEEME.txt ASSETS.md CHANGELOG.md $OUT/$d/
    printf 'Party Hardy v%s\n' "$VERSION" > $OUT/$d/VERSION.txt
    # music of music/, more/ and DJ/ is shipped; intros/*.mp3 is not (only .bpm and README)
    mkdir -p $OUT/$d/music $OUT/$d/more $OUT/$d/DJ $OUT/$d/intros
    [ -d music ] && cp music/*.mp3 $OUT/$d/music/ 2>/dev/null || true
    [ -d more ] && cp more/*.mp3 $OUT/$d/more/ 2>/dev/null || true
    [ -d DJ ] && cp DJ/*.mp3 $OUT/$d/DJ/ 2>/dev/null || true
    [ -d intros ] && cp intros/*.bpm $OUT/$d/intros/ 2>/dev/null || true
    [ -f intros/README.md ] && cp intros/README.md $OUT/$d/intros/ || true
    # safety net: no personal data, imports, backups or intros mp3 in the package
    find $OUT/$d \( -name 'progress.json*' -o -name 'settings.json*' -o -name 'editor_prefs.json*' -o -name '.DS_Store' \) -type f -delete
    rm -f $OUT/$d/intros/*.mp3
    find $OUT/$d \( -name imports -o -name AI_with_Claude -o -name 'assets_backup_*' \) -type d -prune -exec rm -rf {} +
  done
  (cd $OUT/mac && ditto -c -k --keepParent . ../../PartyHardy-Mac.zip 2>/dev/null || zip -qr ../../PartyHardy-Mac.zip .)
  (cd $OUT/win && zip -qr ../../PartyHardy-Windows.zip .)
fi
echo "built v$VERSION into $OUT"
