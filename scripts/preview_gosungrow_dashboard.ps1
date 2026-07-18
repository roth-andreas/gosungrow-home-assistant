param(
    [ValidateSet("overview", "aggregates", "trends", "sources")]
    [string]$View = "overview",

    [ValidateSet("desktop", "mobile")]
    [string]$Device = "desktop",

    [ValidateSet("export_high", "battery_charge", "evening_discharge", "grid_import")]
    [string]$Scenario = "battery_charge",

    [ValidateSet("automatic", "warning", "dialog", "manual", "saved", "missing", "multi", "readonly", "save_error", "empty")]
    [string]$SourceState = "automatic",

    [ValidateSet("dark", "light")]
    [string]$Theme = "dark",

    [ValidateSet("en", "de", "sv")]
    [string]$Language = "en",

    [string]$OutputPath,

    [switch]$Open
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-PreviewBrowser {
    $commands = @("msedge.exe", "chrome.exe")
    foreach ($command in $commands) {
        $resolved = Get-Command $command -ErrorAction SilentlyContinue
        if ($resolved) {
            return $resolved.Source
        }
    }

    $candidates = @(
        "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe",
        "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe",
        "$env:ProgramFiles\Google\Chrome\Application\chrome.exe",
        "${env:ProgramFiles(x86)}\Google\Chrome\Application\chrome.exe"
    )

    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path $candidate)) {
            return $candidate
        }
    }

    throw "Could not find Microsoft Edge or Google Chrome for headless screenshot export."
}

function Resolve-OutputPath {
    param([string]$Path)

    $resolved = if ([System.IO.Path]::IsPathRooted($Path)) {
        $Path
    } else {
        Join-Path $repoRoot $Path
    }

    $directory = Split-Path -Parent $resolved
    if ($directory -and -not (Test-Path $directory)) {
        New-Item -ItemType Directory -Path $directory -Force | Out-Null
    }

    return $resolved
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$previewPath = Join-Path $repoRoot "tools\preview\gosungrow-dashboard-preview.html"

if (-not (Test-Path $previewPath)) {
    throw "Preview page not found at $previewPath"
}

$previewUri = [System.Uri]::new((Resolve-Path $previewPath).Path)
$url = "$($previewUri.AbsoluteUri)?view=$View&device=$Device&scenario=$Scenario&sourceState=$SourceState&theme=$Theme&language=$Language"

if ($Open -or -not $OutputPath) {
    Start-Process $url
}

if (-not $OutputPath) {
    return
}

$browser = Get-PreviewBrowser
$targetPath = Resolve-OutputPath -Path $OutputPath
$screenshotUrl = "$url&chrome=0"
$windowSize = if ($Device -eq "mobile") { "390,1350" } else { "1440,1100" }

& $browser `
    "--headless=new" `
    "--disable-gpu" `
    "--hide-scrollbars" `
    "--allow-file-access-from-files" `
    "--run-all-compositor-stages-before-draw" `
    "--virtual-time-budget=2500" `
    "--window-size=$windowSize" `
    "--screenshot=$targetPath" `
    $screenshotUrl

Write-Host "Wrote preview to $targetPath"
