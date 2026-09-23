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
# Keep flags identical to the regular build so packaging reuses the CGO cache.
export CGO_CXXFLAGS="${CGO_CXXFLAGS:--O2 -g -std=c++17}"
go build -ldflags '-s -w' -o "$bundle/Contents/MacOS/webfence" ./cmd/webfence
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
"$(brew --prefix qtbase)/bin/macdeployqt" "$bundle" -always-overwrite
printf 'Development bundle: %s/%s\n' "$PWD" "$bundle"
