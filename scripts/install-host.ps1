param(
    [string]$Listen = "0.0.0.0:38495",
    [string]$Token = "",
    [string]$ConfigPath = "",
    [switch]$NoAutoStart
)

$ErrorActionPreference = "Stop"

function Resolve-SourceBinary {
    param(
        [string[]]$Candidates
    )

    foreach ($candidate in $Candidates) {
        if (Test-Path $candidate) {
            return (Resolve-Path $candidate).Path
        }
    }

    throw "Could not find a URL Bridge host binary. Checked: $($Candidates -join ', ')"
}

function Get-YamlScalar {
    param(
        [string]$Path,
        [string]$Key
    )

    if (-not (Test-Path $Path)) {
        return $null
    }

    foreach ($line in Get-Content $Path) {
        if ($line -match "^\s*$([regex]::Escape($Key)):\s*(.*)\s*$") {
            $value = $Matches[1].Trim()
            if ($value.StartsWith("'") -and $value.EndsWith("'")) {
                $value = $value.Substring(1, $value.Length - 2).Replace("''", "'")
            }
            return $value
        }
    }

    return $null
}

function ConvertTo-YamlSingleQuoted {
    param([string]$Value)

    return "'" + $Value.Replace("'", "''") + "'"
}

function Normalize-BoolValue {
    param([string]$Value)

    switch ($Value.ToLowerInvariant()) {
        "true" { return "true" }
        "false" { return "false" }
        default { throw "Invalid discovery value in config: $Value" }
    }
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$rootDir = Split-Path -Parent $scriptDir
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$binaryName = "urlbridge-host-windows-$arch.exe"
$sourceBinary = Resolve-SourceBinary @(
    (Join-Path $scriptDir "urlbridge-host.exe"),
    (Join-Path $scriptDir $binaryName),
    (Join-Path $rootDir "urlbridge-host.exe"),
    (Join-Path $rootDir $binaryName),
    (Join-Path $rootDir "dist\$binaryName")
)

$installDir = Join-Path $env:LOCALAPPDATA "URLBridgeHost"
$installedBinary = Join-Path $installDir "urlbridge-host.exe"
$runnerScript = Join-Path $installDir "start-host.ps1"
$logFile = Join-Path $installDir "host.log"
$discovery = "true"

if (-not $ConfigPath) {
    $ConfigPath = Join-Path $installDir "config.yaml"
}

New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Copy-Item -Force $sourceBinary $installedBinary

if (Test-Path $ConfigPath) {
    if (-not $PSBoundParameters.ContainsKey("Listen")) {
        $existingListen = Get-YamlScalar -Path $ConfigPath -Key "listen_addr"
        if ($existingListen) {
            $Listen = $existingListen
        }
    }

    if (-not $PSBoundParameters.ContainsKey("Token")) {
        $existingToken = Get-YamlScalar -Path $ConfigPath -Key "token"
        if ($existingToken) {
            $Token = $existingToken
        }
    }

    $existingDiscovery = Get-YamlScalar -Path $ConfigPath -Key "discovery"
    if ($existingDiscovery) {
        $discovery = Normalize-BoolValue $existingDiscovery
    }
}

if (-not $Token) {
    $Token = (& $installedBinary token).Trim()
}

$configDir = Split-Path -Parent $ConfigPath
New-Item -ItemType Directory -Force -Path $configDir | Out-Null
$configContent = @"
listen_addr: $(ConvertTo-YamlSingleQuoted $Listen)
token: $(ConvertTo-YamlSingleQuoted $Token)
discovery: $discovery
"@
Set-Content -Path $ConfigPath -Value $configContent -Encoding ASCII

$installedBinaryLiteral = $installedBinary.Replace("'", "''")
$configPathLiteral = $ConfigPath.Replace("'", "''")
$runnerContent = @"
`$arguments = @('--config', '$configPathLiteral')
Start-Process -WindowStyle Hidden -FilePath '$installedBinaryLiteral' -ArgumentList `$arguments
"@
Set-Content -Path $runnerScript -Value $runnerContent -Encoding ASCII

$processes = Get-CimInstance Win32_Process -Filter "Name = 'urlbridge-host.exe'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -eq $installedBinary }
foreach ($process in $processes) {
    try {
        $process | Invoke-CimMethod -MethodName Terminate | Out-Null
    }
    catch {
    }
}

$arguments = @("--config", $ConfigPath)
Start-Process -WindowStyle Hidden -FilePath $installedBinary -ArgumentList $arguments | Out-Null

$runKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
if ($NoAutoStart) {
    Remove-ItemProperty -Path $runKey -Name "URLBridgeHost" -ErrorAction SilentlyContinue
    $autostartStatus = "disabled; started only for the current session"
}
else {
    $runCommand = "powershell.exe -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$runnerScript`""
    Set-ItemProperty -Path $runKey -Name "URLBridgeHost" -Value $runCommand
    $autostartStatus = "enabled at user logon"
}

"URL Bridge host installed."
"Installed binary: $installedBinary"
"Config file: $ConfigPath"
"Listen address: $Listen"
"Token: $Token"
"Autostart: $autostartStatus"
