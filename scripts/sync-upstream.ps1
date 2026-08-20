[CmdletBinding()]
param(
    [switch]$Help,
    [switch]$Apply
)

$ErrorActionPreference = 'Stop'

$coreUrl = 'https://github.com/OpenListTeam/OpenList.git'
$desktopUrl = 'https://github.com/OpenListTeam/OpenList-Desktop.git'
$sdkUrl = 'https://github.com/OpenListTeam/115-sdk-go.git'

if ($Help) {
    @"
OpenList 115 Media Drive upstream sync helper

Core upstream:    OpenListTeam/OpenList
Desktop upstream: OpenListTeam/OpenList-Desktop
SDK upstream:     OpenListTeam/115-sdk-go

Default behavior: dry run; prints recommended commands only.
Apply behavior:   .\scripts\sync-upstream.ps1 -Apply

Recommended commands:
  git subtree pull --prefix=core $coreUrl main --squash
  git subtree pull --prefix=desktop $desktopUrl main --squash
  git subtree pull --prefix=third_party/115-sdk-go $sdkUrl main --squash

The Apply mode requires a clean working tree and never uses force push,
reset --hard, checkout, or automatic conflict overwrite.
"@
    exit 0
}

$commands = @(
    "git subtree pull --prefix=core $coreUrl main --squash",
    "git subtree pull --prefix=desktop $desktopUrl main --squash",
    "git subtree pull --prefix=third_party/115-sdk-go $sdkUrl main --squash"
)

if (-not $Apply) {
    Write-Output 'DRY RUN: no upstream sync was executed.'
    $commands | ForEach-Object { Write-Output $_ }
    exit 0
}

$status = git status --porcelain
if (-not [string]::IsNullOrWhiteSpace(($status | Out-String))) {
    throw 'Apply mode requires a clean working tree.'
}

& git subtree pull --prefix=core $coreUrl main --squash
if ($LASTEXITCODE -ne 0) { throw 'Core subtree pull failed.' }
& git subtree pull --prefix=desktop $desktopUrl main --squash
if ($LASTEXITCODE -ne 0) { throw 'Desktop subtree pull failed.' }
& git subtree pull --prefix=third_party/115-sdk-go $sdkUrl main --squash
if ($LASTEXITCODE -ne 0) { throw '115 SDK subtree pull failed.' }
