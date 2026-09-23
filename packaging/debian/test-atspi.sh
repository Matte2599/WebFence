#!/bin/sh
# Isolated Debian/Xvfb/D-Bus session. Tests the OS accessibility bridge only.
set -eu

test_root=$(mktemp -d /tmp/webfence-atspi.XXXXXX)
wm_pid=
app_pid=
cleanup() {
  if [ -n "$app_pid" ]; then kill "$app_pid" 2>/dev/null || true; wait "$app_pid" 2>/dev/null || true; fi
  if [ -n "$wm_pid" ]; then kill "$wm_pid" 2>/dev/null || true; wait "$wm_pid" 2>/dev/null || true; fi
  rm -rf "$test_root"
}
trap cleanup EXIT HUP INT TERM

openbox > "$test_root/openbox.log" 2>&1 &
wm_pid=$!
attempt=0
until xprop -root _NET_SUPPORTING_WM_CHECK | grep -q 'window id #'; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 50 ]; then cat "$test_root/openbox.log"; exit 1; fi
  sleep 0.1
done

for language in en it; do
  export XDG_CONFIG_HOME="$test_root/$language"
  mkdir -p "$XDG_CONFIG_HOME/WebFence"
  printf '%s' "$language" > "$XDG_CONFIG_HOME/WebFence/ui-language"
  webfence > "$test_root/webfence-$language.log" 2>&1 &
  app_pid=$!
  if ! /usr/bin/python3 /opt/webfence-test/test-atspi.py "$language"; then
    cat "$test_root/webfence-$language.log"
    exit 1
  fi
  kill "$app_pid" 2>/dev/null || true
  wait "$app_pid" 2>/dev/null || true
  app_pid=
done

printf '%s\n' 'PASS Debian AT-SPI bridge regression (EN/IT, 10,000 -> 1 -> 0 -> 10,000)'
