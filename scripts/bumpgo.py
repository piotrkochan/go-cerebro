#!/usr/bin/env python3
"""Interactively update the pinned Go toolchain and Docker images."""

import json
import re
import subprocess
import sys
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


ROOT = Path(__file__).resolve().parent.parent
DOCKERFILES = (ROOT / "Dockerfile", ROOT / "Dockerfile.dev")
RELEASES_URL = "https://go.dev/dl/?mode=json&include=all"
TOKEN_URL = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/golang:pull"
ACCEPT = ", ".join((
    "application/vnd.oci.image.index.v1+json",
    "application/vnd.docker.distribution.manifest.list.v2+json",
))
VERSION = re.compile(r"^(\d+)\.(\d+)\.(\d+)$")
GO_IMAGE = re.compile(r"golang:(\d+\.\d+\.\d+)-alpine@sha256:[0-9a-f]{64}")


def version_tuple(value):
    match = VERSION.fullmatch(value)
    if not match:
        raise ValueError(f"invalid Go version: {value}")
    return tuple(map(int, match.groups()))


def single_match(pattern, content, label):
    if isinstance(pattern, re.Pattern):
        matches = list(pattern.finditer(content))
    else:
        matches = list(re.finditer(pattern, content, re.MULTILINE))
    if len(matches) != 1:
        raise ValueError(f"expected exactly one {label}, found {len(matches)}")
    return matches[0]


def current_versions(root=ROOT):
    go_mod = (root / "go.mod").read_text()
    minimum = single_match(r"^go (\d+\.\d+\.\d+)$", go_mod, "go directive").group(1)
    toolchains = list(re.finditer(r"^toolchain go(\d+\.\d+\.\d+)$", go_mod, re.MULTILINE))
    if len(toolchains) > 1:
        raise ValueError("go.mod contains more than one toolchain directive")
    toolchain = toolchains[0].group(1) if toolchains else minimum
    if version_tuple(minimum) > version_tuple(toolchain):
        raise ValueError("go.mod minimum is newer than its toolchain")
    images = [single_match(GO_IMAGE, path.read_text(), str(path)).group(0) for path in (root / "Dockerfile", root / "Dockerfile.dev")]
    if images[0] != images[1] or not images[0].startswith(f"golang:{toolchain}-alpine@"):
        raise ValueError("Go images in Dockerfiles must match the go.mod toolchain and each other")
    return minimum, toolchain, images[0]


def fetch_json(url):
    with urlopen(Request(url, headers={"User-Agent": "go-cerebro-bumpgo/1"}), timeout=20) as response:
        return json.load(response)


def stable_newer_versions(current):
    versions = set()
    for release in fetch_json(RELEASES_URL):
        if not release.get("stable", False):
            continue
        value = release.get("version", "")
        if not value.startswith("go") or not VERSION.fullmatch(value[2:]):
            continue
        if version_tuple(value[2:]) > version_tuple(current):
            versions.add(value[2:])
    return sorted(versions, key=version_tuple, reverse=True)


def image_digest(version, token):
    tag = f"{version}-alpine"
    request = Request(
        f"https://registry-1.docker.io/v2/library/golang/manifests/{tag}",
        method="HEAD",
        headers={"Authorization": f"Bearer {token}", "Accept": ACCEPT},
    )
    try:
        with urlopen(request, timeout=20) as response:
            digest = response.headers.get("Docker-Content-Digest", "")
            if not re.fullmatch(r"sha256:[0-9a-f]{64}", digest):
                raise ValueError(f"registry returned no valid digest for golang:{tag}")
            return digest
    except HTTPError as error:
        if error.code == 404:
            return None
        raise


def available_updates(current):
    releases = stable_newer_versions(current)
    if not releases:
        return []
    token = fetch_json(TOKEN_URL)["token"]
    updates = []
    for version in releases:
        digest = image_digest(version, token)
        if digest:
            updates.append((version, digest))
        if len(updates) == 3:
            break
    return updates


def choose_update(current, updates):
    print(f"Current Go toolchain: {current}")
    if not updates:
        print("No newer stable Go release with a golang:<version>-alpine image is available.")
        return None
    for number, (version, digest) in enumerate(updates, 1):
        print(f"{number}) Go {version}  golang:{version}-alpine@{digest[:19]}…")
    while True:
        try:
            answer = input(f"Choose 1-{len(updates)} (Enter = 1, q = cancel): ").strip().lower()
        except EOFError:
            return None
        if answer in ("q", "0"):
            return None
        if answer == "":
            return updates[0]
        if answer.isdigit() and 1 <= int(answer) <= len(updates):
            return updates[int(answer) - 1]
        print("Invalid choice.")


def planned_files(root, minimum, current, old_image, version, digest):
    new_image = f"golang:{version}-alpine@{digest}"
    old_comment = f"# Go {current}; update with make bumpgo."
    new_comment = f"# Go {version}; update with make bumpgo."
    changes = {}
    for name in ("Dockerfile", "Dockerfile.dev"):
        path = root / name
        content = path.read_text()
        if content.count(old_image) != 1 or content.count(old_comment) != 1:
            raise ValueError(f"{name} must contain one matching Go image and version comment")
        changes[path] = content.replace(old_image, new_image).replace(old_comment, new_comment)

    path = root / "go.mod"
    content = path.read_text()
    new_minimum = minimum
    if version_tuple(version)[:2] > version_tuple(minimum)[:2]:
        major, minor, _ = version_tuple(version)
        new_minimum = f"{major}.{minor}.0"
    content, count = re.subn(rf"^go {re.escape(minimum)}$", f"go {new_minimum}", content, count=1, flags=re.MULTILINE)
    if count != 1:
        raise ValueError("go directive changed while preparing update")
    content, count = re.subn(rf"^toolchain go{re.escape(current)}$", f"toolchain go{version}", content, count=1, flags=re.MULTILINE)
    if count == 0:
        content = content.replace(f"go {new_minimum}\n", f"go {new_minimum}\n\ntoolchain go{version}\n", 1)
    changes[path] = content
    return changes


def apply_update(root, minimum, current, old_image, version, digest):
    changes = planned_files(root, minimum, current, old_image, version, digest)
    sum_path = root / "go.sum"
    originals = {path: path.read_bytes() for path in changes}
    originals[sum_path] = sum_path.read_bytes() if sum_path.exists() else None
    try:
        for path, content in changes.items():
            path.write_text(content)
        subprocess.run(["go", "mod", "tidy"], cwd=root, check=True)
        _, final_toolchain, _ = current_versions(root)
        if final_toolchain != version:
            raise ValueError("go mod tidy changed the selected Go toolchain")
    except (OSError, ValueError, KeyboardInterrupt, subprocess.CalledProcessError):
        for path, content in originals.items():
            if content is None:
                path.unlink(missing_ok=True)
            else:
                path.write_bytes(content)
        raise


def main():
    try:
        minimum, current, old_image = current_versions()
        updates = available_updates(current)
        selected = choose_update(current, updates)
        if selected is None:
            print("No files changed.")
            return 0
        version, digest = selected
        apply_update(ROOT, minimum, current, old_image, version, digest)
        print(f"Updated Go to {version} and pinned both Docker images to {digest}.")
        print("go mod tidy completed; review go.mod and go.sum before committing.")
        return 0
    except (ValueError, KeyError, OSError, URLError, subprocess.CalledProcessError) as error:
        print(f"bumpgo: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
