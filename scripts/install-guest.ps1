param(
    [string]$HostUrl = "",
    [string]$Token = "",
    [string]$ConfigPath = "",
    [int]$TimeoutSeconds = 5,
    [switch]$NoOpenSettings
)

$ErrorActionPreference = "Stop"

function Resolve-Binary {
    param(
        [string[]]$Candidates
    )

    foreach ($candidate in $Candidates) {
        if (Test-Path $candidate) {
            return (Resolve-Path $candidate).Path
        }
    }

    throw "Could not find a URL Bridge guest binary. Checked: $($Candidates -join ', ')"
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$rootDir = Split-Path -Parent $scriptDir
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

if (-not $ConfigPath) {
    $ConfigPath = Join-Path (Join-Path $env:LOCALAPPDATA "URLBridge") "config.yaml"
}

if ($arch -eq "arm64") {
    $guestCtl = Resolve-Binary @(
        (Join-Path $scriptDir "urlbridge-guestctl-arm64.exe"),
        (Join-Path $scriptDir "urlbridge-guestctl.exe"),
        (Join-Path $rootDir "dist\urlbridge-guestctl-arm64.exe"),
        (Join-Path $rootDir "dist\urlbridge-guestctl.exe")
    )
}
else {
    $guestCtl = Resolve-Binary @(
        (Join-Path $scriptDir "urlbridge-guestctl.exe"),
        (Join-Path $rootDir "dist\urlbridge-guestctl.exe")
    )
}

$arguments = @("install")
$arguments += @("--config", $ConfigPath)
if ($HostUrl) {
    $arguments += @("--host-url", $HostUrl)
}
if ($Token) {
    $arguments += @("--token", $Token)
}
if ($NoOpenSettings) {
    $arguments += "--no-open-settings"
}
$arguments += @("--timeout", $TimeoutSeconds.ToString())

& $guestCtl @arguments
