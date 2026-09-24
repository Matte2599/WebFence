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
    $signatureLockPath = Join-Path $bundle 'notices/native/msys2-binary-signature-lock.json'
    if (-not (Test-Path -LiteralPath $signatureLockPath -PathType Leaf) -or
        (Get-FileHash -Algorithm SHA256 -LiteralPath $signatureLockPath).Hash -ne $native.binary_signature_lock_sha256) {
        throw 'Missing or changed reviewed MSYS2 binary signature lock'
    }
    $binarySignatureLock = Get-Content -Raw -LiteralPath $signatureLockPath | ConvertFrom-Json
    $signer = $binarySignatureLock.key
    $signingKeyPath = Join-Path $bundle 'notices/native/msys2-binary-signing-key.asc'
    if ($binarySignatureLock.schema -ne 1 -or
        $signer.fingerprint -ne '5F944B027F7FE2091985AA2EFA11531AA0AA7F57' -or
        -not (Test-Path -LiteralPath $signingKeyPath -PathType Leaf) -or
        (Get-FileHash -Algorithm SHA256 -LiteralPath $signingKeyPath).Hash -ne $signer.public_key_sha256) {
        throw 'Missing or changed MSYS2 binary signing key'
    }
    $signaturePins = @{}
    foreach ($pin in @($binarySignatureLock.packages)) {
        if ($signaturePins.ContainsKey($pin.name) -or -not $lockedPackages.ContainsKey($pin.name) -or
            $pin.version -ne $lockedPackages[$pin.name].version -or
            $pin.archive_sha256 -ne $lockedPackages[$pin.name].sha256) {
            throw "Unreviewed binary package signature: $($pin.name)"
        }
        $signaturePins[$pin.name] = $pin
    }
    if ($signaturePins.Count -ne $packageNames.Count) {
        throw 'Binary signature set differs from packaged owners'
    }
    $recordedDlls = @{}
    $rootDlls = @{}
    foreach ($file in $native.files) {
        if ($packageNames -notcontains $file.package -or $file.path.Contains('\') -or
            $file.path.Split('/') -contains '..' -or $recordedDlls.ContainsKey($file.path) -or
            $null -eq $file.PSObject.Properties['imports']) {
            throw "Invalid DLL package mapping: $($file.path)"
        }
        $recordedDlls[$file.path] = $true
        if (-not $file.path.Contains('/')) { $rootDlls[$file.path.ToLowerInvariant()] = $true }
        $binary = Join-Path $bundle $file.path
        if (-not (Test-Path -LiteralPath $binary -PathType Leaf) -or
            (Get-FileHash -Algorithm SHA256 -LiteralPath $binary).Hash -ne $file.sha256) {
            throw "Packaged DLL checksum mismatch: $($file.path)"
        }
    }
    $actualDlls = @(Get-ChildItem -LiteralPath $bundle -Recurse -File -Filter '*.dll')
    if ($actualDlls.Count -ne $recordedDlls.Count) {
        throw 'Extracted DLL count differs from native inventory'
    }
    foreach ($binary in $actualDlls) {
        $relative = [IO.Path]::GetRelativePath($bundle, $binary.FullName).Replace('\', '/')
        if (-not $recordedDlls.ContainsKey($relative)) {
            throw "Uninventoried DLL in extracted ZIP: $relative"
        }
    }
    if ($null -eq $native.PSObject.Properties['executable_imports']) {
        throw 'Missing executable import inventory'
    }
    $systemImports = @{}
    foreach ($name in @($native.system_imports)) {
        if ($name -cnotmatch '^[a-z0-9_.+-]+\.dll$' -or $systemImports.ContainsKey($name)) {
            throw "Invalid system import: $name"
        }
        $systemImports[$name] = $true
    }
    $importRecords = @(@{ path = 'webfence.exe'; names = $native.executable_imports })
    foreach ($file in $native.files) {
        $importRecords += @{ path = $file.path; names = $file.imports }
    }
    foreach ($record in $importRecords) {
        $seenImports = @{}
        foreach ($name in @($record.names)) {
            if ($name -cnotmatch '^[a-z0-9_.+-]+\.dll$' -or $seenImports.ContainsKey($name)) {
                throw "Invalid or duplicate PE import in $($record.path): $name"
            }
            $seenImports[$name] = $true
            if (-not $systemImports.ContainsKey($name) -and -not $rootDlls.ContainsKey($name)) {
                throw "Unresolved PE import in $($record.path): $name"
            }
        }
    }
    $qtFiles = @($native.files | Where-Object { $_.package -eq 'mingw-w64-ucrt-x86_64-qt6-base' } |
        ForEach-Object { $_.path } | Sort-Object)
    if ($qtFiles.Count -lt 1) { throw 'No Qt DLLs in the extracted ZIP inventory' }
    Write-Output "PASS extracted ZIP DLL and static-import inventory: $($actualDlls.Count) DLLs, $($importRecords.Count) PE files"
    Write-Output "Qt files in extracted ZIP: $($qtFiles -join ', ')"
    $noticeCount = 0
    foreach ($owner in $packageNames) {
        $record = $native.packages.PSObject.Properties[$owner].Value
        if (-not $lockedPackages.ContainsKey($owner) -or
            $record.version -ne $lockedPackages[$owner].version -or
            $record.binary_package.sha256 -ne $lockedPackages[$owner].sha256 -or
            $record.binary_package.published_sha256_source -ne $lockedPackages[$owner].source_page) {
            throw "Unreviewed MSYS2 binary archive: $owner"
        }
        $signaturePin = $signaturePins[$owner]
        $signatureEvidence = $record.binary_package.signature_verification
        $signatureRelative = "notices/native/$owner/build/package.sig"
        $signaturePath = Join-Path $bundle $signatureRelative
        if ($record.binary_package.signature_file -ne $signatureRelative -or
            $signatureEvidence.method -ne 'offline_openpgp_detached_signature' -or
            $signatureEvidence.signer_fingerprint -ne $signer.fingerprint -or
            $signatureEvidence.public_key_sha256 -ne $signer.public_key_sha256 -or
            $signatureEvidence.signature_sha256 -ne $signaturePin.signature_sha256 -or
            $signatureEvidence.archive_sha256 -ne $signaturePin.archive_sha256 -or
            -not (Test-Path -LiteralPath $signaturePath -PathType Leaf) -or
            (Get-FileHash -Algorithm SHA256 -LiteralPath $signaturePath).Hash -ne $signaturePin.signature_sha256) {
            throw "Missing or changed offline binary signature evidence: $owner"
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
    Write-Output "PASS extracted ZIP native notices, checksums and package signature evidence: $($native.files.Count) DLLs, $($packageNames.Count) owners, $noticeCount source-matched notices, 27 Qt license references"
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
        $vcsLockPath = Join-Path $sourceRoot 'winpthreads-vcs-lock.json'
        if (-not (Test-Path -LiteralPath $vcsLockPath -PathType Leaf) -or
            (Get-FileHash -Algorithm SHA256 -LiteralPath $vcsLockPath).Hash -ne $attachment.winpthreads_vcs_lock_sha256) {
            throw 'Missing or changed winpthreads VCS source lock'
        }
        $vcsLock = Get-Content -Raw -LiteralPath $vcsLockPath | ConvertFrom-Json
        $winpthreads = @($manifest.archives | Where-Object { $_.base -eq $vcsLock.source_package })
        if ($winpthreads.Count -ne 1 -or $winpthreads[0].archive_sha256 -ne $vcsLock.archive_sha256) {
            throw 'winpthreads archive differs from reviewed VCS lock'
        }
        $vcsChecks = @($winpthreads[0].source_checks | Where-Object { $_.source -eq $vcsLock.vcs_source })
        if ($vcsChecks.Count -ne 1 -or $vcsChecks[0].status -ne 'verified' -or
            $vcsChecks[0].verification.method -ne 'isolated_git_archive_sha256' -or
            $vcsChecks[0].verification.commit -ne $vcsLock.commit -or
            $vcsChecks[0].verification.sha256 -ne $vcsLock.git_archive_sha256 -or
            $vcsChecks[0].verification.size_bytes -ne $vcsLock.git_archive_size_bytes) {
            throw 'winpthreads offline Git checksum evidence is missing or changed'
        }
        $signatureLockPath = Join-Path $sourceRoot 'source-signature-lock.json'
        if (-not (Test-Path -LiteralPath $signatureLockPath -PathType Leaf) -or
            (Get-FileHash -Algorithm SHA256 -LiteralPath $signatureLockPath).Hash -ne $attachment.source_signature_lock_sha256) {
            throw 'Missing or changed Windows source signature lock'
        }
        $signatureLock = Get-Content -Raw -LiteralPath $signatureLockPath | ConvertFrom-Json
        if (@($signatureLock.entries).Count -ne 8) { throw 'Expected eight reviewed source signatures' }
        foreach ($entry in $signatureLock.entries) {
            $keyFile = Join-Path $sourceRoot $entry.public_key_file
            if (-not (Test-Path -LiteralPath $keyFile -PathType Leaf) -or
                (Get-FileHash -Algorithm SHA256 -LiteralPath $keyFile).Hash -ne $entry.public_key_sha256) {
                throw "Missing or changed source signing key: $($entry.source_package)"
            }
            $sources = @($manifest.archives | Where-Object { $_.base -eq $entry.source_package })
            if ($sources.Count -ne 1 -or $sources[0].version -ne $entry.version -or
                $sources[0].archive_sha256 -ne $entry.archive_sha256) {
                throw "Signed source archive differs from lock: $($entry.source_package)"
            }
            $signatures = @($sources[0].source_checks | Where-Object { $_.source -eq $entry.signature_source })
            if ($signatures.Count -ne 1 -or $signatures[0].status -ne 'verified' -or
                $signatures[0].signature_verified -ne $true -or
                $signatures[0].verification.method -ne 'offline_openpgp_detached_signature' -or
                $signatures[0].verification.primary_fingerprint -ne $entry.primary_fingerprint -or
                $signatures[0].verification.signer_fingerprint -ne $entry.signer_fingerprint -or
                $signatures[0].verification.signature_sha256 -ne $entry.signature_sha256 -or
                $signatures[0].verification.payload_sha256 -ne $entry.payload_sha256) {
                throw "Offline source signature evidence is missing or changed: $($entry.source_package)"
            }
        }
        Write-Output "PASS extracted ZIP source attachment: $($attachment.archive_count) archives, $archiveBytes bytes, inventory and recipe hashes; winpthreads Git archive and 8 detached signatures verified"
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
            $loadedBundleModules = @{}
            $externalModules = @{}
            $moduleSamples = 0
            if ($trial -eq '--soak-test=10s') {
                $bundlePrefix = [IO.Path]::GetFullPath($bundle).TrimEnd('\') + '\'
                $windowsPrefix = [IO.Path]::GetFullPath($env:SystemRoot).TrimEnd('\') + '\'
                $traceUntil = [DateTime]::UtcNow.AddSeconds(20)
                while (-not $process.HasExited -and [DateTime]::UtcNow -lt $traceUntil) {
                    try {
                        $process.Refresh()
                        $modules = @($process.Modules)
                    } catch [System.ComponentModel.Win32Exception] {
                        if ($process.HasExited) { break }
                        Start-Sleep -Milliseconds 100
                        continue
                    } catch {
                        if ($process.HasExited) { break }
                        throw
                    }
                    foreach ($module in $modules) {
                        try {
                            $modulePath = [IO.Path]::GetFullPath($module.FileName)
                        } catch [System.ComponentModel.Win32Exception] {
                            continue  # The module may have unloaded between enumeration and inspection.
                        }
                        if ($modulePath.StartsWith($bundlePrefix, [StringComparison]::OrdinalIgnoreCase)) {
                            $relative = [IO.Path]::GetRelativePath($bundle, $modulePath).Replace('\', '/')
                            if ($relative -ne 'webfence.exe' -and -not $recordedDlls.ContainsKey($relative)) {
                                throw "Uninventoried module loaded from WebFence ZIP: $relative"
                            }
                            $loadedBundleModules[$relative] = $true
                        } elseif (-not $modulePath.StartsWith($windowsPrefix, [StringComparison]::OrdinalIgnoreCase)) {
                            $externalModules[$modulePath] = $true
                        }
                    }
                    $moduleSamples++
                    Start-Sleep -Milliseconds 100
                }
            }
            if (-not $process.WaitForExit(120000)) {
                $process.Kill($true)
                throw "Packaged $platform test timed out"
            }
            Write-Output ($stdout.GetAwaiter().GetResult())
            Write-Output ($stderr.GetAwaiter().GetResult())
            if ($process.ExitCode -ne 0) { throw "Packaged $platform test failed: $($process.ExitCode)" }
            $process.Dispose()
            Write-Output "PASS packaged $platform $trial with no MSYS2/Go in PATH and a Unicode/spaced directory"
            if ($trial -eq '--soak-test=10s') {
                Write-Output "Observed packaged $platform modules ($moduleSamples samples): $(($loadedBundleModules.Keys | Sort-Object) -join ', ')"
                Write-Output "External module paths for review: $(($externalModules.Keys | Sort-Object) -join ', ')"
                $expectedPlugin = if ($platform -eq 'windows') { 'platforms/qwindows.dll' } else { 'platforms/qoffscreen.dll' }
                foreach ($required in @('webfence.exe', 'Qt6Core.dll', 'Qt6Gui.dll', 'Qt6Widgets.dll', $expectedPlugin)) {
                    if (-not $loadedBundleModules.ContainsKey($required)) {
                        throw "Packaged $platform soak did not expose expected loaded module: $required"
                    }
                }
                if ($moduleSamples -lt 1) { throw "Packaged $platform soak produced no process module sample" }
                Write-Output "PASS packaged $platform module samples: $moduleSamples; expected Qt platform plugin observed"
            }
        }
    }
} finally {
    Remove-Item -LiteralPath $probeRoot -Recurse -Force
}
