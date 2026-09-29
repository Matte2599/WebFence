param(
    [switch]$ControlOnly,
    [ValidateSet('All', 'Chrome', 'Edge')][string]$Browser = 'All'
)
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '../..')
$candidates = @(
    "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe",
    "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe",
    "$env:ProgramFiles\Google\Chrome\Application\chrome.exe",
    "${env:ProgramFiles(x86)}\Google\Chrome\Application\chrome.exe"
)
$browsers = @($candidates | Where-Object {
    (Test-Path $_) -and ($Browser -eq 'All' -or
        ($Browser -eq 'Chrome' -and $_ -like '*\Google\Chrome\*') -or
        ($Browser -eq 'Edge' -and $_ -like '*\Microsoft\Edge\*'))
})
if (!$browsers.Count) { throw 'Browser executable unavailable' }
$mode = '--browser'
if ($ControlOnly) { $mode = '--browser-control' }
$failed = $false
foreach ($browserExecutable in $browsers) {
    Write-Output "Browser trial ($mode): $browserExecutable"
    ./m3-appcontainer.exe $mode $browserExecutable
    if ($LASTEXITCODE -ne 0) { $failed = $true }
}
if ($failed) { throw "Browser trial failed ($mode)" }
