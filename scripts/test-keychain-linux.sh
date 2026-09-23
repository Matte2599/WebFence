#!/bin/sh
# Run only synthetic credentials on a private session bus and disposable keyring.
set -eu
if [ "${1:-}" != '--inside' ]; then
    test "$(uname -s)" = Linux
    command -v dbus-run-session >/dev/null
    command -v gnome-keyring-daemon >/dev/null
    command -v gdbus >/dev/null
    keychain_temp=$(mktemp -d)
    trap 'rm -rf "$keychain_temp"' EXIT HUP INT TERM
    mkdir -m 700 "$keychain_temp/data" "$keychain_temp/runtime"
    export XDG_DATA_HOME="$keychain_temp/data"
    export XDG_RUNTIME_DIR="$keychain_temp/runtime"
    export WEBFENCE_ISOLATED_SECRET_SERVICE=1
    dbus-run-session -- sh "$0" --inside
    exit
fi
# An empty bus must fail without activating a service or falling back to a file.
WEBFENCE_EXPECT_NO_SECRET_SERVICE=1 go test -tags=keychainintegration ./internal/credentials -run '^TestSessionWithoutSecretService$' -count=1
printf '%s\n' 'webfence synthetic test password' | gnome-keyring-daemon --foreground --unlock --components=secrets >"$XDG_RUNTIME_DIR/daemon.log" 2>&1 &
keychain_pid=$!
trap 'kill "$keychain_pid" 2>/dev/null || true; wait "$keychain_pid" 2>/dev/null || true' EXIT HUP INT TERM
keychain_attempt=0
until gdbus call --session --dest org.freedesktop.DBus --object-path /org/freedesktop/DBus --method org.freedesktop.DBus.NameHasOwner org.freedesktop.secrets 2>/dev/null | rg -q 'true' &&
    gdbus call --session --dest org.freedesktop.secrets --object-path /org/freedesktop/secrets --method org.freedesktop.Secret.Service.ReadAlias default 2>/dev/null | rg -q '/org/freedesktop/secrets/collection/'; do
    keychain_attempt=$((keychain_attempt + 1))
    if [ "$keychain_attempt" -ge 20 ]; then
        cat "$XDG_RUNTIME_DIR/daemon.log"
        exit 1
    fi
    sleep 0.5
done
go test -race -tags=keychainintegration ./internal/credentials -count=1 -v
