#!/usr/bin/env python3
"""Synthetic startup trial, not a desktop launcher. See ADR-022 (IT/EN)."""
import argparse
import base64
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
        self.handler = None
        self.waiting = {}
        self.replies = {}
        self.events = 0

    def call(self, method, params=None, session=None):
        if len(self.waiting) >= 8:
            raise RuntimeError("CDP nesting limit")
        self.sequence += 1
        request_id = self.sequence
        self.waiting[request_id] = session
        try:
            return self._call(request_id, method, params, session)
        finally:
            self.waiting.pop(request_id)

    def _call(self, request_id, method, params, session):
        request = {"id": request_id, "method": method, "params": params or {}}
        if session:
            request["sessionId"] = session
        wire = json.dumps(request).encode() + b"\0"
        if len(wire) > MAX_FRAME:
            raise RuntimeError("CDP request limit")
        while wire:
            wire = wire[os.write(self.write_fd, wire):]
        for _ in range(128):
            if request_id in self.replies:
                response = self.replies.pop(request_id)
                if response.get("error") or not isinstance(response.get("result"), dict):
                    raise RuntimeError("CDP command rejected")
                return response["result"]
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
            response_id = response.get("id")
            if response_id is not None:
                if type(response_id) is not int or response_id not in self.waiting:
                    raise RuntimeError("CDP unexpected response id")
                if response.get("sessionId") != self.waiting[response_id]:
                    raise RuntimeError("CDP session mismatch")
                if response_id in self.replies:
                    raise RuntimeError("CDP duplicate response")
                self.replies[response_id] = response
            else:
                self.events += 1
                if self.events > 1024:
                    raise RuntimeError("CDP global event limit")
                if self.handler:
                    self.handler(response)
        raise RuntimeError("CDP event limit")


class BrokerPipe:
    def __init__(self, process):
        self.process = process
        self.buffer = b""

    def receive(self):
        deadline = time.monotonic() + 6
        while b"\n" not in self.buffer:
            remaining = deadline - time.monotonic()
            if remaining <= 0 or not select.select([self.process.stdout], [], [], remaining)[0]:
                raise RuntimeError("broker pipe deadline")
            chunk = os.read(self.process.stdout.fileno(), min(4096, MAX_FRAME + 1 - len(self.buffer)))
            if not chunk:
                raise RuntimeError("broker pipe EOF")
            self.buffer += chunk
            if len(self.buffer.split(b"\n", 1)[0]) > MAX_FRAME:
                raise RuntimeError("broker pipe frame limit")
        data, self.buffer = self.buffer.split(b"\n", 1)
        reply = json.loads(data)
        if not isinstance(reply, dict):
            raise RuntimeError("broker pipe reply type")
        return reply

    def call(self, request):
        frame = json.dumps(request).encode() + b"\n"
        if len(frame) > 16384:
            raise RuntimeError("broker request frame limit")
        self.process.stdin.write(frame)
        self.process.stdin.flush()
        return self.receive()


def mediated(cdp, session, executable):
    for scheme in ("http", "https"):
        process = subprocess.Popen([str(executable), scheme], stdin=subprocess.PIPE,
                                   stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                                   env={"PATH": "/usr/bin:/bin"}, bufsize=0)
        broker = BrokerPipe(process)
        try:
            origin = broker.receive().get("origin", "")
            if not origin.startswith(scheme + "://site.test:"):
                raise RuntimeError("invalid synthetic broker origin")
            requests = 0
            def paused(event):
                nonlocal requests
                if event.get("method") != "Fetch.requestPaused":
                    return
                requests += 1
                if requests > 16 or event.get("sessionId") != session:
                    raise RuntimeError("CDP intercepted request session/budget")
                params = event.get("params", {})
                request_id = params.get("requestId")
                if not isinstance(request_id, str) or not 0 < len(request_id) <= 1024:
                    raise RuntimeError("invalid intercepted request id")
                request = params.get("request", {})
                if request.get("hasPostData") or request.get("postData"):
                    cdp.call("Fetch.failRequest", {"requestId": request_id, "errorReason": "BlockedByClient"}, session)
                    return
                reply = broker.call({"method": request.get("method", ""), "url": request.get("url", ""),
                                     "type": params.get("resourceType", "")})
                status, body, header = reply.get("status"), reply.get("body"), reply.get("header")
                if not isinstance(status, int) or not 100 <= status <= 599 or not isinstance(body, str) or not isinstance(header, dict):
                    raise RuntimeError("invalid mediated response")
                if len(base64.b64decode(body, validate=True)) > 16384:
                    raise RuntimeError("mediated response body limit")
                entries = [{"name": name, "value": value} for name, values in header.items() for value in values]
                cdp.call("Fetch.fulfillRequest", {"requestId": request_id, "responseCode": status,
                         "responseHeaders": entries, "body": body}, session)
            cdp.handler = paused
            cdp.call("Fetch.enable", {"patterns": [{"urlPattern": "*", "requestStage": "Request"}]}, session)
            if cdp.call("Page.navigate", {"url": origin + "/app/"}, session).get("errorText"):
                raise RuntimeError("mediated navigation failed")
            want = origin + "|" + str(scheme == "https").lower() + "||synthetic|502|502|403"
            for _ in range(100):
                result = value(cdp.call("Runtime.evaluate", {"expression": "window.wfResult||''", "returnByValue": True}, session))
                if result == want:
                    break
                if result == "failed":
                    raise RuntimeError("mediated page script failed")
                time.sleep(0.02)
            else:
                raise RuntimeError("mediated page result deadline")
            cdp.call("Fetch.disable", {}, session)
            cdp.handler = None
            if broker.call({"finish": True}) != {"verified": True} or process.wait(timeout=2) != 0:
                raise RuntimeError("broker fixture invariants failed")
            print("PASS macOS mediated " + scheme.upper() + ": exact origin, script/fetch, cookie redaction, scope/redirect and in-flight revocation")
        finally:
            cdp.handler = None
            if process.poll() is None:
                process.kill()
            process.wait(timeout=5)
            process.stdin.close()
            process.stdout.close()


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


def trial(source, root, broker_executable=None):
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
                if broker_executable:
                    mediated(cdp, session, broker_executable)
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
    parser.add_argument("--broker-executable", type=Path)
    args = parser.parse_args()
    if sys.platform != "darwin" or platform.machine() != "arm64":
        raise RuntimeError("this pinned trial requires macOS ARM64")
    if hashlib.sha256(args.runtime_archive.read_bytes()).hexdigest() != ARCHIVE_SHA256:
        raise RuntimeError("runtime checksum mismatch")
    if args.broker_executable and (not args.broker_executable.is_absolute() or not args.broker_executable.is_file()):
        raise RuntimeError("absolute compiled fixture broker path required")
    with tempfile.TemporaryDirectory(prefix="wf-m3-headless-") as directory:
        root = Path(directory).resolve()
        with zipfile.ZipFile(args.runtime_archive) as archive:
            archive.extractall(root)
        source = root / "chrome-headless-shell-mac-arm64"
        for executable in source.glob("*"):
            if executable.name == "chrome-headless-shell" or executable.suffix == ".dylib":
                executable.chmod(0o755)
        trial(source, root, args.broker_executable)


if __name__ == "__main__":
    if sys.argv[1:2] == ["--worker"]:
        worker(sys.argv[2:])
    else:
        main()
