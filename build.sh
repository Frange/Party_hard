#!/bin/bash
# Builds the game and the editor for Windows and macOS (Intel + Apple Silicon) into ./dist
# Requires Go. Run from the project folder.
set -e
cd "$(dirname "$0")"
export CGO_ENABLED=0
OUT=dist
rm -f PartyHard-Mac.zip PartyHard-Windows.zip
rm -rf $OUT && mkdir -p $OUT/mac $OUT/win $OUT/tmp

mkapp() { # name binary dest
  local name="$1" bin="$2" dest="$3" id="$4"
  rm -rf "$dest/$name.app" && mkdir -p "$dest/$name.app/Contents/MacOS"
  cp "$bin" "$dest/$name.app/Contents/MacOS/$name" && chmod +x "$dest/$name.app/Contents/MacOS/$name"
  cat > "$dest/$name.app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PLIST-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleName</key><string>$name</string>
<key>CFBundleDisplayName</key><string>$name</string>
<key>CFBundleIdentifier</key><string>$id</string>
<key>CFBundleExecutable</key><string>$name</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleVersion</key><string>1.0</string>
<key>CFBundleShortVersionString</key><string>1.0</string>
<key>LSMinimumSystemVersion</key><string>11.0</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
PLIST
  codesign --force --deep -s - "$dest/$name.app" 2>/dev/null || true
}

for tags in "" "editor"; do
  suffix=""; [ -n "$tags" ] && suffix="Editor"
  GOOS=windows GOARCH=amd64 go build -tags "$tags" -ldflags "-H=windowsgui -s -w" -o "$OUT/win/PartyHard$suffix.exe" .
  GOOS=darwin GOARCH=amd64 go build -tags "$tags" -ldflags "-s -w" -o "$OUT/tmp/ph$suffix-amd64" .
  GOOS=darwin GOARCH=arm64 go build -tags "$tags" -ldflags "-s -w" -o "$OUT/tmp/ph$suffix-arm64" .
  lipo -create "$OUT/tmp/ph$suffix-amd64" "$OUT/tmp/ph$suffix-arm64" -output "$OUT/tmp/ph$suffix"
  mkapp "PartyHard$suffix" "$OUT/tmp/ph$suffix" "$OUT/mac" "local.partyhard.clone$suffix"
done
rm -rf $OUT/tmp

# package: executables + assets + docs, zipped per platform (assets are the ones in ./assets)
if [ -d assets ]; then
  for d in mac win; do
    cp -R assets $OUT/$d/assets
    cp LEEME.txt ASSETS.md $OUT/$d/
  done
  (cd $OUT/mac && ditto -c -k --keepParent . ../../PartyHard-Mac.zip 2>/dev/null || zip -qr ../../PartyHard-Mac.zip .)
  (cd $OUT/win && zip -qr ../../PartyHard-Windows.zip .)
fi
echo "built into $OUT"
