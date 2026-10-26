Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Assert-ReleaseTag {
    param([string]$Tag, [string]$Version, [string]$Revision)
    if ($Version -notmatch '^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(alpha|beta|rc)\.(0|[1-9]\d*))?$' -or
        $Tag -cne "v$Version" -or $Revision -cnotmatch '^[a-f0-9]{40}$') {
        throw 'Invalid release identity.'
    }
    $head = & git rev-parse HEAD
    if ($LASTEXITCODE -ne 0 -or $head -cne $Revision) { throw 'Publisher source differs from the verified build revision.' }
    $references = @(& git ls-remote --exit-code origin "refs/tags/$Tag" "refs/tags/$Tag^{}")
    if ($LASTEXITCODE -ne 0) { throw 'Cannot verify the remote release tag.' }
    $tagRevision = ''
    $peeledRevision = ''
    foreach ($reference in $references) {
        $parts = $reference -split "`t", 2
        if ($parts.Count -ne 2 -or $parts[0] -cnotmatch '^[a-f0-9]{40}$') { throw 'Invalid remote tag response.' }
        if ($parts[1] -ceq "refs/tags/$Tag") { $tagRevision = $parts[0] }
        elseif ($parts[1] -ceq "refs/tags/$Tag^{}") { $peeledRevision = $parts[0] }
        else { throw 'Unexpected remote tag reference.' }
    }
    $actual = if ($peeledRevision) { $peeledRevision } else { $tagRevision }
    if (-not $tagRevision -or $actual -cne $Revision) { throw 'Remote release tag moved away from the verified build revision.' }
}

function Get-VerifiedReleaseAssets {
    param(
        [string]$Directory,
        [string]$Version,
        [string]$Revision,
        [string]$ChecksumsSHA256
    )
    if ($Version -notmatch '^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(alpha|beta|rc)\.(0|[1-9]\d*))?$' -or
        $Revision -cnotmatch '^[a-f0-9]{40}$' -or $ChecksumsSHA256 -cnotmatch '^[a-f0-9]{64}$') {
        throw 'Invalid verified artifact identity.'
    }
    $names = @(
        "PheebKit-$Version-windows-x64.exe",
        "PheebKit-$Version-windows-x64-ffmpeg.exe",
        "PheebKit-$Version-windows-x64-portable-ffmpeg.zip",
        "PheebKit-$Version-windows-x64-setup-ffmpeg.exe"
    )
    $expected = $names + 'SHA256SUMS.txt'
    $entries = @(Get-ChildItem -LiteralPath $Directory -Force)
    if ($entries.Count -ne $expected.Count) { throw 'Expected exactly five release assets.' }
    foreach ($entry in $entries) {
        if ($entry.PSIsContainer -or ($entry.Attributes -band [IO.FileAttributes]::ReparsePoint) -or $entry.Name -cnotin $expected) {
            throw 'Release contains an unexpected file, directory or link.'
        }
    }
    $checksumPath = Join-Path $Directory 'SHA256SUMS.txt'
    if ((Get-Item -LiteralPath $checksumPath).Length -gt 4096 -or
        (Get-FileHash -LiteralPath $checksumPath -Algorithm SHA256).Hash.ToLowerInvariant() -cne $ChecksumsSHA256) {
        throw 'Checksum manifest differs from the verified build output.'
    }
    $hashes = @{}
    foreach ($line in Get-Content -LiteralPath $checksumPath) {
        if ($line -cnotmatch '^([a-f0-9]{64})  ([^/\\]+)$') { throw 'Invalid release checksum entry.' }
        $digest, $name = $Matches[1], $Matches[2]
        if ($name -cnotin $names -or $hashes.ContainsKey($name)) { throw 'Unexpected or duplicate release checksum entry.' }
        if ((Get-FileHash -LiteralPath (Join-Path $Directory $name) -Algorithm SHA256).Hash.ToLowerInvariant() -cne $digest) {
            throw "Release asset digest mismatch: $name"
        }
        $hashes[$name] = $digest
    }
    if ($hashes.Count -ne $names.Count) { throw 'Release checksum manifest is incomplete.' }

    $zip = [IO.Compression.ZipFile]::OpenRead((Join-Path $Directory $names[2]))
    try {
        $manifests = @($zip.Entries | Where-Object FullName -CEQ 'BUILD_INFO.json')
        $applications = @($zip.Entries | Where-Object FullName -CEQ 'PheebKit.exe')
        if ($manifests.Count -ne 1 -or $manifests[0].Length -gt 16384 -or $applications.Count -ne 1) {
            throw 'Portable ZIP requires one bounded build manifest and application.'
        }
        $reader = [IO.StreamReader]::new($manifests[0].Open(), [Text.UTF8Encoding]::new($false, $true))
        try {
            $buffer = [char[]]::new(16385)
            $count = $reader.ReadBlock($buffer, 0, $buffer.Length)
            if ($count -gt 16384) { throw 'Portable build manifest exceeds the size limit.' }
            $manifest = [string]::new($buffer, 0, $count) | ConvertFrom-Json -Depth 8
        } finally { $reader.Dispose() }
        if ($manifest.schemaVersion -ne 1 -or $manifest.version -cne $Version -or
            $manifest.sourceRevision -cne $Revision -or $manifest.modified -isnot [bool] -or $manifest.modified -ne $false -or
            $manifest.framework -cne 'wails-v2.17.0' -or $manifest.distribution -cne 'windows-x64' -or
            $manifest.executableSHA256 -cne $hashes[$names[0]]) {
            throw 'Portable build manifest does not match the verified source and executable.'
        }
        $appStream = $applications[0].Open()
        try { $appHash = (Get-FileHash -InputStream $appStream -Algorithm SHA256).Hash.ToLowerInvariant() }
        finally { $appStream.Dispose() }
        if ($appHash -cne $hashes[$names[0]]) { throw 'Portable application differs from the direct lightweight executable.' }
    } finally { $zip.Dispose() }
    foreach ($name in $expected) { (Get-Item -LiteralPath (Join-Path $Directory $name)).FullName }
}
