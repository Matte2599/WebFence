#!/usr/bin/env python3
"""Synthetic startup trial, not a desktop launcher. See ADR-022 (IT/EN)."""
import argparse
import contextlib
import ctypes
import fcntl
import hashlib
import http.server
import json
import os
from pathlib import Path
import platform
import plistlib
import select
import shutil
import signal
import subprocess
import sys
import tempfile
import threading
import time
import urllib.request
import zipfile

VERSION = "154.0.8037.57"
ARCHIVE_SHA256 = "9ba4d8a9732bd7e431ce9009d1bbe1ccf7f8c5de2a9781054f500a9fe898124d"
BUNDLE_ID = "org.webfence.m3-sandbox-probe"
MAX_FRAME = 65536


class CDP:
    def __init__(self, read_fd, write_fd):
        self.read_fd, self.write_fd = read_fd, write_fd
        self.buffer = b""
        self.sequence = 0
        self.deadline = time.monotonic() + 20

    def call(self, method, params=None, session=None):
        self.sequence += 1
        request = {"id": self.sequence, "method": method, "params": params or {}}
        if session:
            request["sessionId"] = session
        wire = json.dumps(request).encode() + b"\0"
        if len(wire) > MAX_FRAME:
            raise RuntimeError("CDP request limit")
        while wire:
            wire = wire[os.write(self.write_fd, wire):]
        for _ in range(128):
            while b"\0" not in self.buffer:
                remaining = self.deadline - time.monotonic()
                if remaining <= 0 or not select.select([self.read_fd], [], [], remaining)[0]:
                    raise RuntimeError("CDP deadline")
                chunk = os.read(self.read_fd, min(4096, MAX_FRAME + 1 - len(self.buffer)))
                if not chunk:
                    raise RuntimeError("CDP EOF")
                self.buffer += chunk
                if len(self.buffer.split(b"\0", 1)[0]) > MAX_FRAME:
                    raise RuntimeError("CDP frame limit")
            frame, self.buffer = self.buffer.split(b"\0", 1)
            response = json.loads(frame)
            if not isinstance(response, dict):
                raise RuntimeError("CDP invalid response")
            if response.get("id") == self.sequence:
                if response.get("error") or not isinstance(response.get("result"), dict):
                    raise RuntimeError("CDP command rejected")
                if session and response.get("sessionId") != session:
                    raise RuntimeError("CDP session mismatch")
                return response["result"]
        raise RuntimeError("CDP event limit")


def value(result):
    if result.get("exceptionDetails"):
        raise RuntimeError("JavaScript exception")
    remote = result.get("result", {})
    if remote.get("type") != "string":
        raise RuntimeError("JavaScript result type")
    return remote.get("value")


def page(cdp):
    product = cdp.call("Browser.getVersion").get("product", "")
    if product != "HeadlessChrome/" + VERSION:
        raise RuntimeError("unexpected runtime version")
    target = cdp.call("Target.createTarget", {"url": "about:blank"}).get("targetId")
    if not target:
        raise RuntimeError("missing target")
    session = cdp.call("Target.attachToTarget", {"targetId": target, "flatten": True}).get("sessionId")
    if not session:
        raise RuntimeError("missing session")
    result = cdp.call("Runtime.evaluate", {
        "expression": "document.body.innerHTML='<button>local fixture</button>';document.body.textContent",
        "returnByValue": True,
    }, session)
    if value(result) != "local fixture":
        raise RuntimeError("DOM control failed")
    return session


def sign(path, entitlements, root, label):
    ent = root / (label + ".plist")
    ent.write_bytes(plistlib.dumps(entitlements))
    subprocess.run(["/usr/bin/codesign", "--force", "--sign", "-", "--entitlements", str(ent), str(path)],
                   check=True, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, timeout=15)
    subprocess.run(["/usr/bin/codesign", "--verify", "--strict", str(path)],
                   check=True, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, timeout=15)


