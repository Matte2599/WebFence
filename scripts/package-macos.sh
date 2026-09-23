#!/bin/sh
# Local development bundle only; no distribution signature or notarization.
set -eu

if [ "$(uname -s)" != Darwin ] || [ "$(uname -m)" != arm64 ]; then
  echo 'This prototype packaging script requires macOS on Apple Silicon.' >&2
  exit 1
fi

cd "$(dirname "$0")/.."
# Stage on the host's temporary filesystem: repeatedly rewriting load commands
# directly on an external volume is slow and can leave a partial final bundle.
build_temp=$(mktemp -d "${TMPDIR:-/tmp}/webfence-macos.XXXXXX")
publish_temp=''
backup=''
cleanup() {
  if [ -n "$backup" ] && [ -d "$backup" ] && [ ! -e dist/WebFence.app ]; then
    mv "$backup" dist/WebFence.app || return
  fi
  rm -rf "$build_temp"
  if [ -n "$publish_temp" ]; then rm -rf "$publish_temp"; fi
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
bundle="$build_temp/WebFence.app"
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
qt_prefix=$(brew --prefix qtbase)
"$qt_prefix/bin/macdeployqt" "$bundle" -always-overwrite
sh scripts/build-qt-cocoa.sh "$qt_prefix" "$bundle/Contents/PlugIns/platforms/libqcocoa.dylib"
python3 scripts/package-macos-notices.py "$bundle" "$qt_prefix"
codesign --force --deep --sign - "$bundle"
codesign --verify --deep --strict "$bundle"
mkdir -p dist
# Copy fully before renaming on the destination filesystem. Retain the previous
# generated artifact for rollback until the new bundle has been published.
publish_temp=$(mktemp -d dist/.webfence-publish.XXXXXX)
ditto "$bundle" "$publish_temp/WebFence.app"
codesign --verify --deep --strict "$publish_temp/WebFence.app"
if [ -e dist/WebFence.app ]; then
  backup="$publish_temp/previous.app"
  mv dist/WebFence.app "$backup"
fi
mv "$publish_temp/WebFence.app" dist/WebFence.app
printf 'Development bundle: %s/dist/WebFence.app\n' "$PWD"
