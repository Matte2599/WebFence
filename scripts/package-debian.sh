#!/bin/sh
# Development package; Qt is supplied by Debian, not copied into this package.
set -eu
[ "$(uname -s)" = Linux ] || { echo 'Debian Linux required.' >&2; exit 1; }
arch=$(dpkg --print-architecture)
case "$arch" in amd64|arm64) ;; *) echo 'Only amd64 and arm64 are supported.' >&2; exit 1;; esac
[ "$(go env GOARCH)" = "$arch" ] || { echo 'Native build required.' >&2; exit 1; }
cd "$(dirname "$0")/.."
repo=$PWD
stage=$(mktemp -d)
pending=''
cleanup() {
  rm -rf "$stage"
  if [ -n "$pending" ]; then rm -f "$pending"; fi
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
root="$stage/root"
doc="$root/usr/share/doc/webfence"
mkdir -p "$root/usr/bin" "$root/usr/share/applications" "$doc" "$root/DEBIAN" "$stage/debian"
export CGO_CXXFLAGS="${CGO_CXXFLAGS:--O2 -g -std=c++17}"
go build -buildvcs=false -ldflags '-s -w' -o "$root/usr/bin/webfence" ./cmd/webfence
install -m 644 packaging/debian/webfence.desktop "$root/usr/share/applications/"
desktop-file-validate "$root/usr/share/applications/webfence.desktop"
install -m 644 LICENSE "$doc/copyright"
python3 scripts/package-project-docs.py "$doc"
python3 scripts/package-go-notices.py "$root/usr/bin/webfence" "$doc/notices"
cat > "$stage/debian/control" <<'CONTROL'
Source: webfence
Section: devel
Priority: optional
Maintainer: Matteo Luigi Feroldi

Package: webfence
Architecture: any
Description: WebFence offline desktop feasibility prototype
CONTROL
deps=$(cd "$stage" && dpkg-shlibdeps -O -e"$root/usr/bin/webfence")
case "$deps" in shlibs:Depends=*) deps=${deps#shlibs:Depends=};; *) echo 'Could not derive ELF dependencies.' >&2; exit 1;; esac
cat > "$root/DEBIAN/control" <<CONTROL
Package: webfence
Version: 0.0.1~m0
Section: devel
Priority: optional
Architecture: $arch
Maintainer: Matteo Luigi Feroldi
Homepage: https://github.com/Matte2599/WebFence
Depends: $deps
Suggests: qt6-wayland
Description: WebFence offline desktop feasibility prototype
 Native Italian/English Qt interface using synthetic examples only.
 No web scanning, CVE retrieval or AI analysis. Development package.
 Source-available WebFence Community License; company use requires permission.
CONTROL
# Include inputs and native build versions; no claim of a complete SBOM.
dpkg-query -W -f='${binary:Package}\t${Version}\n' libqt6core6 libqt6gui6 libqt6widgets6 libc6 libstdc++6 > "$doc/native-build.tsv"
(find cmd internal scripts packaging DOCS experiments/qt -type f ! -name '*.pyc' ! -name '.DS_Store' -print0; printf 'go.mod\0go.sum\0LICENSE\0README.md\0README.en.md\0AGENTS.md\0MEMORY.md\0CONTRIBUTING.md\0SECURITY.md\0') | sort -z | xargs -0 sha256sum > "$doc/source-inputs.sha256"
printf 'revision=%s\ndirty=%s\n' "${WEBFENCE_SOURCE_REVISION:-unrecorded}" "${WEBFENCE_SOURCE_DIRTY:-unknown}" > "$doc/source-revision.txt"
# Report the installed footprint to APT instead of its default unknown/zero size.
installed_size=$(du -sk "$root/usr" | awk '{print $1}')
printf 'Installed-Size: %s\n' "$installed_size" >> "$root/DEBIAN/control"
chmod -R go-w "$root"
dpkg-deb --root-owner-group --build "$root" "$stage/webfence.deb"
dpkg-deb --info "$stage/webfence.deb"
mkdir -p "$repo/dist"
# Rename only after a complete package exists on the destination filesystem.
pending=$(mktemp "$repo/dist/.webfence-deb.XXXXXX")
cp "$stage/webfence.deb" "$pending"
mv "$pending" "$repo/dist/webfence_0.0.1~m0_${arch}.deb"
