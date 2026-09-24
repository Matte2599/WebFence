// Check the native Cocoa accessibility bridge after a model reset.
// Exit 77 means the test host does not grant Accessibility access to this client.
import ApplicationServices
import Foundation

enum TrialError: Error, CustomStringConvertible {
    case failed(String)
    var description: String {
        switch self { case .failed(let message): return message }
    }
}

func attribute(_ element: AXUIElement, _ key: CFString) -> (AXError, CFTypeRef?) {
    var value: CFTypeRef?
    let status = AXUIElementCopyAttributeValue(element, key, &value)
    return (status, value)
}

func elements(_ element: AXUIElement, _ key: CFString) -> [AXUIElement] {
    let (status, value) = attribute(element, key)
    return status == .success ? (value as? [AXUIElement] ?? []) : []
}

func label(_ element: AXUIElement, _ key: CFString) -> String {
    let (status, value) = attribute(element, key)
    return status == .success ? (value as? String ?? "") : ""
}

func describe(_ element: AXUIElement, _ key: CFString) -> String {
    let (status, value) = attribute(element, key)
    guard status == .success else { return "AXError \(status.rawValue)" }
    return String(reflecting: value as? String)
}

func find(_ root: AXUIElement, depth: Int = 0, matching: (AXUIElement) -> Bool) -> AXUIElement? {
    if matching(root) { return root }
    if depth >= 8 { return nil }
    for child in elements(root, kAXChildrenAttribute as CFString).prefix(60) {
        if let match = find(child, depth: depth + 1, matching: matching) { return match }
    }
    return nil
}

func wait<T>(seconds: TimeInterval, for result: () -> T?) -> T? {
    let deadline = Date().addingTimeInterval(seconds)
    repeat {
        if let value = result() { return value }
        Thread.sleep(forTimeInterval: 0.2)
    } while Date() < deadline
    return nil
}

func snapshot(_ window: AXUIElement) -> String {
    guard let table = find(window, matching: {
        label($0, kAXRoleAttribute as CFString) == "AXTable"
    }) else { return "AXTable unavailable" }
    let (rowsStatus, rowsValue) = attribute(table, kAXRowsAttribute as CFString)
    guard rowsStatus == .success, let rows = rowsValue as? [AXUIElement] else {
        return "AXRows AXError \(rowsStatus.rawValue)"
    }
    guard let first = rows.first else { return "AXRows=0" }
    let (cellsStatus, cellsValue) = attribute(first, kAXChildrenAttribute as CFString)
    let cells = cellsValue as? [AXUIElement] ?? []
    let firstCell = cells.first
    return "AXRows=\(rows.count), rowRole=\(describe(first, kAXRoleAttribute as CFString)), " +
        "cells=\(cells.count) (AXError \(cellsStatus.rawValue)), " +
        "cellRole=\(firstCell.map { describe($0, kAXRoleAttribute as CFString) } ?? "none"), " +
        "cellTitle=\(firstCell.map { describe($0, kAXTitleAttribute as CFString) } ?? "none"), " +
        "cellValue=\(firstCell.map { describe($0, kAXValueAttribute as CFString) } ?? "none")"
}

func inspect(_ window: AXUIElement, expectedCount: Int, expectedID: String?) throws {
    guard let table = wait(seconds: 10, for: {
        find(window, matching: { label($0, kAXRoleAttribute as CFString) == "AXTable" })
    }) else { throw TrialError.failed("AXTable unavailable") }

    guard let rows = wait(seconds: 10, for: { () -> [AXUIElement]? in
        let (status, value) = attribute(table, kAXRowsAttribute as CFString)
        guard status == .success, let found = value as? [AXUIElement], found.count == expectedCount else { return nil }
        return found
    }) else { throw TrialError.failed("AXRows did not reach \(expectedCount)") }

    if expectedCount == 0 {
        print("PASS AXRows=0")
        return
    }
    guard let first = rows.first else { throw TrialError.failed("AXRows has no first row") }
    if let last = rows.last, rows.count > 1 {
        print("DIAG AXRows first-last equal=\(CFEqual(first, last)), " +
              "hashes=\(CFHash(first))/\(CFHash(last))")
    }

    // Give Cocoa time to settle; the historical defect persisted at rest.
    Thread.sleep(forTimeInterval: 0.5)
    let (roleStatus, roleValue) = attribute(first, kAXRoleAttribute as CFString)
    guard roleStatus == .success, (roleValue as? String) == "AXRow" else {
        throw TrialError.failed("first AXRow invalid after reset: AXError \(roleStatus.rawValue)")
    }
    let (cellStatus, cellValue) = attribute(first, kAXChildrenAttribute as CFString)
    guard cellStatus == .success, let cells = cellValue as? [AXUIElement], cells.count == 4 else {
        throw TrialError.failed("first AXRow cells unavailable: AXError \(cellStatus.rawValue)")
    }
    let id = label(cells[0], kAXTitleAttribute as CFString)
    guard id == expectedID else {
        throw TrialError.failed("first AXCell ID \(String(reflecting: id)); expected \(String(reflecting: expectedID))")
    }
    print("PASS AXRows=\(rows.count), first AXRow has 4 cells, ID=\(id)")
}

