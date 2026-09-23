"""Exercise the installed WebFence process through Linux AT-SPI, not Qt internals."""

import sys
import time

import pyatspi


LABELS = {
    "en": ("Load 10,000 examples", "Filter by ID or URL", "Synthetic results"),
    "it": ("Carica 10.000 esempi", "Filtra per ID o URL", "Risultati sintetici"),
}
LANGUAGE = sys.argv[1]
LOAD, SEARCH, RESULTS = LABELS[LANGUAGE]


def find(root, role, name, depth=0):
    if depth > 7:
        return None
    try:
        if root.getRoleName() == role and root.name == name:
            return root
        for child in root:
            match = find(child, role, name, depth + 1)
            if match is not None:
                return match
    except (LookupError, RuntimeError):
        # Qt may replace nodes during a model reset; retry from the root.
        return None
    return None


def application():
    for item in pyatspi.Registry.getDesktop(0):
        if item.name == "webfence" and find(item, "push button", LOAD):
            return item
    return None


def wait_for(description, probe, seconds=20):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        value = probe()
        if value is not None:
            return value
        time.sleep(0.1)
    raise AssertionError(f"{LANGUAGE}: AT-SPI did not expose {description}")


app = wait_for("the localized application", application)
button = wait_for("the load button", lambda: find(app, "push button", LOAD))
search = wait_for("the filter", lambda: find(app, "text", SEARCH))


def table_with_rows(expected, first_cell):
    table = find(app, "table", RESULTS)
    if table is None:
        return None
    interface = table.queryTable()
    if interface.nRows != expected:
        return None
    if interface.nColumns != 4:
        raise AssertionError(f"{LANGUAGE}: expected 4 columns, got {interface.nColumns}")
    if first_cell is not None:
        try:
            cell = interface.getAccessibleAt(0, 0)
            if cell.getRoleName() != "table cell" or cell.name != first_cell:
                return None
        except Exception:
            # The bridge may expose the new count before its cell references.
            return None
    return interface


def assert_table(expected, first_cell=None):
    wait_for(f"{expected} rows and first cell {first_cell!r}",
             lambda: table_with_rows(expected, first_cell))
    print(f"{LANGUAGE}: {expected} rows, first cell {first_cell!r}", flush=True)


assert_table(0)
if not button.queryAction().doAction(0):
    raise AssertionError(f"{LANGUAGE}: load action failed")
assert_table(10000, "DEMO-00001")
search.queryEditableText().setTextContents("DEMO-10000")
assert_table(1, "DEMO-10000")
search.queryEditableText().setTextContents("no-such-fixture")
assert_table(0)
search.queryEditableText().setTextContents("")
assert_table(10000, "DEMO-00001")
