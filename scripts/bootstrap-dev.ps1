[CmdletBinding()]
param(
    [string]$ProjectRoot = (Split-Path -Parent $PSScriptRoot)
)

$ErrorActionPreference = 'Stop'

$isWindowsHost = [System.Environment]::OSVersion.Platform -eq [System.PlatformID]::Win32NT
if (-not $isWindowsHost) {
    throw 'OpenList 115 Media Drive development bootstrap requires Windows.'
}

$architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
if ($architecture -ne 'X64') {
    throw "Only Windows x64 is supported by this bootstrap script; detected $architecture."
}

$toolchainLine = Select-String -LiteralPath (Join-Path $ProjectRoot 'core\go.mod') -Pattern '^\s*toolchain\s+go(?<version>[0-9]+\.[0-9]+\.[0-9]+)\s*$' | Select-Object -First 1
if (-not $toolchainLine) {
    throw 'Could not determine the required Go toolchain from core/go.mod.'
}

$goVersion = $toolchainLine.Matches[0].Groups['version'].Value
$goRoot = [System.IO.Path]::GetFullPath((Join-Path $ProjectRoot '.tools\go'))
$projectRootFull = [System.IO.Path]::GetFullPath($ProjectRoot).TrimEnd('\')
if (-not $goRoot.StartsWith("$projectRootFull\", [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to operate outside the project .tools directory: $goRoot"
}

$goExe = Join-Path $goRoot 'bin\go.exe'
$downloadsRoot = [System.IO.Path]::GetFullPath((Join-Path $ProjectRoot '.tools\downloads'))
New-Item -ItemType Directory -Force -Path $downloadsRoot | Out-Null

function Get-GoVersionText([string]$Executable) {
    $text = (& $Executable version 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) {
        return $null
    }
    return $text
}

if (Test-Path -LiteralPath $goExe -PathType Leaf) {
    $localVersion = Get-GoVersionText $goExe
    if ($localVersion -match "\bgo$([regex]::Escape($goVersion))(\s|$)") {
        Write-Output "Go portable path: $goRoot"
        Write-Output "Go version: $localVersion"
        Write-Output 'Source: existing project-local toolchain'
        exit 0
    }
}

$systemGo = Get-Command go.exe -ErrorAction SilentlyContinue
if ($systemGo) {
    $systemVersion = Get-GoVersionText $systemGo.Source
    if ($systemVersion -match "\bgo$([regex]::Escape($goVersion))(\s|$)") {
        $systemRoot = (& $systemGo.Source env GOROOT 2>$null | Out-String).Trim()
        Write-Output "Go portable path: $systemRoot"
        Write-Output "Go version: $systemVersion"
        Write-Output 'Source: compatible system Go (no system changes made)'
        exit 0
    }
}

$metadataUri = 'https://go.dev/dl/?mode=json&include=all'
Write-Output "Reading official Go release metadata: $metadataUri"
$releases = Invoke-RestMethod -Uri $metadataUri -Method Get
$release = $releases | Where-Object { $_.version -eq "go$goVersion" } | Select-Object -First 1
if (-not $release) {
    throw "Go release go$goVersion was not found in official go.dev metadata."
}

$asset = $release.files | Where-Object {
    $_.filename -eq "go$goVersion.windows-amd64.zip" -and $_.kind -eq 'archive'
} | Select-Object -First 1
if (-not $asset -or [string]::IsNullOrWhiteSpace($asset.sha256)) {
    throw "Official metadata has no Windows amd64 archive and SHA-256 for go$goVersion."
}

$archivePath = Join-Path $downloadsRoot $asset.filename
$expectedSha = $asset.sha256.ToLowerInvariant()
if (Test-Path -LiteralPath $archivePath -PathType Leaf) {
    $existingSha = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($existingSha -ne $expectedSha) {
        Remove-Item -LiteralPath $archivePath -Force
    }
}

if (-not (Test-Path -LiteralPath $archivePath -PathType Leaf)) {
    $downloadUri = "https://go.dev/dl/$($asset.filename)"
    Write-Output "Downloading official Go archive: $downloadUri"
    Invoke-WebRequest -Uri $downloadUri -OutFile $archivePath
}

$actualSha = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actualSha -ne $expectedSha) {
    Remove-Item -LiteralPath $archivePath -Force
    throw "SHA-256 mismatch for $($asset.filename). Expected $expectedSha, got $actualSha."
}

$extractRoot = Join-Path $downloadsRoot "extract-go-$goVersion"
if (Test-Path -LiteralPath $extractRoot) {
    Remove-Item -LiteralPath $extractRoot -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $extractRoot | Out-Null
Expand-Archive -LiteralPath $archivePath -DestinationPath $extractRoot -Force
$extractedGoRoot = Join-Path $extractRoot 'go'
if (-not (Test-Path -LiteralPath (Join-Path $extractedGoRoot 'bin\go.exe') -PathType Leaf)) {
    throw 'The official Go archive did not contain go\bin\go.exe.'
}

if (Test-Path -LiteralPath $goRoot) {
    Remove-Item -LiteralPath $goRoot -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $goRoot | Out-Null
Get-ChildItem -LiteralPath $extractedGoRoot -Force | Copy-Item -Destination $goRoot -Recurse -Force

$installedVersion = Get-GoVersionText $goExe
if ($installedVersion -notmatch "\bgo$([regex]::Escape($goVersion))(\s|$)") {
    throw "Portable Go installation completed but reported an unexpected version: $installedVersion"
}

Write-Output "Go portable path: $goRoot"
Write-Output "Go version: $installedVersion"
Write-Output "SHA-256: $actualSha"
Write-Output 'Source: official go.dev release metadata and archive'
