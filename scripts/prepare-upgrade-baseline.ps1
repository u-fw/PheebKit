param([Parameter(Mandatory)][string]$Ref, [switch]$Published)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$workspace = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
Set-Location -LiteralPath $workspace
if ($Ref -notmatch '^v\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?$') { throw 'Upgrade baseline must be an exact version tag' }
git show-ref --verify --quiet "refs/tags/$Ref"
if ($LASTEXITCODE -ne 0) { throw 'Upgrade baseline tag is not available locally; fetch the reviewed tag first' }
if ($Published) {
    . (Join-Path $PSScriptRoot 'published-upgrade.ps1')
    Save-PublishedUpgrade -Ref $Ref -Workspace $workspace
    return
}
$toolsPrefix = [IO.Path]::GetFullPath((Join-Path $workspace '.tools')) + [IO.Path]::DirectorySeparatorChar
$checkout = [IO.Path]::GetFullPath((Join-Path $toolsPrefix ('upgrade-source-' + [guid]::NewGuid().ToString('N'))))
$destination = [IO.Path]::GetFullPath((Join-Path $toolsPrefix "upgrade-candidates/$Ref"))
if (-not $checkout.StartsWith($toolsPrefix, [StringComparison]::OrdinalIgnoreCase) -or -not $destination.StartsWith($toolsPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'Upgrade work must remain inside workspace tools' }
if (Test-Path -LiteralPath $destination) { throw 'Upgrade candidate already exists and will not be overwritten' }
$registered = $false
try {
    # Old tags omitted JS/HTML EOL rules; use their canonical LF source bytes.
    git -c core.eol=lf worktree add --detach $checkout "refs/tags/$Ref"
    if ($LASTEXITCODE -ne 0) { throw 'Cannot create isolated baseline checkout' }
    $registered = $true
    Set-Location -LiteralPath $checkout
    $oldVersion = (Get-Content VERSION -Raw).Trim()
    if ("v$oldVersion" -ne $Ref) { throw 'Baseline tag does not match its VERSION' }
    # Earlier tags expect their documented frontend bootstrap before Go tests.
    npm --prefix frontend ci
    if ($LASTEXITCODE -ne 0) { throw 'Baseline dependencies failed' }
    npm --prefix frontend run build
    if ($LASTEXITCODE -ne 0) { throw 'Baseline frontend bootstrap failed' }
    # Run only a deliberately selected project tag, with normal release checks.
    & (Join-Path $checkout 'scripts/release-windows.ps1') -InnoCompiler (Join-Path $workspace '.tools/inno-7.1.0/ISCC.exe')
    if ($LASTEXITCODE -ne 0) { throw 'Baseline release build failed' }
    New-Item -ItemType Directory -Path $destination -Force | Out-Null
    Get-ChildItem -LiteralPath (Join-Path $checkout "release/$oldVersion") -File | Copy-Item -Destination $destination
    $destination | Set-Content -LiteralPath (Join-Path $workspace '.tools/upgrade-baseline-path.txt') -Encoding utf8
    Write-Host "Upgrade baseline prepared: $Ref"
} finally {
    Set-Location -LiteralPath $workspace
    if ($registered) {
        if (-not [IO.Path]::GetFullPath($checkout).StartsWith($toolsPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'Refusing cleanup outside tools directory' }
        git worktree remove $checkout
        if ($LASTEXITCODE -ne 0) { throw "Baseline checkout retained for inspection: $checkout" }
    }
}
