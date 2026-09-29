#!/bin/sh
set -eu

if [ "$(uname -s)" != Darwin ]; then
  echo 'The macOS sandbox lab requires macOS' >&2
  exit 1
fi

cd "$(dirname "$0")/../.."
lab_dir=$(mktemp -d)
bundle_id=org.webfence.m3-sandbox-probe
trap 'rm -rf "$lab_dir"' EXIT

CGO_ENABLED=0 go build -o "$lab_dir/parent" ./experiments/m3-macos-sandbox
bundle="$lab_dir/SandboxProbe.app"
mkdir -p "$bundle/Contents/MacOS"
cp "$lab_dir/parent" "$bundle/Contents/MacOS/SandboxProbe"
cat > "$bundle/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleIdentifier</key><string>$bundle_id</string>
  <key>CFBundleExecutable</key><string>SandboxProbe</string>
  <key>CFBundleName</key><string>WebFence M3 Sandbox Probe</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>LSBackgroundOnly</key><true/>
</dict></plist>
PLIST
codesign --force --sign - \
  --entitlements experiments/m3-macos-sandbox/probe.entitlements.plist \
  "$bundle"

WF_M3_SANDBOXED_PROBE="$bundle/Contents/MacOS/SandboxProbe" \
WF_M3_CHROME="${WF_M3_CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}" \
  "$lab_dir/parent"
