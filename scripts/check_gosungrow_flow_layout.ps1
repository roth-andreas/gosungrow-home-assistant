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

    throw "Could not find Microsoft Edge or Google Chrome."
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$previewPath = Join-Path $repoRoot "tools\preview\gosungrow-dashboard-preview.html"
$previewUri = [System.Uri]::new((Resolve-Path $previewPath).Path)
$preferredEdge = "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe"
$preferredChrome = "$env:ProgramFiles\Google\Chrome\Application\chrome.exe"
$browser = if (Test-Path $preferredEdge) {
    $preferredEdge
} elseif (Test-Path $preferredChrome) {
    $preferredChrome
} else {
    Get-PreviewBrowser
}

$scenarios = @("export_high", "battery_charge", "evening_discharge", "grid_import")
$devices = @("desktop", "mobile")
$failures = @()

foreach ($scenario in $scenarios) {
    foreach ($device in $devices) {
        $url = "$($previewUri.AbsoluteUri)?view=overview&device=$device&scenario=$scenario&chrome=0&inspect=1"
        $stdoutPath = Join-Path $env:TEMP "gosungrow-flow-check-$scenario-$device-dom.txt"
        $stderrPath = Join-Path $env:TEMP "gosungrow-flow-check-$scenario-$device-err.txt"
        if (Test-Path $stdoutPath) { Remove-Item $stdoutPath -Force }
        if (Test-Path $stderrPath) { Remove-Item $stderrPath -Force }

        $process = Start-Process `
            -FilePath $browser `
            -ArgumentList @(
                "--headless=new",
                "--disable-gpu",
                "--allow-file-access-from-files",
                "--virtual-time-budget=2500",
                "--window-size=1600,1100",
                "--dump-dom",
                $url
            ) `
            -NoNewWindow `
            -Wait `
            -PassThru `
            -RedirectStandardOutput $stdoutPath `
            -RedirectStandardError $stderrPath

        $dom = if (Test-Path $stdoutPath) { Get-Content $stdoutPath -Raw } else { "" }

        if (-not $dom) {
            throw "No DOM output captured for $scenario / $device"
        }

        $match = [regex]::Match($dom, '<script[^>]*id="inspect-report"[^>]*>(?<json>[\s\S]*?)</script>')
        if (-not $match.Success) {
            throw "No inspect report found for $scenario / $device"
        }

        $report = $match.Groups["json"].Value | ConvertFrom-Json
        if ($report.ok) {
            Write-Host "[OK] $scenario / $device"
            continue
        }

        Write-Host "[WARN] $scenario / $device"
        foreach ($warning in $report.warnings) {
            Write-Host "  - $warning"
        }
        $failures += [PSCustomObject]@{
            Scenario = $scenario
            Device = $device
            Warnings = @($report.warnings)
        }
    }
}

if ($failures.Count -gt 0) {
    throw "Flow layout check found $($failures.Count) failing scenario/device combinations."
}
