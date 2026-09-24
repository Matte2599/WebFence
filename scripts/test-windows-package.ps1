param([Parameter(Mandatory = $true)][string]$Archive, [switch]$RequireSources)
$ErrorActionPreference = 'Stop'
$probeRoot = Join-Path ([IO.Path]::GetTempPath()) ('WebFence package à ' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $probeRoot | Out-Null
try {
    Expand-Archive -LiteralPath $Archive -DestinationPath $probeRoot
    $bundle = Join-Path $probeRoot 'WebFence'
    foreach ($document in @('LICENSE', 'README.md', 'README.en.md', 'DOCS/README.md',
                            'DOCS/it/DEVELOPMENT.md', 'DOCS/en/DEVELOPMENT.md')) {
        if (-not (Test-Path -LiteralPath (Join-Path $bundle $document) -PathType Leaf)) {
            throw "Missing packaged documentation: $document"
        }
    }
    Write-Output 'PASS extracted ZIP includes license and Italian/English documentation entry points'
    $native = Get-Content -Raw -LiteralPath (Join-Path $bundle 'native-build.json') | ConvertFrom-Json
    $packageNames = @($native.packages.PSObject.Properties.Name)
    if ($native.schema -ne 1 -or @($native.files).Count -lt 1 -or $packageNames.Count -lt 1) {
        throw 'Invalid packaged native inventory'
    }
    $lockPath = Join-Path $bundle 'notices/native/msys2-binary-lock.json'
    if (-not (Test-Path -LiteralPath $lockPath -PathType Leaf) -or
        (Get-FileHash -Algorithm SHA256 -LiteralPath $lockPath).Hash -ne $native.binary_lock_sha256) {
        throw 'Missing or changed reviewed MSYS2 binary checksum lock'
    }
    $binaryLock = Get-Content -Raw -LiteralPath $lockPath | ConvertFrom-Json
    $lockedPackages = @{}
    foreach ($pin in @($binaryLock.packages)) {
        if ($lockedPackages.ContainsKey($pin.name) -or
            $pin.source_page -ne ('https://packages.msys2.org/packages/' + $pin.name)) {
            throw "Duplicate or invalid reviewed binary entry: $($pin.name)"
        }
        $lockedPackages[$pin.name] = $pin
    }
    if ($binaryLock.schema_version -ne 1 -or $lockedPackages.Count -ne $packageNames.Count) {
        throw 'Reviewed binary package set differs from native inventory'
    }
    foreach ($file in $native.files) {
        if ($packageNames -notcontains $file.package -or $file.path.Contains('\') -or
            $file.path.Split('/') -contains '..') {
            throw "Invalid DLL package mapping: $($file.path)"
        }
        $binary = Join-Path $bundle $file.path
        if (-not (Test-Path -LiteralPath $binary -PathType Leaf) -or
            (Get-FileHash -Algorithm SHA256 -LiteralPath $binary).Hash -ne $file.sha256) {
            throw "Packaged DLL checksum mismatch: $($file.path)"
        }
    }
    $noticeCount = 0
    foreach ($owner in $packageNames) {
        $record = $native.packages.PSObject.Properties[$owner].Value
        if (-not $lockedPackages.ContainsKey($owner) -or
            $record.version -ne $lockedPackages[$owner].version -or
            $record.binary_package.sha256 -ne $lockedPackages[$owner].sha256 -or
            $record.binary_package.published_sha256_source -ne $lockedPackages[$owner].source_page) {
            throw "Unreviewed MSYS2 binary archive: $owner"
        }
        $notices = @($record.license_file_records)
        $paths = @($record.license_files)
        if ($notices.Count -lt 1 -or $notices.Count -ne $paths.Count -or
            $record.binary_package.verified_notice_count -ne $notices.Count) {
            throw "Incomplete packaged notice inventory: $owner"
        }
        $seen = @{}
        foreach ($notice in $notices) {
            $prefix = "notices/native/$owner/"
            if (-not $notice.path.StartsWith($prefix, [StringComparison]::Ordinal) -or
                $notice.path.Contains('\') -or $notice.path.Split('/') -contains '..' -or
                $seen.ContainsKey($notice.path) -or $paths -notcontains $notice.path -or
                -not $notice.binary_package_member.StartsWith('ucrt64/share/', [StringComparison]::Ordinal)) {
                throw "Invalid packaged notice mapping: $owner"
            }
            $seen[$notice.path] = $true
            $path = Join-Path $bundle $notice.path
            if (-not (Test-Path -LiteralPath $path -PathType Leaf) -or
                (Get-FileHash -Algorithm SHA256 -LiteralPath $path).Hash -ne $notice.sha256) {
                throw "Packaged notice checksum mismatch: $($notice.path)"
            }
            $noticeCount++
        }
    }
    $qtOwner = $native.packages.PSObject.Properties['mingw-w64-ucrt-x86_64-qt6-base'].Value
    if ($null -eq $qtOwner -or $qtOwner.qt_license_reference_count -ne 27) {
        throw 'Qt attribution license references were not verified during packaging'
    }
    Write-Output "PASS extracted ZIP native notices and published checksum lock: $($native.files.Count) DLLs, $($packageNames.Count) owners, $noticeCount source-matched notices, 27 Qt license references"
    $sourceRoot = Join-Path $bundle 'msys2-sources'
    if ($RequireSources -or (Test-Path -LiteralPath $sourceRoot)) {
        $attachment = Get-Content -Raw -LiteralPath (Join-Path $sourceRoot 'attachment.json') | ConvertFrom-Json
        $manifest = Get-Content -Raw -LiteralPath (Join-Path $sourceRoot 'source-materials.json') | ConvertFrom-Json
        if ($attachment.distribution_ready -ne $false -or $attachment.corresponding_sources_complete -ne $false) {
            throw 'Source materials must retain incomplete distribution status'
        }
        $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $bundle 'native-build.json')).Hash
        if ($hash -ne $attachment.current_native_build_sha256 -or $hash -ne
            (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $sourceRoot 'native-build.current.json')).Hash) {
            throw 'Source attachment is not bound to the packaged native inventory'
        }
        foreach ($entry in @(@('source-materials.json', 'collection_manifest_sha256'),
                             @('native-build.input.json', 'collection_input_sha256'))) {
            if ((Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $sourceRoot $entry[0])).Hash -ne $attachment.($entry[1])) {
                throw "Source attachment metadata checksum mismatch: $($entry[0])"
            }
        }
        $archiveBytes = [long]0
        foreach ($record in $manifest.archives) {
            $archiveFile = Join-Path $sourceRoot $record.archive
            if ((Get-FileHash -Algorithm SHA256 -LiteralPath $archiveFile).Hash -ne $record.archive_sha256 -or
                (Get-Item -LiteralPath $archiveFile).Length -ne $record.size_bytes) {
                throw "Packaged source archive checksum/size mismatch: $($record.archive)"
            }
            $archiveBytes += $record.size_bytes
            foreach ($name in @('PKGBUILD', '.SRCINFO')) {
                $recipe = Join-Path (Join-Path $sourceRoot $record.recipe_directory) $name
                if ((Get-FileHash -Algorithm SHA256 -LiteralPath $recipe).Hash -ne $record.files.($name).sha256) {
                    throw "Packaged source recipe checksum mismatch: $recipe"
                }
            }
        }
        if ($manifest.archives.Count -lt 1 -or $manifest.archives.Count -ne $attachment.archive_count -or
            $archiveBytes -ne $attachment.archive_bytes) { throw 'Source attachment totals differ' }
        Write-Output "PASS extracted ZIP source attachment: $($attachment.archive_count) archives, $archiveBytes bytes, inventory and recipe hashes"
    }
    foreach ($platform in @('offscreen', 'windows')) {
        foreach ($trial in @('--self-test', '--soak-test=10s')) {
            $info = [Diagnostics.ProcessStartInfo]::new()
            $info.FileName = Join-Path $bundle 'webfence.exe'
            $info.ArgumentList.Add($trial)
            $info.WorkingDirectory = $probeRoot
            $info.UseShellExecute = $false
            $info.RedirectStandardOutput = $true
            $info.RedirectStandardError = $true
            foreach ($key in @($info.Environment.Keys)) {
                if ($key -like 'QT*' -or $key -eq 'QML2_IMPORT_PATH') { $info.Environment.Remove($key) | Out-Null }
            }
            $info.Environment['PATH'] = "$env:SystemRoot\System32;$env:SystemRoot"
            $info.Environment['QT_QPA_PLATFORM'] = $platform
            $process = [Diagnostics.Process]::Start($info)
            $stdout = $process.StandardOutput.ReadToEndAsync()
            $stderr = $process.StandardError.ReadToEndAsync()
            if (-not $process.WaitForExit(120000)) {
                $process.Kill($true)
                throw "Packaged $platform test timed out"
            }
            Write-Output ($stdout.GetAwaiter().GetResult())
            Write-Output ($stderr.GetAwaiter().GetResult())
            if ($process.ExitCode -ne 0) { throw "Packaged $platform test failed: $($process.ExitCode)" }
            $process.Dispose()
            Write-Output "PASS packaged $platform $trial with no MSYS2/Go in PATH and a Unicode/spaced directory"
        }
    }
} finally {
    Remove-Item -LiteralPath $probeRoot -Recurse -Force
}
