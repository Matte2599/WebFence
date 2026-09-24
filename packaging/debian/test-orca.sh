#!/bin/sh
# Exercise Orca's speech-command path against the installed synthetic GUI.
# Speech Dispatcher uses a private dummy module and ALSA null output: no audio
# is emitted, so this is not a real-desktop or audible-reader acceptance test.
set -eu

language=${1-}
case "$language" in
  en) severity=Low; status=Synthetic ;;
  it) severity=Bassa; status=Sintetica ;;
  *) printf '%s\n' 'Expected en or it' >&2; exit 2 ;;
esac

test_root=$(mktemp -d /tmp/webfence-orca.XXXXXX)
export HOME="$test_root/home"
export XDG_CONFIG_HOME="$test_root/config"
export XDG_RUNTIME_DIR="$test_root/runtime"
export QT_LINUX_ACCESSIBILITY_ALWAYS_ON=1
export QT_QPA_PLATFORM=xcb
mkdir -p "$HOME" "$XDG_CONFIG_HOME/WebFence" "$XDG_RUNTIME_DIR" "$test_root/speechlog"
chmod 700 "$XDG_RUNTIME_DIR"
printf '%s' "$language" > "$XDG_CONFIG_HOME/WebFence/ui-language"
printf 'AddModule "dummy" "sd_dummy" ""\nDefaultModule dummy\nAudioOutputMethod "alsa"\nAudioALSADevice "null"\n' > "$test_root/speechd.conf"

wm_pid=
speechd_pid=
app_pid=
orca_pid=
cleanup() {
  if [ -n "$orca_pid" ]; then kill -KILL "$orca_pid" 2>/dev/null || true; wait "$orca_pid" 2>/dev/null || true; fi
  if [ -n "$app_pid" ]; then kill "$app_pid" 2>/dev/null || true; wait "$app_pid" 2>/dev/null || true; fi
  if [ -n "$speechd_pid" ]; then kill "$speechd_pid" 2>/dev/null || true; wait "$speechd_pid" 2>/dev/null || true; fi
  if [ -n "$wm_pid" ]; then kill "$wm_pid" 2>/dev/null || true; wait "$wm_pid" 2>/dev/null || true; fi
  rm -rf "$test_root"
}
fail() {
  printf 'FAIL Orca %s: %s\n' "$language" "$1" >&2
  for log in speechd.log modules.log atspi.log orca.stdout webfence.log; do
    if [ -f "$test_root/$log" ]; then tail -30 "$test_root/$log" >&2; fi
  done
  if [ -f "$test_root/orca.log" ]; then tail -65 "$test_root/orca.log" >&2; fi
  exit 1
}
trap cleanup EXIT HUP INT TERM

openbox > "$test_root/openbox.log" 2>&1 &
wm_pid=$!
attempt=0
until xprop -root _NET_SUPPORTING_WM_CHECK | grep -q 'window id #'; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 50 ]; then fail 'window manager did not start'; fi
  sleep 0.1
done

speech-dispatcher -s -C "$test_root" -L "$test_root/speechlog" -t 0 > "$test_root/speechd.log" 2>&1 &
speechd_pid=$!
attempt=0
until grep -Fq 'Module dummy loaded.' "$test_root/speechd.log" &&
      timeout 5 spd-say -o dummy -x -w '<speak>ready</speak>' > "$test_root/speech-client.log" 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 10 ] || ! kill -0 "$speechd_pid" 2>/dev/null; then fail 'private dummy speech module did not finish'; fi
  sleep 0.1
done

webfence > "$test_root/webfence.log" 2>&1 &
app_pid=$!
/usr/bin/python3 /opt/webfence-test/test-atspi.py "$language" > "$test_root/atspi.log" 2>&1 || fail 'AT-SPI baseline failed'

orca --enable=speech --debug-file="$test_root/orca.log" > "$test_root/orca.stdout" 2>&1 &
orca_pid=$!
attempt=0
until [ -f "$test_root/orca.log" ] && grep -Fq 'SPEECH DISPATCHER: Speaking' "$test_root/orca.log"; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 100 ] || ! kill -0 "$orca_pid" 2>/dev/null; then fail 'Orca did not start speech'; fi
  sleep 0.1
done

xdotool search --onlyvisible --name WebFence windowactivate --sync || fail 'window activation failed'
xdotool key F6 || fail 'F6 failed'
sleep 1
xdotool key Home || fail 'Home failed'
sleep 1
xdotool key Down || fail 'Down failed'

speech_command_contains() {
  grep -F 'SPEECH DISPATCHER: Speaking' "$test_root/orca.log" | grep -Fq "$1"
}
attempt=0
until speech_command_contains 'DEMO-00002.' && speech_command_contains "$severity." &&
      speech_command_contains 'https://example.invalid/catalog/00002.' &&
      speech_command_contains "$status."; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 100 ]; then fail 'localized row speech commands missing'; fi
  sleep 0.1
done
if [ -s "$test_root/webfence.log" ]; then fail 'application logged an error'; fi
if ! kill -0 "$orca_pid" 2>/dev/null; then fail 'Orca exited before verification'; fi

printf 'PASS Orca %s synthetic row speech commands (dummy output; no audible claim)\n' "$language"
