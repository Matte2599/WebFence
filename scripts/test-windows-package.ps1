param([Parameter(Mandatory = $true)][string]$Archive)
$ErrorActionPreference = 'Stop'
$probeRoot = Join-Path ([IO.Path]::GetTempPath()) ('WebFence package à ' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $probeRoot | Out-Null
try {
    Expand-Archive -LiteralPath $Archive -DestinationPath $probeRoot
    $bundle = Join-Path $probeRoot 'WebFence'
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