def prepare_bundle(source, root):
    # .bundle preserves standalone Chromium path handling. A fabricated .app
    # makes it look for a Chromium application/framework and its helper bundles.
    bundle = root / "SandboxProbe.bundle"
    mac, resources = bundle / "Contents/MacOS", bundle / "Contents/Resources"
    mac.mkdir(parents=True)
    resources.mkdir()
    for item in source.iterdir():
        destination = mac if item.name == "chrome-headless-shell" or item.suffix == ".dylib" else resources
        if item.is_dir():
            shutil.copytree(item, destination / item.name)
        else:
            shutil.copy2(item, destination / item.name)
    for item in resources.iterdir():
        (mac / item.name).symlink_to("../Resources/" + item.name)
    (bundle / "Contents/Info.plist").write_bytes(plistlib.dumps({
        "CFBundleIdentifier": BUNDLE_ID, "CFBundleExecutable": "chrome-headless-shell",
        "CFBundlePackageType": "APPL", "CFBundleName": "WebFence M3 Probe",
        "CFBundleVersion": "1", "LSBackgroundOnly": True,
    }))
    helper = resources / "Helper"
    shutil.copytree(source, helper)
    sign(helper / "chrome-headless-shell", {
        "com.apple.security.app-sandbox": True, "com.apple.security.inherit": True,
    }, root, "inherit")
    return bundle, mac / "chrome-headless-shell", helper / "chrome-headless-shell"


def worker(arguments):
    # Trusted pre-exec gate reserves the exact browser PID before signing.
    gate, read_fd, write_fd = map(int, arguments[:3])
    if os.read(gate, 1) != b"1":
        raise RuntimeError("launch gate closed")
    os.close(gate)
    read_copy = fcntl.fcntl(read_fd, fcntl.F_DUPFD, 10)
    write_copy = fcntl.fcntl(write_fd, fcntl.F_DUPFD, 10)
    os.dup2(read_copy, 3, inheritable=True)
    os.dup2(write_copy, 4, inheritable=True)
    for fd in {read_copy, write_copy, read_fd, write_fd} - {3, 4}:
        os.close(fd)
    os.execve(arguments[3], arguments[3:], dict(os.environ))


@contextlib.contextmanager
def launch(executable, profile, root, bundle=None, helper=None):
    read_child, write_parent = os.pipe()
    read_parent, write_child = os.pipe()
    gate_read, gate_write = os.pipe()
    process = None
    diagnostics = bytearray()
    def drain(pipe):
        while chunk := pipe.read(4096):
            diagnostics.extend(chunk[:max(0, 8192 - len(diagnostics))])
    try:
        args = [str(executable), "--remote-debugging-pipe", "--disable-crash-reporter",
                "--disable-background-networking", "--no-first-run", "--user-data-dir=" + str(profile)]
        if bundle:
            # Only this sealed synthetic App Sandbox trial uses this flag. It
            # avoids a forbidden second Seatbelt initialization; the OS sandbox
            # is checked below for browser, renderer, GPU and network service.
            args += ["--no-sandbox", "--browser-subprocess-path=" + str(helper)]
        args += ["about:blank"]
        environment = {"HOME": str(Path.home()), "PATH": "/usr/bin:/bin", "LANG": "en_US.UTF-8"}
        process = subprocess.Popen([sys.executable, str(Path(__file__).resolve()), "--worker",
                                    str(gate_read), str(read_child), str(write_child), *args],
                                   pass_fds=(gate_read, read_child, write_child), start_new_session=True,
                                   stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                                   stderr=subprocess.PIPE, env=environment)
        drainer = threading.Thread(target=drain, args=(process.stderr,), daemon=True)
        drainer.start()
        if bundle:
            channel = "org.chromium.Chromium.MachPortRendezvousServer." + str(process.pid)
            sign(bundle, {"com.apple.security.app-sandbox": True,
                          "com.apple.security.temporary-exception.mach-register.global-name": [channel],
                          "com.apple.security.temporary-exception.mach-lookup.global-name": [channel]}, root, "main")
        os.write(gate_write, b"1")
        for fd in (read_child, write_child, gate_read, gate_write):
            os.close(fd)
        read_child = write_child = gate_read = gate_write = -1
        yield CDP(read_parent, write_parent)
    finally:
        if process:
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.wait(timeout=5)
            drainer.join(timeout=2)
            process.stderr.close()
        for fd in (read_child, write_child, gate_read, gate_write, read_parent, write_parent):
            if fd >= 0:
                os.close(fd)
        if sys.exc_info()[0] and diagnostics:
            print("Bounded runtime diagnostic:\n" + diagnostics.decode(errors="replace"), file=sys.stderr)


