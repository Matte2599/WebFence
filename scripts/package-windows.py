#!/usr/bin/env python3
"""Build a development ZIP from an existing WebFence UCRT64 executable."""
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile


def run(*args):
    return subprocess.check_output([str(a) for a in args], text=True).strip()


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def package(prefix, executable):
    if sys.platform != "win32":
        raise ValueError("Native Windows UCRT64 Python required")
    prefix, executable = Path(prefix).resolve(), Path(executable).resolve()
    repo = Path(__file__).resolve().parent.parent
    dist = repo / "dist"
    dist.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".webfence-windows-", dir=dist) as temporary:
        stage = Path(temporary)
        bundle = stage / "WebFence"
        bundle.mkdir()
        shutil.copyfile(executable, bundle / "webfence.exe")
        subprocess.run([str(prefix / "bin/windeployqt6.exe"), "--release", "--no-translations",
                        "--no-system-d3d-compiler", "--no-opengl-sw", str(bundle / "webfence.exe")], check=True)
        sources = {}
        for source in prefix.rglob("*.dll"):
            sources.setdefault(source.name.lower(), []).append(source)

        def source_for(name):
            matches = sources.get(name.lower(), [])
            if len(matches) != 1:
                raise ValueError("Missing or ambiguous UCRT64 DLL: " + name)
            return matches[0]

        (bundle / "platforms").mkdir(exist_ok=True)
        shutil.copyfile(source_for("qoffscreen.dll"), bundle / "platforms/qoffscreen.dll")
        (bundle / "qt.conf").write_text("[Paths]\nPrefix=.\nPlugins=.\n", encoding="utf-8")
        queue = [bundle / "webfence.exe", *bundle.rglob("*.dll")]
        seen, system_imports = set(), set()
        system = Path(os.environ["SystemRoot"]) / "System32"
        while queue:
            binary = queue.pop()
            if binary in seen:
                continue
            seen.add(binary)
            description = run(prefix / "bin/objdump.exe", "-p", binary)
            if "file format pei-x86-64" not in description:
                raise ValueError("Expected an x86-64 PE file: " + str(binary))
            if binary.name == "webfence.exe" and "(Windows GUI)" not in description:
                raise ValueError("Build the desktop executable with -ldflags=-H=windowsgui")
            for name in re.findall(r"DLL Name:\s*(\S+)", description):
                if not re.fullmatch(r"[a-zA-Z0-9_.+-]+\.dll", name, flags=re.I):
                    raise ValueError("Invalid imported DLL name")
                lower = name.lower()
                if lower.startswith(("api-ms-win-", "ext-ms-win-")) or (system / name).is_file():
                    system_imports.add(lower)
                    continue
                target = bundle / name
                if not target.exists():
                    shutil.copyfile(source_for(name), target)
                queue.append(target)

        notices = bundle / "notices"
        subprocess.run([sys.executable, str(repo / "scripts/package-go-notices.py"),
                        str(bundle / "webfence.exe"), str(notices / "go")], check=True)
        owners, files = {}, []
        for binary in sorted(bundle.rglob("*.dll")):
            source = source_for(binary.name)
            owner = run("pacman", "-Qoq", run("cygpath", "-u", source))
            if owner not in owners:
                metadata = run("pacman", "-Qi", owner)
                licenses = []
                for line in run("pacman", "-Ql", owner).splitlines():
                    _, path = line.split(" ", 1)
                    if "/share/licenses/" in path and not path.endswith("/"):
                        local = Path(run("cygpath", "-w", path))
                        if local.is_file():
                            relative = path.split("/share/licenses/", 1)[1]
                            target = notices / "native" / owner / relative
                            target.parent.mkdir(parents=True, exist_ok=True)
                            shutil.copyfile(local, target)
                            licenses.append(str(target.relative_to(bundle)))
                if not licenses:
                    raise ValueError("Missing packaged license notices for " + owner)
                owners[owner] = {"pacman_metadata": metadata, "license_files": licenses}
            files.append({"path": binary.relative_to(bundle).as_posix(), "sha256": digest(binary),
                          "source_sha256": digest(source), "package": owner})
        (bundle / "native-build.json").write_text(json.dumps({"schema": 1, "files": files,
            "packages": owners, "system_imports": sorted(system_imports),
            "scope": "PE import closure and installed MSYS2 notices; not full source compliance or dynamic-load coverage"}, indent=2) + "\n")
        for name in ["LICENSE", "README.md", "README.en.md"]:
            shutil.copyfile(repo / name, bundle / name)
        archive = shutil.make_archive(str(stage / "webfence-windows-amd64"), "zip", stage, "WebFence")
        final = dist / "webfence-windows-amd64.zip"
        os.replace(archive, final)
        print(str(final), digest(final))


if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit("Usage: package-windows.py UCRT64_PREFIX WEBFENCE_EXE")
    package(*sys.argv[1:])
