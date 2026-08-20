[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$localGoRoot = [System.IO.Path]::GetFullPath((Join-Path $projectRoot '.tools\go'))
$localGoExe = Join-Path $localGoRoot 'bin\go.exe'

if (Test-Path -LiteralPath $localGoExe -PathType Leaf) {
    $goRoot = $localGoRoot
    $goExe = $localGoExe
} else {
    $systemGo = Get-Command go.exe -ErrorAction SilentlyContinue
    if (-not $systemGo) {
        throw 'No project-local or system Go was found. Run .\scripts\bootstrap-dev.ps1 first.'
    }
    $goExe = $systemGo.Source
    $goRoot = (& $goExe env GOROOT 2>$null | Out-String).Trim()
    if ([string]::IsNullOrWhiteSpace($goRoot)) {
        throw "Could not determine GOROOT from $goExe."
    }
}

$goBin = Join-Path $goRoot 'bin'
$pathEntries = @($env:Path -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
if (-not ($pathEntries | Where-Object { $_.TrimEnd('\') -ieq $goBin.TrimEnd('\') })) {
    $env:Path = "$goBin;$env:Path"
}
$env:GOROOT = $goRoot

Write-Output "GOROOT=$env:GOROOT"
Write-Output "Go executable=$goExe"
& $goExe version
if ($LASTEXITCODE -ne 0) {
    throw 'The selected Go toolchain failed to execute.'
}
