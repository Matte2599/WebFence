#!/bin/sh
# Local development bundle only; no distribution signature or notarization.
set -eu

if [ "$(uname -s)" != Darwin ] || [ "$(uname -m)" != arm64 ]; then
  echo 'This prototype packaging script requires macOS on Apple Silicon.' >&2
  exit 1
fi

cd "$(dirname "$0")/.."
bundle=dist/WebFence.app
mkdir -p "$bundle/Contents/MacOS"
# Opt in only for accessibility investigation: FYNE_BUILD_TAGS=accessibility.
# Fyne's experimental bridge has not passed WebFence's M0 acceptance gate.
go build -tags "${FYNE_BUILD_TAGS:-}" -trimpath -o "$bundle/Contents/MacOS/webfence" ./cmd/webfence
cat > "$bundle/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key><string>webfence</string>
  <key>CFBundleIdentifier</key><string>io.github.Matte2599.WebFence</string>
  <key>CFBundleName</key><string>WebFence</string>
  <key>CFBundleDisplayName</key><string>WebFence</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>0.0.1</string>
  <key>CFBundleVersion</key><string>1</string>
  <key>NSHighResolutionCapable</key><true/>
  <key>NSPrincipalClass</key><string>NSApplication</string>
  <key>NSHumanReadableCopyright</key><string>Copyright 2026 Matteo Luigi Feroldi. WebFence Community License 1.0.</string>
</dict>
</plist>
PLIST
plutil -lint "$bundle/Contents/Info.plist"
printf 'Development bundle: %s/%s\n' "$PWD" "$bundle"
