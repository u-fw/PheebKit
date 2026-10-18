param([string]$Directory = '.tools/ci-ffmpeg')
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$workspace = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
Set-Location -LiteralPath $workspace
go run ./cmd/ci-ffmpeg -directory $Directory
if ($LASTEXITCODE -ne 0) { throw 'Pinned test-tool setup failed' }
$manifestPath = Join-Path ([IO.Path]::GetFullPath($Directory)) 'manifest.json'
$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
foreach ($name in @('ffmpeg', 'ffprobe')) {
    $path = $manifest.$name
    if (-not [IO.Path]::GetFullPath($path).StartsWith([IO.Path]::GetFullPath($Directory) + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Test binary escaped tools directory' }
    if ((Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant() -ne $manifest."$($name)SHA256") { throw 'Test binary digest mismatch' }
}
$env:PHEEBKIT_FFMPEG = $manifest.ffmpeg
$env:PHEEBKIT_FFPROBE = $manifest.ffprobe
if ($env:GITHUB_ENV) {
    "PHEEBKIT_FFMPEG=$($manifest.ffmpeg)" | Add-Content -LiteralPath $env:GITHUB_ENV -Encoding utf8
    "PHEEBKIT_FFPROBE=$($manifest.ffprobe)" | Add-Content -LiteralPath $env:GITHUB_ENV -Encoding utf8
}
Write-Host "Test tools verified: $($manifest.version). Not included in application packages."
