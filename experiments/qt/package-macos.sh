#!/bin/sh
# Feasibility bundle only: not a signed/notarized product release.
set -eu
if [ "$(uname -s)" != Darwin ] || [ "$(uname -m)" != arm64 ]; then
  echo 'Requires Apple Silicon macOS.' >&2
  exit 1
fi
cd "$(dirname "$0")"
export CGO_CXXFLAGS="${CGO_CXXFLAGS:--O2 -g} -std=c++17"
bundle=../../dist/WebFence-Qt-Lab.app
mkdir -p "$bundle/Contents/MacOS"
go build -ldflags '-s -w' -o "$bundle/Contents/MacOS/webfence-qt" .
cat > "$bundle/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>webfence-qt</string>
<key>CFBundleIdentifier</key><string>io.github.Matte2599.WebFence.QtLab</string>
<key>CFBundleName</key><string>WebFence Qt Lab</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>0.0.1</string>
<key>CFBundleVersion</key><string>1</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
PLIST
plutil -lint "$bundle/Contents/Info.plist"
"$(brew --prefix qtbase)/bin/macdeployqt" "$bundle" -always-overwrite
printf 'Experimental local bundle: %s\n' "$bundle"
