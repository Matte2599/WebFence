param([switch]$ControlOnly)
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '../..')
$candidates = @(
    "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe",
    "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe",
    "$env:ProgramFiles\Google\Chrome\Application\chrome.exe",
    "${env:ProgramFiles(x86)}\Google\Chrome\Application\chrome.exe"
)
$browsers = @($candidates | Where-Object { Test-Path $_ })
if (!$browsers.Count) { throw 'Browser executable unavailable' }
$mode = '--browser'
if ($ControlOnly) { $mode = '--browser-control' }
$failed = $false
foreach ($browser in $browsers) {
    Write-Output "Browser trial ($mode): $browser"
    ./m3-appcontainer.exe $mode $browser
    if ($LASTEXITCODE -ne 0) { $failed = $true }
}
if ($failed) { throw "Browser trial failed ($mode)" }
