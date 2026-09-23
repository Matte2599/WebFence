#!/usr/bin/env python3
"""Record the Go modules actually linked into a trusted, locally built binary."""
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys


def output(*args):
    return subprocess.check_output(args, text=True).strip()


def collect(binary, destination):
    binary, destination = Path(binary), Path(destination)
    destination.mkdir(parents=True, exist_ok=True)
    build = json.loads(output("go", "version", "-m", "-json", str(binary)))
    if build["Path"] != "github.com/Matte2599/WebFence/cmd/webfence":
        raise ValueError("Not a WebFence desktop executable")
    if build["GoVersion"] != output("go", "env", "GOVERSION"):
        raise ValueError("Use the binary's Go toolchain to collect its notices")
    modules = []
    for dep in build.get("Deps", []):
        if dep.get("Replace"):
            raise ValueError("Replaced module requires explicit notice/source review")
        module = json.loads(output("go", "list", "-m", "-json", dep["Path"]))
        if module["Version"] != dep["Version"] or module.get("Sum") != dep.get("Sum"):
            raise ValueError("Binary dependency differs from the module graph")
        source = Path(module["Dir"])
        notices = [p for p in source.iterdir() if p.is_file() and
                   p.name.upper().startswith(("LICENSE", "COPYING", "NOTICE", "COPYRIGHT"))]
        if not notices:
            raise ValueError("No top-level notices found for " + dep["Path"])
        folder = destination / dep["Path"] / dep["Version"]
        folder.mkdir(parents=True, exist_ok=True)
        for p in notices:
            shutil.copyfile(p, folder / p.name)
        modules.append({"path": dep["Path"], "version": dep["Version"], "sum": dep["Sum"]})
    go_root = Path(output("go", "env", "GOROOT"))
    go_license = go_root / "LICENSE"
    if not go_license.is_file() and go_root.name == "libexec":
        go_license = go_root.parent / "LICENSE"  # Homebrew Go layout.
    shutil.copyfile(go_license, destination / "Go-LICENSE")
    record = {"schema": 1, "go_version": build["GoVersion"], "linked_modules": modules,
              "executable_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
              "scope": "Go module provenance and top-level notices; not a complete native SBOM"}
    (destination / "go-build.json").write_text(json.dumps(record, indent=2) + "\n")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit("Usage: package-go-notices.py EXECUTABLE OUTPUT_DIRECTORY")
    collect(*sys.argv[1:])