func trial(bundle: URL) throws {
    let executable = bundle.appendingPathComponent("Contents/MacOS/webfence")
    guard FileManager.default.isExecutableFile(atPath: executable.path) else {
        throw TrialError.failed("WebFence bundle executable missing")
    }
    let temp = FileManager.default.temporaryDirectory.appendingPathComponent("webfence-ax-\(UUID().uuidString)")
    try FileManager.default.createDirectory(at: temp, withIntermediateDirectories: true)
    defer { try? FileManager.default.removeItem(at: temp) }

    let process = Process()
    process.executableURL = executable
    var env = ProcessInfo.processInfo.environment
    env["HOME"] = temp.path
    env["XDG_CONFIG_HOME"] = temp.path
    env["QT_QPA_PLATFORM"] = "cocoa"
    process.environment = env
    process.standardOutput = FileHandle.nullDevice
    process.standardError = FileHandle.nullDevice
    try process.run()
    defer {
        if process.isRunning { process.terminate() }
        process.waitUntilExit()
    }

    let app = AXUIElementCreateApplication(process.processIdentifier)
    AXUIElementSetMessagingTimeout(app, 5)
    guard let window = wait(seconds: 15, for: { elements(app, kAXWindowsAttribute as CFString).first }) else {
        throw TrialError.failed("AXWindow unavailable for launched WebFence")
    }
    guard let load = find(window, matching: {
        label($0, kAXRoleAttribute as CFString) == "AXButton" &&
            (label($0, kAXTitleAttribute as CFString).contains("10.000") ||
             label($0, kAXTitleAttribute as CFString).contains("10,000"))
    }) else { throw TrialError.failed("load-fixtures AXButton unavailable") }
    guard let search = find(window, matching: { label($0, kAXRoleAttribute as CFString) == "AXTextField" }) else {
        throw TrialError.failed("search AXTextField unavailable")
    }
    let press = AXUIElementPerformAction(load, kAXPressAction as CFString)
    guard press == .success else { throw TrialError.failed("load AXPress: AXError \(press.rawValue)") }

    var failures = 0
    func observe(_ stage: String, count: Int, id: String?) {
        do {
            try inspect(window, expectedCount: count, expectedID: id)
        } catch {
            failures += 1
            print("FAIL \(stage): \(error)")
            print("DIAG \(stage): \(snapshot(window))")
        }
    }
    observe("initial/10000", count: 10_000, id: "DEMO-00001")
    for round in 1...3 {
        for (stage, query, count, id) in [
            ("one", "DEMO-10000", 1, "DEMO-10000"),
            ("empty", "no-such-fixture", 0, ""),
            ("restored", "", 10_000, "DEMO-00001")
        ] {
            let result = AXUIElementSetAttributeValue(search, kAXValueAttribute as CFString, query as CFTypeRef)
            guard result == .success else {
                throw TrialError.failed("round \(round)/\(stage) filter AXValue: AXError \(result.rawValue)")
            }
            observe("round \(round)/\(stage)", count: count, id: count == 0 ? nil : id)
        }
    }
    guard failures == 0 else { throw TrialError.failed("\(failures) AX stage failures across 3 rounds") }
}

guard CommandLine.arguments.count == 2 else {
    fputs("usage: test-macos-ax-reset WebFence.app\n", stderr)
    exit(64)
}
guard AXIsProcessTrusted() else {
    fputs("UNAVAILABLE: Accessibility permission is absent for the AX test client\n", stderr)
    exit(77)
}
do {
    try trial(bundle: URL(fileURLWithPath: CommandLine.arguments[1], isDirectory: true))
    print("PASS native Cocoa AX reset 10000 → 1 → 0 → 10000, 3 rounds")
} catch {
    fputs("FAIL native Cocoa AX reset: \(error)\n", stderr)
    exit(1)
}
