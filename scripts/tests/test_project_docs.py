"""Check documentation links in the actual subset copied into a package."""
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


class ProjectDocsPackageTest(unittest.TestCase):
    def test_packaged_documentation_has_no_missing_local_links(self):
        repository = Path(__file__).resolve().parents[2]
        with tempfile.TemporaryDirectory(prefix='webfence-doc-package-') as temporary:
            target = Path(temporary) / 'notices'
            result = subprocess.run(
                [sys.executable, str(repository / 'scripts/package-project-docs.py'), str(target)],
                cwd=repository, capture_output=True, text=True, timeout=30)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertTrue((target / 'DOCS/evidence/windows-vcs-2026-09-24.md').is_file())


if __name__ == '__main__':
    unittest.main()
