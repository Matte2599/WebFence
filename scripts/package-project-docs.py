#!/usr/bin/env python3
"""Copy project documentation and check local inline Markdown link targets.

Used only on trusted repository/build inputs. Checks files, not heading anchors
or remote URLs. It neither executes referenced scripts nor contacts websites.
"""
import argparse
from pathlib import Path
import re
import shutil
from urllib.parse import unquote, urlsplit

FILES = ('LICENSE', 'README.md', 'README.en.md', 'AGENTS.md', 'MEMORY.md',
         'CONTRIBUTING.md', 'SECURITY.md', 'experiments/qt/README.md',
         'scripts/build-qt-cocoa.sh')
TREES = ('DOCS', 'scripts/qt-cocoa')


def check(root):
    root = Path(root).resolve()
    for relative in (*FILES, 'DOCS/README.md', 'scripts/qt-cocoa/README.md'):
        if not (root / relative).is_file():
            raise ValueError('Missing packaged document: ' + relative)
    documents = [root / name for name in FILES if name.endswith('.md')]
    for tree in TREES:
        documents.extend((root / tree).rglob('*.md'))
    links = 0
    for document in documents:
        text = document.read_text(encoding='utf-8')
        # The project currently uses inline Markdown links. Do not interpret
        # ordinary code paths as hyperlinks or claim full Markdown validation.
        for target in re.findall(r'\]\(([^\s)]+)(?:\s+[^)]*)?\)', text):
            if urlsplit(target).scheme or target.startswith('#'):
                continue
            path = unquote(target.split('#')[0])
            if not path:
                continue
            resolved = (document.parent / path).resolve()
            if not resolved.is_relative_to(root) or not resolved.exists():
                raise ValueError('Missing or external local link: ' +
                                 str(document.relative_to(root)) + ' -> ' + target)
            links += 1
    print(f'PASS packaged project documentation: {len(documents)} Markdown files, {links} local file links')


def copy(root):
    source = Path(__file__).resolve().parent.parent
    root = Path(root)
    # Fail before copying if this would replace any existing documentation.
    for relative in (*FILES, *TREES):
        target = root / relative
        if target.exists() or target.is_symlink():
            raise FileExistsError('Refusing to replace packaged documentation: ' + relative)
    for relative in FILES:
        target = root / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        with (source / relative).open('rb') as original, target.open('xb') as output:
            shutil.copyfileobj(original, output)
    for relative in TREES:
        shutil.copytree(source / relative, root / relative, ignore=shutil.ignore_patterns('.DS_Store'))
    check(root)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('documentation_root')
    parser.add_argument('--check', action='store_true', help='Validate an existing packaged documentation tree')
    args = parser.parse_args()
    (check if args.check else copy)(args.documentation_root)
