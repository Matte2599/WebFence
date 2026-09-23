#!/bin/sh
# Build only the pinned Cocoa plugin in isolation; never modify installed Qt.
set -eu
if [ "$(uname -s)" != Darwin ] || [ "$(uname -m)" != arm64 ] || [ "$#" -ne 2 ]; then
  echo 'Usage on Apple Silicon macOS: sh scripts/build-qt-cocoa.sh QT_PREFIX OUTPUT_DYLIB' >&2
  exit 1
fi
qt_prefix=$1
output=$2
version=$("$qt_prefix/bin/qtpaths" --qt-version)
if [ "$version" != 6.11.2 ]; then
  echo "Cocoa correction is validated only for Qt 6.11.2; found $version. Review before upgrading." >&2
  exit 1
fi
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
stage=$(mktemp -d "${TMPDIR:-/tmp}/webfence-cocoa.XXXXXX")
trap 'rm -rf "$stage"' EXIT
trap 'exit 1' HUP INT TERM
archive="$stage/qtbase.tar.xz"
if [ -n "${WEBFENCE_QT_SOURCE_ARCHIVE:-}" ]; then
  cp "$WEBFENCE_QT_SOURCE_ARCHIVE" "$archive"
else
  curl --fail --location --proto '=https' --proto-redir '=https' --connect-timeout 20 --max-time 180 \
    https://download.qt.io/official_releases/qt/6.11/6.11.2/submodules/qtbase-everywhere-src-6.11.2.tar.xz \
    --output "$archive"
fi
expected=5b2e00eccaf5a4d8c14134ffa0ea8dfd0a35ae1ffc7f8d87fa4305a1ed23cf22
actual=$(shasum -a 256 "$archive" | cut -d ' ' -f 1)
if [ "$actual" != "$expected" ]; then
  echo 'Qt source checksum mismatch; refusing to extract or build.' >&2
  exit 1
fi
tar -xJf "$archive" -C "$stage"
(cd "$stage/qtbase-everywhere-src-6.11.2" && patch --batch --fuzz=0 -p1 < "$script_dir/qt-cocoa/accessibility.patch")
cp "$script_dir/qt-cocoa/CMakeLists.txt" "$stage/CMakeLists.txt"
printf 'set(QT_REPO_MODULE_VERSION "6.11.2")\n' > "$stage/.cmake.conf"
cmake -S "$stage" -B "$stage/build" -G Ninja \
  -DCMAKE_BUILD_TYPE=Release -DCMAKE_OSX_DEPLOYMENT_TARGET=14.0 \
  -DCMAKE_PREFIX_PATH="$qt_prefix" -DCMAKE_INSTALL_PREFIX="$stage/install" \
  -DQT_SKIP_AUTO_PLUGIN_INCLUSION=ON
cmake --build "$stage/build" --target QCocoaIntegrationPlugin --parallel 4
plugin="$stage/build/share/qt/plugins/platforms/libqcocoa.dylib"
for framework in QtCore QtGui; do
  install_name_tool -change "$qt_prefix/lib/$framework.framework/Versions/A/$framework" \
    "@executable_path/../Frameworks/$framework.framework/Versions/A/$framework" "$plugin"
done
install_name_tool -delete_rpath "$qt_prefix/lib" "$plugin"
otool -L "$plugin" | awk 'NR > 1 && $1 !~ /^@executable_path\/\.\.\/Frameworks\// && $1 !~ /^\/System\/Library\// && $1 !~ /^\/usr\/lib\// { bad=1; print "Unexpected Cocoa dependency: " $1 > "/dev/stderr" } END { exit bad }'
codesign --force --sign - "$plugin"
codesign --verify --strict "$plugin"
mkdir -p "$(dirname "$output")"
cp "$plugin" "$output"
