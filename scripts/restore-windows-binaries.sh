#!/usr/bin/env bash
# Run from the repository root in MSYS2 UCRT64 before building the desktop.
set -euo pipefail

cache_root=${1:-/var/cache/pacman/pkg}
binary_lock=${2:-packaging/windows/msys2-binary-lock.json}
plan=$(python3 - "$binary_lock" <<'PY'
import sys
sys.path.insert(0, 'scripts')
from msys2_binary_metadata import load_binary_lock
for entry in load_binary_lock(sys.argv[1]).values():
    print(entry['name'], entry['version'], entry['sha256'], sep='\t')
PY
)
mkdir -p "$cache_root"
pending=()
download=''
trap 'if [ -n "$download" ]; then rm -f "$download"; fi' EXIT
while IFS=$'\t' read -r name version digest; do
    installed=$(pacman -Q "$name" | awk '{print $2}')
    filename="$name-$version-any.pkg.tar.zst"
    archive="$cache_root/$filename"
    if ! printf '%s  %s\n' "$digest" "$archive" | sha256sum --check --status -; then
        download=$(mktemp "$cache_root/.webfence-reviewed.XXXXXX")
        curl --fail --location --proto '=https' --proto-redir '=https' \
            --connect-timeout 20 --max-time 180 --max-filesize 536870912 --retry 3 \
            "https://mirror.msys2.org/mingw/ucrt64/$filename" --output "$download"
        printf '%s  %s\n' "$digest" "$download" | sha256sum --check -
        mv -f "$download" "$archive"
        download=''
    fi
    if [ "$installed" != "$version" ]; then
        pending+=("$archive")
    fi
done <<< "$plan"
if [ "${#pending[@]}" -gt 0 ]; then
    # Restore mutually dependent packages in one checked transaction.
    pacman -U --noconfirm "${pending[@]}"
fi
while IFS=$'\t' read -r name version digest; do
    test "$(pacman -Q "$name" | awk '{print $2}')" = "$version"
    printf 'PASS reviewed Windows package: %s %s\n' "$name" "$version"
done <<< "$plan"
