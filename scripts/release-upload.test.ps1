Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'release-upload.ps1')
. (Join-Path $PSScriptRoot 'published-upgrade.ps1')

$workspace = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$toolsRoot = [IO.Path]::GetFullPath((Join-Path $workspace '.tools')) + [IO.Path]::DirectorySeparatorChar
$root = [IO.Path]::GetFullPath((Join-Path $toolsRoot ('release-upload-tests-' + [guid]::NewGuid().ToString('N'))))
New-Item -ItemType Directory -Path $root | Out-Null
$priorLocation = Get-Location
$version = '0.2.0-rc.7'
$passed = 0

function Invoke-TestGit([string[]]$Arguments) {
    $output = @(& git @Arguments 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "Fixture git command failed: $($output -join ' ')" }
}

function Invoke-Case([string]$Name, [scriptblock]$Body) {
    & $Body
    $script:passed++
    Write-Output "PASS $Name"
}

function Assert-Refused([scriptblock]$Body, [string]$Message) {
    $errorMessage = ''
    try { & $Body | Out-Null } catch { $errorMessage = $_.Exception.Message }
    if (-not $errorMessage -or $errorMessage -notmatch $Message) {
        throw "Expected refusal matching '$Message'; got '$errorMessage'"
    }
}

function New-PublishedReleaseFixture {
    $names = @(
        "PheebKit-$version-windows-x64.exe",
        "PheebKit-$version-windows-x64-ffmpeg.exe",
        "PheebKit-$version-windows-x64-portable-ffmpeg.zip",
        "PheebKit-$version-windows-x64-setup-ffmpeg.exe",
        'SHA256SUMS.txt'
    )
    return [pscustomobject]@{
        tag_name = "v$version"
        draft = $false
        assets = @($names | ForEach-Object { [pscustomobject]@{
            name = $_
            state = 'uploaded'
            digest = 'sha256:' + ('a' * 64)
            size = 100
            browser_download_url = "https://github.com/u-fw/PheebKit/releases/download/v$version/$_"
        } })
    }
}

Invoke-Case 'accept complete published baseline identities without executing files' {
    $assets = Get-PublishedUpgradeAssets (New-PublishedReleaseFixture) $version
    if ($assets.Count -ne 5 -or $assets['SHA256SUMS.txt'] -cne ('a' * 64)) { throw 'Published asset identity differs.' }
}
foreach ($change in @('tag', 'draft', 'duplicate', 'origin', 'digest', 'size', 'checksum-size', 'incomplete', 'state')) {
    Invoke-Case "reject published baseline $change tampering" {
        $fixture = New-PublishedReleaseFixture
        switch ($change) {
            tag { $fixture.tag_name = 'v9.0.0' }
            draft { $fixture.draft = $true }
            duplicate { $fixture.assets[1] = $fixture.assets[0] }
            origin { $fixture.assets[0].browser_download_url = 'https://example.invalid/tool.exe' }
            digest { $fixture.assets[0].digest = '' }
            size { $fixture.assets[0].size = 801MB }
            checksum-size { $fixture.assets[4].size = 4097 }
            incomplete { $fixture.assets = @($fixture.assets | Select-Object -First 4) }
            state { $fixture.assets[0].state = 'new' }
        }
        Assert-Refused { Get-PublishedUpgradeAssets $fixture $version } 'Invalid published|Published (?:baseline|upgrade)'
    }
}

