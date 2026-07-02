#!/usr/bin/env python3
import hashlib
import pathlib
import sys

root = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else 'dist')
for path in sorted(p for p in root.rglob('*') if p.is_file()):
    h = hashlib.sha256()
    with path.open('rb') as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b''):
            h.update(chunk)
    print(f'{h.hexdigest()}  {path}')
