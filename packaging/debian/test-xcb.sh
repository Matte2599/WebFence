#!/bin/sh
# Disposable Xvfb desktop only. No user session or personal window manager.
set -eu
openbox > /tmp/webfence-openbox.log 2>&1 &
wm_pid=$!
trap 'kill "$wm_pid" 2>/dev/null || true' EXIT
attempt=0
until xprop -root _NET_SUPPORTING_WM_CHECK | grep -q 'window id #'; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 50 ]; then cat /tmp/webfence-openbox.log; exit 1; fi
  sleep 0.1
done
QT_QPA_PLATFORM=xcb webfence --self-test