function New-ReleaseFixture {
    param([string]$Name, [scriptblock]$ChangeManifest, [switch]$DuplicateManifest, [switch]$DifferentPortableApp)
    $directory = Join-Path $root $Name
    New-Item -ItemType Directory -Path $directory | Out-Null
    $lightName = "PheebKit-$version-windows-x64.exe"
    $names = @($lightName, "PheebKit-$version-windows-x64-ffmpeg.exe", "PheebKit-$version-windows-x64-setup-ffmpeg.exe")
    foreach ($name in $names) { [IO.File]::WriteAllText((Join-Path $directory $name), "Inert test bytes: $name") }
    $manifest = [ordered]@{
        schemaVersion = 1
        version = $version
        sourceRevision = $revision
        modified = $false
        framework = 'wails-v2.17.0'
        distribution = 'windows-x64'
        executableSHA256 = (Get-FileHash -LiteralPath (Join-Path $directory $lightName) -Algorithm SHA256).Hash.ToLowerInvariant()
    }
    if ($ChangeManifest) { & $ChangeManifest $manifest }
    $zipPath = Join-Path $directory "PheebKit-$version-windows-x64-portable-ffmpeg.zip"
    $zip = [IO.Compression.ZipFile]::Open($zipPath, [IO.Compression.ZipArchiveMode]::Create)
    try {
        for ($i = 0; $i -lt $(if ($DuplicateManifest) { 2 } else { 1 }); $i++) {
            $writer = [IO.StreamWriter]::new($zip.CreateEntry('BUILD_INFO.json').Open())
            try { $writer.Write(($manifest | ConvertTo-Json -Compress)) } finally { $writer.Dispose() }
        }
        $writer = [IO.StreamWriter]::new($zip.CreateEntry('PheebKit.exe').Open())
        try {
            $writer.Write($(if ($DifferentPortableApp) { 'different inert bytes' } else { [IO.File]::ReadAllText((Join-Path $directory $lightName)) }))
        } finally { $writer.Dispose() }
    } finally { $zip.Dispose() }
    $checksums = @(Get-ChildItem -LiteralPath $directory -File | Sort-Object Name | ForEach-Object {
        '{0}  {1}' -f (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant(), $_.Name
    })
    $checksums | Set-Content -LiteralPath (Join-Path $directory 'SHA256SUMS.txt') -Encoding ascii
    return $directory
}

function Confirm-Fixture([string]$Directory, [string]$ExpectedChecksum) {
    if (-not $ExpectedChecksum) { $ExpectedChecksum = (Get-FileHash -LiteralPath (Join-Path $Directory 'SHA256SUMS.txt') -Algorithm SHA256).Hash.ToLowerInvariant() }
    @(Get-VerifiedReleaseAssets -Directory $Directory -Version $version -Revision $revision -ChecksumsSHA256 $ExpectedChecksum)
}

try {
    $repository = Join-Path $root 'repository'
    $remote = Join-Path $root 'origin.git'
    Invoke-TestGit @('init', '-q', '--bare', $remote)
    Invoke-TestGit @('init', '-q', $repository)
    Set-Location -LiteralPath $repository
    Invoke-TestGit @('remote', 'add', 'origin', $remote)
    [IO.File]::WriteAllText((Join-Path $repository 'fixture.txt'), 'first')
    Invoke-TestGit @('add', 'fixture.txt')
    $commitOptions = @('-c', 'user.name=PheebKit tests', '-c', 'user.email=tests@example.invalid', '-c', 'commit.gpgsign=false')
    Invoke-TestGit ($commitOptions + @('commit', '-qm', 'first fixture'))
    $revision = (& git rev-parse HEAD).Trim()
    Invoke-TestGit @('tag', "v$version")
    Invoke-TestGit @('push', '-q', 'origin', "refs/tags/v$version")

    Invoke-Case 'accept exactly five matching inert assets without running an executable' {
        $assets = @(Confirm-Fixture (New-ReleaseFixture 'valid'))
        if ($assets.Count -ne 5) { throw 'Expected five verified upload paths.' }
    }
    Invoke-Case 'reject a modified direct executable' {
        $directory = New-ReleaseFixture 'changed-exe'
        Add-Content -LiteralPath (Join-Path $directory "PheebKit-$version-windows-x64.exe") -Value 'changed'
        Assert-Refused { Confirm-Fixture $directory } 'digest mismatch'
    }
    Invoke-Case 'reject a checksum manifest differing from the verified job output' {
        $directory = New-ReleaseFixture 'changed-checksums'
        Assert-Refused { Confirm-Fixture $directory ('0' * 64) } 'manifest differs'
    }
    Invoke-Case 'reject unexpected asset names even when the file count matches' {
        $directory = New-ReleaseFixture 'unexpected'
        Rename-Item -LiteralPath (Join-Path $directory "PheebKit-$version-windows-x64.exe") -NewName 'other.exe'
        Assert-Refused { Confirm-Fixture $directory } 'unexpected file'
    }
    Invoke-Case 'reject directories masquerading as assets' {
        $directory = New-ReleaseFixture 'directory'
        $path = Join-Path $directory "PheebKit-$version-windows-x64.exe"
        Remove-Item -LiteralPath $path
        New-Item -ItemType Directory -Path $path | Out-Null
        Assert-Refused { Confirm-Fixture $directory } 'unexpected file'
    }
    Invoke-Case 'reject extra files' {
        $directory = New-ReleaseFixture 'extra'
        [IO.File]::WriteAllText((Join-Path $directory 'extra.txt'), 'extra')
        Assert-Refused { Confirm-Fixture $directory } 'exactly five'
    }
    Invoke-Case 'reject duplicate checksum entries' {
        $directory = New-ReleaseFixture 'duplicate-hash'
        $path = Join-Path $directory 'SHA256SUMS.txt'
        $lines = @(Get-Content -LiteralPath $path)
        @($lines + $lines[0]) | Set-Content -LiteralPath $path -Encoding ascii
        Assert-Refused { Confirm-Fixture $directory } 'duplicate'
    }
    Invoke-Case 'reject missing checksum entries' {
        $directory = New-ReleaseFixture 'missing-hash'
        $path = Join-Path $directory 'SHA256SUMS.txt'
        @(Get-Content -LiteralPath $path)[0..2] | Set-Content -LiteralPath $path -Encoding ascii
        Assert-Refused { Confirm-Fixture $directory } 'incomplete'
    }
    Invoke-Case 'reject checksum path traversal' {
        $directory = New-ReleaseFixture 'traversal'
        ('a' * 64) + '  ../file.exe' | Set-Content -LiteralPath (Join-Path $directory 'SHA256SUMS.txt') -Encoding ascii
        Assert-Refused { Confirm-Fixture $directory } 'Invalid release checksum'
    }
    foreach ($change in @('sourceRevision', 'version', 'modified', 'framework', 'executableSHA256')) {
        Invoke-Case "reject mismatched $change in the package manifest" {
            $directory = New-ReleaseFixture "bad-$change" -ChangeManifest {
                param($manifest)
                $manifest[$change] = if ($change -eq 'modified') { $true } else { 'mismatch' }
            }
            Assert-Refused { Confirm-Fixture $directory } 'manifest does not match'
        }
    }
    Invoke-Case 'reject oversized build metadata' {
        $directory = New-ReleaseFixture 'oversized' -ChangeManifest { param($manifest) $manifest['padding'] = 'x' * 20000 }
        Assert-Refused { Confirm-Fixture $directory } 'bounded build manifest'
    }
    Invoke-Case 'reject duplicate ZIP manifests' {
        $directory = New-ReleaseFixture 'duplicate-manifest' -DuplicateManifest
        Assert-Refused { Confirm-Fixture $directory } 'one bounded build manifest'
    }
    Invoke-Case 'reject a portable application with different bytes' {
        $directory = New-ReleaseFixture 'portable-differs' -DifferentPortableApp
        Assert-Refused { Confirm-Fixture $directory } 'Portable application differs'
    }
    Invoke-Case 'accept a matching lightweight tag' { Assert-ReleaseTag -Tag "v$version" -Version $version -Revision $revision }
    Invoke-Case 'accept a matching annotated tag' {
        Invoke-TestGit ($commitOptions + @('-c', 'tag.gpgsign=false', 'tag', '-a', 'v0.2.0-rc.8', '-m', 'annotated fixture'))
        Invoke-TestGit @('push', '-q', 'origin', 'refs/tags/v0.2.0-rc.8')
        Assert-ReleaseTag -Tag 'v0.2.0-rc.8' -Version '0.2.0-rc.8' -Revision $revision
    }
    Invoke-Case 'reject an invalid or missing tag' {
        Assert-Refused { Assert-ReleaseTag -Tag '--help' -Version $version -Revision $revision } 'Invalid release identity'
        Assert-Refused { Assert-ReleaseTag -Tag 'v0.2.0-rc.9' -Version '0.2.0-rc.9' -Revision $revision } 'Cannot verify'
    }
    Invoke-Case 'reject a moved remote tag while publisher remains pinned' {
        [IO.File]::WriteAllText((Join-Path $repository 'fixture.txt'), 'second')
        Invoke-TestGit @('add', 'fixture.txt')
        Invoke-TestGit ($commitOptions + @('commit', '-qm', 'second fixture'))
        Invoke-TestGit @('tag', '-f', "v$version")
        Invoke-TestGit @('push', '-q', '--force', 'origin', "refs/tags/v$version")
        Assert-Refused { Assert-ReleaseTag -Tag "v$version" -Version $version -Revision $revision } 'Publisher source differs'
        Invoke-TestGit @('switch', '-q', '--detach', $revision)
        Assert-Refused { Assert-ReleaseTag -Tag "v$version" -Version $version -Revision $revision } 'tag moved'
    }
    Invoke-Case 'reject a moved annotated tag while publisher remains pinned' {
        $nextRevision = (& git rev-parse --verify "refs/tags/v$version").Trim()
        if ($LASTEXITCODE -ne 0) { throw 'Cannot locate the second fixture commit.' }
        Invoke-TestGit ($commitOptions + @('-c', 'tag.gpgsign=false', 'tag', '-fa', 'v0.2.0-rc.8', $nextRevision, '-m', 'moved annotated fixture'))
        Invoke-TestGit @('push', '-q', '--force', 'origin', 'refs/tags/v0.2.0-rc.8')
        Assert-Refused { Assert-ReleaseTag -Tag 'v0.2.0-rc.8' -Version '0.2.0-rc.8' -Revision $revision } 'tag moved'
    }
    Write-Output "$passed upload and tag checks passed."
} finally {
    Set-Location -LiteralPath $priorLocation
    if (-not [IO.Path]::GetFullPath($root).StartsWith($toolsRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Refusing fixture cleanup outside workspace tools.' }
    if (Test-Path -LiteralPath $root) { Remove-Item -LiteralPath $root -Recurse -Force }
}
