#!/usr/bin/env python3
"""Bounded macOS/Linux trial; keep raw Qt events and current process RSS."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("executable", type=Path)
    parser.add_argument("output", type=Path, help="new directory; never overwrite an earlier trial")
    parser.add_argument("--seconds", type=int, default=1800)
    args = parser.parse_args()
    if platform.system() not in {"Darwin", "Linux"}:
        parser.error("RSS collector supports macOS/Linux only")
    if not 10 <= args.seconds <= 3600:
        parser.error("duration must be between 10 and 3600 seconds")
    executable = args.executable.resolve(strict=True)
    args.output.mkdir(parents=True, exist_ok=False)
    metadata = {"schema": 1, "executable_sha256": hashlib.sha256(executable.read_bytes()).hexdigest(),
                "system": platform.system(), "release": platform.release(), "arch": platform.machine(),
                "logical_cpus": os.cpu_count(), "requested_seconds": args.seconds,
                "qt_platform_override": os.environ.get("QT_QPA_PLATFORM"),
                "qt_scale_override": os.environ.get("QT_SCALE_FACTOR"),
                "started_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}
    (args.output / "metadata.json").write_text(json.dumps(metadata, indent=2) + "\n")
    started = time.monotonic()
    previous_sample = started
    samples = []
    completed = False
    child = None
    try:
        with (args.output / "events.jsonl").open("w") as stdout, (args.output / "stderr.log").open("w") as stderr, (args.output / "rss.jsonl").open("w") as rss:
            child = subprocess.Popen([str(executable), f"--soak-test={args.seconds}s"], stdout=stdout, stderr=stderr)
            while child.poll() is None:
                now = time.monotonic()
                if now - started > args.seconds + 30:
                    raise RuntimeError("trial exceeded its external deadline (possible event-loop hang)")
                if now - previous_sample > 30:
                    raise RuntimeError("sampling gap exceeds 30 seconds; trial cannot establish continuous activity")
                value = subprocess.run(["ps", "-o", "rss=", "-p", str(child.pid)], capture_output=True, text=True, timeout=3)
                if value.returncode == 0 and value.stdout.strip().isdigit():
                    sample = {"elapsed_seconds": now - started, "rss_bytes": int(value.stdout.strip()) * 1024}
                    samples.append(sample)
                    rss.write(json.dumps(sample) + "\n")
                    rss.flush()
                elif child.poll() is None:
                    raise RuntimeError("unable to measure current process RSS")
                previous_sample = now
                try:
                    child.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    pass
        events = []
        for line in (args.output / "events.jsonl").read_text().splitlines():
            if line.startswith("{"):
                events.append(json.loads(line))
        passed = [event for event in events if event.get("event") == "passed"]
        if child.returncode != 0 or len(passed) != 1 or not samples:
            raise RuntimeError("trial failed or did not record successful completion")
        end = passed[0]
        if end["requested_seconds"] != args.seconds or end["elapsed_seconds"] < args.seconds:
            raise RuntimeError("trial duration does not match the request")
        summary = {"exit_code": child.returncode, "elapsed_seconds": time.monotonic() - started,
                   "cycles": end["cycles"], "qt_platform": end["qt_platform"],
                   "rss_first_bytes": samples[0]["rss_bytes"], "rss_last_bytes": samples[-1]["rss_bytes"],
                   "rss_peak_sampled_bytes": max(s["rss_bytes"] for s in samples),
                   "samples": len(samples), "scope": "workload completed; memory trend requires review; not an assistive trial"}
        (args.output / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
        completed = True
        print(json.dumps(summary), flush=True)
    finally:
        if child is not None and child.poll() is None:
            child.terminate()
            try:
                child.wait(timeout=3)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait()
        if not completed:
            (args.output / "INCOMPLETE").write_text("No successful trial; inspect raw logs.\n")


if __name__ == "__main__":
    main()
