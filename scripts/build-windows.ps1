param(
    [string]$OutputPath,
    [string]$BundleArchive,
    [string]$BundleVersion
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'Native Windows build requires PowerShell 7 on Windows.' }
$workspace = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
Set-Location -LiteralPath $workspace
. (Join-Path $PSScriptRoot 'legacy.ps1')
$taskWailsVersion = wails version
if ($LASTEXITCODE -ne 0 -or ($taskWailsVersion -join "`n") -notmatch 'v2\.17\.0\b') { throw 'Install the pinned stable Wails CLI: go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0' }
# Old v3 resources would duplicate icons/version resources in the new linker.
$oldResource = Join-Path $workspace $PheebKitLegacyResourceFilename
if (Test-Path -LiteralPath $oldResource) {
    $archive = Join-Path $workspace '.tools'
    New-Item -ItemType Directory -Force -Path $archive | Out-Null
    Move-Item -LiteralPath $oldResource -Destination (Join-Path $archive ('legacy-resource-' + [guid]::NewGuid().ToString('N') + '.syso'))
}
node scripts/sync-version.mjs
if ($LASTEXITCODE -ne 0) { throw 'Version synchronization failed' }
$taskRevision = (git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $taskRevision -notmatch '^[a-f0-9]{40}$') { throw 'Cannot determine build source revision' }
$taskDirty = git status --porcelain --untracked-files=normal
if ($LASTEXITCODE -ne 0) { throw 'Cannot determine build source status' }
$taskModified = if ($taskDirty) { 'true' } else { 'false' }
$taskLinkerFlags = "-X pheebkit/internal/buildinfo.Revision=$taskRevision -X pheebkit/internal/buildinfo.Modified=$taskModified"
$embeddedPayload = Join-Path $workspace 'internal/bundledffmpeg/runtime.zip'
$buildArguments = @('build', '-trimpath', '-webview2', 'error', '-skipbindings', '-nosyncgomod', '-nocolour', '-ldflags', $taskLinkerFlags)
if ($BundleArchive) {
    if (-not $BundleVersion) { throw 'Bundled builds require -BundleVersion.' }
    $taskBundleArchive = (Resolve-Path -LiteralPath $BundleArchive).Path
    if (-not $taskBundleArchive.StartsWith($workspace + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Bundled FFmpeg archive must remain inside this workspace.' }
    $taskBundleSHA256 = (Get-FileHash -LiteralPath $taskBundleArchive -Algorithm SHA256).Hash.ToLowerInvariant()
    $taskLinkerFlags += " -X pheebkit/internal/bundledffmpeg.PayloadSHA256=$taskBundleSHA256 -X pheebkit/internal/bundledffmpeg.DistributionVersion=$BundleVersion"
    $buildArguments[$buildArguments.Count - 1] = $taskLinkerFlags
    $buildArguments += @('-tags', 'pheebkit_bundled_ffmpeg')
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $embeddedPayload) | Out-Null
    if (Test-Path -LiteralPath $embeddedPayload) { throw 'Generated bundled FFmpeg payload is already present.' }
    Copy-Item -LiteralPath $taskBundleArchive -Destination $embeddedPayload
}
try {
    wails @buildArguments
    if ($LASTEXITCODE -ne 0) { throw 'Wails production build failed' }
} finally {
    if (Test-Path -LiteralPath $embeddedPayload) {
        Remove-Item -LiteralPath $embeddedPayload -Force
    }
}
New-Item -ItemType Directory -Force -Path (Join-Path $workspace 'bin') | Out-Null
if (-not $OutputPath) { $OutputPath = Join-Path $workspace 'bin/PheebKit.exe' }
$taskOutput = [IO.Path]::GetFullPath($OutputPath)
if (-not $taskOutput.StartsWith($workspace + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Build output must remain inside this workspace' }
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $taskOutput) | Out-Null
Copy-Item -LiteralPath (Join-Path $workspace 'build/bin/PheebKit.exe') -Destination $taskOutput