def sandboxed_processes(cdp):
    entries = cdp.call("SystemInfo.getProcessInfo").get("processInfo", [])
    kinds = {entry.get("type") for entry in entries}
    if not {"browser", "renderer", "GPU", "network.mojom.NetworkService"}.issubset(kinds):
        raise RuntimeError("missing browser descendants")
    check = ctypes.CDLL("/usr/lib/libsandbox.dylib").sandbox_check
    check.restype = ctypes.c_int
    for entry in entries:
        if check(ctypes.c_int(int(entry["id"])), None, ctypes.c_int(0)) != 1:
            raise RuntimeError("process is not OS sandboxed: " + entry["type"])


def trial(source, root):
    hits = []
    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            hits.append(self.path)
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"<p>local server fixture</p>")
        def log_message(self, *_):
            pass
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        url = "http://127.0.0.1:" + str(server.server_port) + "/"
        outside = root / "outside.html"
        outside.write_text("<p>outside fixture</p>")
        # Actual browser positive control, same runtime, separate profile.
        with launch(source / "chrome-headless-shell", root / "control", root) as cdp:
            session = page(cdp)
            if cdp.call("Page.navigate", {"url": url}, session).get("errorText") or not hits:
                raise RuntimeError("HTTP positive control failed")
            if cdp.call("Page.navigate", {"url": outside.as_uri()}, session).get("errorText"):
                raise RuntimeError("file positive control failed")
        hits.clear()
        bundle, executable, helper = prepare_bundle(source, root)
        container_tmp = Path.home() / "Library/Containers" / BUNDLE_ID / "Data/tmp"
        container_tmp.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(prefix="wf-headless-", dir=container_tmp) as profile_dir:
            profile = Path(profile_dir)
            inside = profile / "inside.html"
            inside.write_text("<p>inside fixture</p>")
            with launch(executable, profile, root, bundle, helper) as cdp:
                session = page(cdp)
                sandboxed_processes(cdp)
                for target in (url, outside.as_uri()):
                    if cdp.call("Page.navigate", {"url": target}, session).get("errorText") != "net::ERR_ACCESS_DENIED":
                        raise RuntimeError("direct access not denied")
                if hits:
                    raise RuntimeError("confined browser contacted listener")
                if cdp.call("Page.navigate", {"url": inside.as_uri()}, session).get("errorText"):
                    raise RuntimeError("private file control failed")
                # DOM becomes ready asynchronously after Page.navigate.
                for _ in range(20):
                    result = cdp.call("Runtime.evaluate", {
                        "expression": "document.readyState==='complete'?document.body.textContent:''",
                        "returnByValue": True}, session)
                    if value(result) == "inside fixture":
                        break
                    time.sleep(0.05)
                else:
                    raise RuntimeError("private fixture DOM failed")
                sandboxed_processes(cdp)
        print("PASS macOS App Sandbox startup: CDP/DOM, sandboxed descendants, direct HTTP/file denied, private file readable")
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--runtime-archive", required=True, type=Path)
    args = parser.parse_args()
    if sys.platform != "darwin" or platform.machine() != "arm64":
        raise RuntimeError("this pinned trial requires macOS ARM64")
    if hashlib.sha256(args.runtime_archive.read_bytes()).hexdigest() != ARCHIVE_SHA256:
        raise RuntimeError("runtime checksum mismatch")
    with tempfile.TemporaryDirectory(prefix="wf-m3-headless-") as directory:
        root = Path(directory).resolve()
        with zipfile.ZipFile(args.runtime_archive) as archive:
            archive.extractall(root)
        source = root / "chrome-headless-shell-mac-arm64"
        for executable in source.glob("*"):
            if executable.name == "chrome-headless-shell" or executable.suffix == ".dylib":
                executable.chmod(0o755)
        trial(source, root)


if __name__ == "__main__":
    if sys.argv[1:2] == ["--worker"]:
        worker(sys.argv[2:])
    else:
        main()
