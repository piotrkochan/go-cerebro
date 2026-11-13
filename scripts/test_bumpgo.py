import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import bumpgo


class BumpGoTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.old_image = "golang:1.27.1-alpine@sha256:" + "a" * 64
        (self.root / "go.mod").write_text("module example.test/app\n\ngo 1.27.0\n\ntoolchain go1.27.1\n")
        (self.root / "go.sum").write_text("existing checksums\n")
        for name in ("Dockerfile", "Dockerfile.dev"):
            (self.root / name).write_text(f"# Go 1.27.1; update with make bumpgo.\nFROM {self.old_image}\n")

    def test_patch_update_preserves_minimum_and_updates_both_images(self):
        with patch.object(bumpgo.subprocess, "run") as tidy:
            bumpgo.apply_update(self.root, "1.27.0", "1.27.1", self.old_image, "1.27.4", "sha256:" + "b" * 64)
        tidy.assert_called_once_with(["go", "mod", "tidy"], cwd=self.root, check=True)
        self.assertIn("go 1.27.0\n", (self.root / "go.mod").read_text())
        self.assertIn("toolchain go1.27.4\n", (self.root / "go.mod").read_text())
        for name in ("Dockerfile", "Dockerfile.dev"):
            content = (self.root / name).read_text()
            self.assertIn("# Go 1.27.4; update with make bumpgo.", content)
            self.assertIn("golang:1.27.4-alpine@sha256:" + "b" * 64, content)

    def test_new_minor_updates_minimum_from_go_mod(self):
        with patch.object(bumpgo.subprocess, "run"):
            bumpgo.apply_update(self.root, "1.27.0", "1.27.1", self.old_image, "1.28.0", "sha256:" + "b" * 64)
        self.assertIn("go 1.28.0\n", (self.root / "go.mod").read_text())
        self.assertIn("toolchain go1.28.0\n", (self.root / "go.mod").read_text())

    def test_enter_selects_newest_option(self):
        updates = [("1.27.4", "sha256:" + "b" * 64), ("1.27.3", "sha256:" + "c" * 64)]
        with patch("builtins.input", return_value=""):
            self.assertEqual(updates[0], bumpgo.choose_update("1.27.1", updates))

    def test_tidy_failure_restores_all_files(self):
        paths = [self.root / name for name in ("go.mod", "go.sum", "Dockerfile", "Dockerfile.dev")]
        before = {path: path.read_bytes() for path in paths}

        def failed_tidy(*args, **kwargs):
            (self.root / "go.sum").write_text("changed\n")
            raise subprocess.CalledProcessError(1, args[0])

        with patch.object(bumpgo.subprocess, "run", side_effect=failed_tidy):
            with self.assertRaises(subprocess.CalledProcessError):
                bumpgo.apply_update(self.root, "1.27.0", "1.27.1", self.old_image, "1.27.4", "sha256:" + "b" * 64)
        self.assertEqual(before, {path: path.read_bytes() for path in paths})


if __name__ == "__main__":
    unittest.main()
