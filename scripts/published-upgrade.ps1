Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-PublishedUpgradeAssets {
    param([object]$Release, [string]$Version)
    if ($Version -cnotmatch '^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(alpha|beta|rc)\.(0|[1-9]\d*))?$' -or
        $Release.tag_name -cne "v$Version" -or $Release.draft -isnot [bool] -or $Release.draft) {
        throw 'Invalid published upgrade identity.'
    }
    $expected = @(
        "PheebKit-$Version-windows-x64.exe",
        "PheebKit-$Version-windows-x64-ffmpeg.exe",
        "PheebKit-$Version-windows-x64-portable-ffmpeg.zip",
        "PheebKit-$Version-windows-x64-setup-ffmpeg.exe",
        'SHA256SUMS.txt'
    )
    if (@($Release.assets).Count -ne $expected.Count) { throw 'Published baseline must contain exactly five reviewed assets.' }
    $assets = [Collections.Generic.Dictionary[string, string]]::new([StringComparer]::Ordinal)
    foreach ($asset in $Release.assets) {
        if ($asset.name -cnotin $expected -or $assets.ContainsKey($asset.name) -or
            $asset.state -cne 'uploaded' -or $asset.digest -cnotmatch '^sha256:[a-f0-9]{64}$' -or
            $asset.browser_download_url -cne "https://github.com/u-fw/PheebKit/releases/download/v$Version/$($asset.name)" -or
            $asset.size -le 0 -or $asset.size -gt 800MB -or
            ($asset.name -ceq 'SHA256SUMS.txt' -and $asset.size -gt 4096)) {
            throw 'Published upgrade asset has an unreviewed name, origin, digest or size.'
        }
        $assets.Add($asset.name, $asset.digest.Substring(7))
    }
    return ,$assets
}

function Save-PublishedUpgrade {
    param([string]$Ref, [string]$Workspace)
    if ($Ref -cnotmatch '^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(alpha|beta|rc)\.(0|[1-9]\d*))?$') { throw 'Invalid published upgrade tag.' }
    . (Join-Path $PSScriptRoot 'release-upload.ps1')
    $version = $Ref.Substring(1)
    $revision = (& git rev-parse --verify "$Ref^{commit}").Trim()
    if ($LASTEXITCODE -ne 0 -or $revision -cnotmatch '^[a-f0-9]{40}$') { throw 'Cannot resolve the reviewed baseline tag.' }
    $response = & gh api "repos/u-fw/PheebKit/releases/tags/$Ref"
    if ($LASTEXITCODE -ne 0) { throw 'Cannot read the published baseline release.' }
    $release = $response | ConvertFrom-Json -Depth 12
    $assets = Get-PublishedUpgradeAssets -Release $release -Version $version
    $destination = [IO.Path]::GetFullPath((Join-Path $Workspace ".tools/upgrade-candidates/$Ref"))
    $prefix = [IO.Path]::GetFullPath((Join-Path $Workspace '.tools')) + [IO.Path]::DirectorySeparatorChar
    if (-not $destination.StartsWith($prefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'Baseline must remain inside workspace tools.' }
    if (Test-Path -LiteralPath $destination) { throw 'Upgrade candidate already exists and will not be overwritten.' }
    New-Item -ItemType Directory -Path $destination | Out-Null
    foreach ($name in $assets.Keys) {
        & gh release download $Ref --repo 'u-fw/PheebKit' --pattern $name --dir $destination
        if ($LASTEXITCODE -ne 0) { throw "Baseline download failed: $name" }
        $path = Join-Path $destination $name
        if ((Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant() -cne $assets[$name]) {
            throw "Published baseline digest mismatch: $name"
        }
    }
    # Validate all bytes, the checksum manifest and bounded inert ZIP metadata
    # against the exact tag before any installer or executable may run.
    $null = Get-VerifiedReleaseAssets -Directory $destination -Version $version -Revision $revision -ChecksumsSHA256 $assets['SHA256SUMS.txt']
    $destination | Set-Content -LiteralPath (Join-Path $Workspace '.tools/upgrade-baseline-path.txt') -Encoding utf8
    Write-Host "Verified published upgrade baseline: $Ref ($revision)"
}
