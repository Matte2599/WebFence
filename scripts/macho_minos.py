"""Read macOS deployment requirements and declare them on private bundle staging.

A load-command floor is necessary metadata, not proof of runtime compatibility
or an author-approved support policy. Never rewrite binary load commands.
"""
import plistlib
import re


def version(value):
    if not isinstance(value, str) or not re.fullmatch(r'[0-9]{1,5}(?:\.[0-9]{1,3}){0,2}', value):
        raise ValueError('Invalid macOS version')
    parts = tuple(int(x) for x in value.split('.'))
    parts += (0,) * (3 - len(parts))
    if not 10 <= parts[0] <= 65535 or any(x > 255 for x in parts[1:]):
        raise ValueError('Invalid macOS version range')
    return parts


def display(parts):
    return '.'.join(str(x) for x in parts)


def parse_otool(text):
    records = []
    for block in re.split(r'(?m)^Load command [0-9]+\s*$', text):
        commands = re.findall(r'^\s*cmd (LC_\S+)\s*$', block, flags=re.M)
        if not commands:
            continue
        if len(commands) != 1:
            raise ValueError('Ambiguous Mach-O load command')
        command = commands[0]
        if command == 'LC_BUILD_VERSION':
            platforms = re.findall(r'^\s*platform (\S+)\s*$', block, flags=re.M)
            if len(platforms) != 1 or platforms[0].upper() not in {'1', 'MACOS'}:
                raise ValueError('Expected macOS Mach-O platform')
            minima = re.findall(r'^\s*minos (\S+)\s*$', block, flags=re.M)
        elif command == 'LC_VERSION_MIN_MACOSX':
            minima = re.findall(r'^\s*version (\S+)\s*$', block, flags=re.M)
        elif command.startswith('LC_VERSION_MIN_'):
            raise ValueError('Expected macOS legacy deployment command')
        else:
            continue
        if len(minima) != 1:
            raise ValueError('Missing or ambiguous Mach-O minimum version')
        records.append({'command': command, 'platform': 'macos', 'minimum': display(version(minima[0]))})
    if len(records) != 1:
        raise ValueError('Expected exactly one deployment command in the arm64 slice')
    return records[0]


def declare_minimum(plist_path, files):
    if not files:
        raise ValueError('No Mach-O deployment requirements')
    requirements = [(version(item['minimum_macos']['minimum']), item['path']) for item in files]
    required = max(value for value, _ in requirements)
    data = plistlib.loads(plist_path.read_bytes())
    if not isinstance(data, dict):
        raise ValueError('Expected bundle property dictionary')
    declared = required
    if 'LSMinimumSystemVersion' in data:
        declared = max(required, version(data['LSMinimumSystemVersion']))
    # Per-architecture declarations add a separate architecture selection policy.
    # This single-arm64 builder requires explicit review before combining them.
    if 'LSMinimumSystemVersionByArchitecture' in data:
        raise ValueError('Per-architecture OS policy requires explicit review')
    data['LSMinimumSystemVersion'] = display(declared)
    plist_path.write_bytes(plistlib.dumps(data, fmt=plistlib.FMT_XML, sort_keys=False))
    return {'architecture': 'arm64', 'mach_o_requirement': display(required),
            'bundle_declaration': display(declared),
            'limiting_files': sorted(path for value, path in requirements if value == required),
            'basis': 'Maximum deployment target recorded in included Mach-O files, preserving a higher existing bundle declaration; not runtime compatibility certification or product support policy.'}
