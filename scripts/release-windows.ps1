param(
    [string]$InnoCompiler = $env:PHEEBKIT_ISCC,
    [string]$StableEvidence,
    [string]$FFmpegDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'Windows x64 release packaging requires native Windows PowerShell 7.' }
if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -ne 'X64') { throw 'Only Windows x64 releases are currently supported.' }
$workspace = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
Set-Location -LiteralPath $workspace

$taskNodeVersion = node --version
if ($LASTEXITCODE -ne 0 -or $taskNodeVersion -notmatch '^v(\d+)\.' -or [int]$Matches[1] -lt 24) {
    throw 'Release packaging requires Node.js 24 or newer on PATH.'
}
$taskGoVersion = go env GOVERSION
if ($LASTEXITCODE -ne 0 -or $taskGoVersion -notmatch '^go(\d+\.\d+\.\d+)$' -or [Version]$Matches[1] -lt [Version]'1.27.3') { throw 'Security release builds require stable Go 1.27.3 or newer; enable automatic toolchain selection.' }

$expectedArchiveSHA256 = '759d0a9831c436a0eb331ad36f236c06cb04aaa0005da46571f0e9d3d9206f6b'
$expectedFFmpegSHA256 = '589e50b766d251afdf181dd664d40bd94407e200b019989fd7468c7d118a28d0'
$expectedFFprobeSHA256 = '02264dfa4f56baaa31e443ece8aa16d397f6b008200e7fd184d22b9009e75fd3'
$ffmpegVersion = '9.0.2'
$ffmpegPackage = 'gyan-9.0.2'

function Invoke-Checked([string]$Program, [string[]]$Arguments) {
    & $Program @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Program failed with exit code $LASTEXITCODE" }
}

function File-SHA256([string]$Path) {
    return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function Resolve-ReleaseFFmpeg {
    param([string]$Directory)
    $ffmpeg = ''
    $ffprobe = ''
    $archiveSHA256 = $expectedArchiveSHA256
    if ($Directory) {
        $root = (Resolve-Path -LiteralPath $Directory).Path
        $ffmpeg = Join-Path $root 'bin/ffmpeg.exe'
        $ffprobe = Join-Path $root 'bin/ffprobe.exe'
    } elseif ($env:PHEEBKIT_FFMPEG -and $env:PHEEBKIT_FFPROBE) {
        $ffmpeg = (Resolve-Path -LiteralPath $env:PHEEBKIT_FFMPEG).Path
        $ffprobe = (Resolve-Path -LiteralPath $env:PHEEBKIT_FFPROBE).Path
        $root = Split-Path -Parent (Split-Path -Parent $ffmpeg)
    } elseif (Test-Path -LiteralPath '.tools/ci-ffmpeg/manifest.json') {
        $manifest = Get-Content -LiteralPath '.tools/ci-ffmpeg/manifest.json' -Raw | ConvertFrom-Json
        if ($manifest.package -ne $ffmpegPackage -or $manifest.version -ne $ffmpegVersion) { throw 'Pinned test-tool manifest has an unexpected FFmpeg package.' }
        $ffmpeg = [IO.Path]::GetFullPath($manifest.ffmpeg)
        $ffprobe = [IO.Path]::GetFullPath($manifest.ffprobe)
        $archiveSHA256 = $manifest.archiveSHA256
        $root = Split-Path -Parent (Split-Path -Parent $ffmpeg)
    } else {
        throw 'Prepared publisher-pinned FFmpeg is required. Set PHEEBKIT_FFMPEG/PHEEBKIT_FFPROBE, pass -FFmpegDirectory, or run scripts/setup-test-ffmpeg.ps1 into a fresh directory.'
    }
    foreach ($item in @(
        @{ Name = 'ffmpeg'; Path = $ffmpeg; Expected = $expectedFFmpegSHA256 },
        @{ Name = 'ffprobe'; Path = $ffprobe; Expected = $expectedFFprobeSHA256 }
    )) {
        if (-not (Test-Path -LiteralPath $item.Path -PathType Leaf)) { throw "Missing bundled $($item.Name).exe." }
        $actual = File-SHA256 $item.Path
        if ($actual -ne $item.Expected) { throw "Bundled $($item.Name).exe digest does not match the publisher-pinned full build." }
    }
    if ($archiveSHA256 -ne $expectedArchiveSHA256) { throw 'FFmpeg archive digest does not match the reviewed builder package.' }
    foreach ($notice in @('LICENSE', 'README.txt')) {
        if (-not (Test-Path -LiteralPath (Join-Path $root $notice) -PathType Leaf)) { throw "Pinned FFmpeg package is missing $notice." }
    }
    if (-not (Test-Path -LiteralPath (Join-Path $root 'presets') -PathType Container)) { throw 'Pinned FFmpeg package is missing its preset metadata.' }
    $ffmpegVersionOutput = @(& $ffmpeg -version)
    if ($LASTEXITCODE -ne 0 -or -not $ffmpegVersionOutput[0].StartsWith('ffmpeg version 9.0.2-full_build-www.gyan.dev ')) { throw 'Verified ffmpeg.exe does not report the publisher-pinned 9.0.2 full build.' }
    $ffprobeVersionOutput = @(& $ffprobe -version)
    if ($LASTEXITCODE -ne 0 -or -not $ffprobeVersionOutput[0].StartsWith('ffprobe version 9.0.2-full_build-www.gyan.dev ')) { throw 'Verified ffprobe.exe does not report the publisher-pinned 9.0.2 full build.' }
    return [pscustomobject]@{
        Root = $root
        FFmpeg = $ffmpeg
        FFprobe = $ffprobe
        VersionLine = $ffmpegVersionOutput[0]
        ArchiveSHA256 = $expectedArchiveSHA256
    }
}

function Assert-BuildProvenance {
    param(
        [string]$Executable,
        [string]$Revision,
        [bool]$ExpectBundledFFmpeg,
        [string]$ExpectedPayloadSHA256
    )
    $actual = & $Executable --build-info | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0 -or $actual.schemaVersion -ne 1 -or $actual.version -ne $version.version -or $actual.sourceRevision -ne $Revision -or $actual.modified -ne $false -or $actual.framework -ne 'wails-v2.17.0') {
        throw 'Executable provenance does not match reviewed source.'
    }
    $bundledProperty = $actual.PSObject.Properties['bundledFFmpeg']
    if ($ExpectBundledFFmpeg) {
        if (-not $bundledProperty -or $bundledProperty.Value.kind -ne 'ffmpeg' -or $bundledProperty.Value.payloadSHA256 -ne $ExpectedPayloadSHA256 -or $bundledProperty.Value.distributionVersion -ne $ffmpegVersion -or $bundledProperty.Value.extractedAtFirstLaunch -ne $true) {
            throw 'Bundled executable does not report the reviewed embedded FFmpeg payload.'
        }
    } elseif ($bundledProperty) {
        throw 'Lightweight executable unexpectedly reports an embedded FFmpeg payload.'
    }
    return $actual
}

function Write-BuildManifest {
    param(
        [string]$Executable,
        [string]$Destination,
        [string]$Revision,
        [bool]$BundledFFmpeg,
        [object]$RuntimeMetadata
    )
    $manifest = [ordered]@{
        schemaVersion = 1
        version = $version.version
        sourceRevision = $Revision
        modified = $false
        framework = 'wails-v2.17.0'
        distribution = $(if ($BundledFFmpeg) { 'windows-x64-ffmpeg' } else { 'windows-x64' })
        executableSHA256 = File-SHA256 $Executable
    }
    if ($BundledFFmpeg) {
        $manifest.bundledFFmpeg = [ordered]@{
            kind = 'ffmpeg'
            payloadSHA256 = File-SHA256 $bundleArchive
            distributionVersion = $RuntimeMetadata.version
            provider = $RuntimeMetadata.provider
            package = $RuntimeMetadata.package
            archiveSHA256 = $RuntimeMetadata.archiveSHA256
            ffmpegSHA256 = $RuntimeMetadata.ffmpegSHA256
            ffprobeSHA256 = $RuntimeMetadata.ffprobeSHA256
            correspondingSource = $RuntimeMetadata.correspondingSource
            extractedAtFirstLaunch = $true
        }
    }
    $manifest | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $Destination -Encoding utf8NoBOM
}

function Copy-SharedPayloadFiles {
    param([string]$Destination, [bool]$BundledFFmpeg)
    New-Item -ItemType Directory -Force -Path $Destination | Out-Null
    Copy-Item -LiteralPath (Join-Path $workspace 'LICENSE') -Destination (Join-Path $Destination 'LICENSE.txt')
    if ((Get-Content -LiteralPath (Join-Path $workspace 'RELEASE_NOTES.md') -TotalCount 1) -ne "# PheebKit $($version.version)") { throw 'Release notes do not match VERSION.' }
    Copy-Item -LiteralPath (Join-Path $workspace 'RELEASE_NOTES.md') -Destination (Join-Path $Destination 'RELEASE_NOTES.md')
    $noticeArguments = @('scripts/dependency-notices.mjs', (Join-Path $Destination 'THIRD_PARTY_NOTICES.txt'))
    if ($BundledFFmpeg) { $noticeArguments += '--bundled-ffmpeg' }
    Invoke-Checked node $noticeArguments
}

$versionInfo = node --input-type=module -e 'import { readVersion } from "./scripts/version.mjs"; console.log(JSON.stringify(readVersion()));'
if ($LASTEXITCODE -ne 0) { throw 'Invalid VERSION file' }
$version = $versionInfo | ConvertFrom-Json
if (-not $version.prerelease) {
    if (-not $StableEvidence) { throw 'Stable packaging requires reviewed acceptance records via -StableEvidence. See docs/stability.md.' }
    Invoke-Checked node @('scripts/stable-evidence.mjs', $StableEvidence)
}
$releaseRoot = Join-Path $workspace 'release'
$final = Join-Path $releaseRoot $version.version
$resolvedFinal = [IO.Path]::GetFullPath($final)
if (-not $resolvedFinal.StartsWith([IO.Path]::GetFullPath($releaseRoot) + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Release output must stay in the workspace release directory.' }
if (Test-Path -LiteralPath $final) { throw "Release directory exists and will not be overwritten: $final" }
if (-not $InnoCompiler) {
    foreach ($candidate in @((Join-Path $workspace '.tools/inno-7.1.0/ISCC.exe'), 'C:/Program Files/Inno Setup 7/ISCC.exe')) {
        if (Test-Path -LiteralPath $candidate) { $InnoCompiler = $candidate; break }
    }
}
if (-not $InnoCompiler -or -not (Test-Path -LiteralPath $InnoCompiler)) { throw 'Install verified Inno Setup 7.1.0 or supply -InnoCompiler / PHEEBKIT_ISCC.' }
$InnoCompiler = (Resolve-Path -LiteralPath $InnoCompiler).Path
$sourceFFmpeg = Resolve-ReleaseFFmpeg $FFmpegDirectory
New-Item -ItemType Directory -Force -Path $releaseRoot | Out-Null
$stage = Join-Path $releaseRoot ('.stage-' + [guid]::NewGuid().ToString('N'))
$output = Join-Path $stage 'output'
New-Item -ItemType Directory -Path $stage, $output | Out-Null
try {
    Invoke-Checked node @('scripts/sync-version.mjs')
    $dirty = git status --porcelain --untracked-files=normal
    if ($LASTEXITCODE -ne 0 -or $dirty) { throw 'Release packaging requires committed sources and synchronized version metadata; commit changes before packaging.' }
    Invoke-Checked node @('--test', 'scripts/version.test.mjs', 'scripts/stable-evidence.test.mjs')
    Invoke-Checked npm @('--prefix', 'frontend', 'ci')
    Invoke-Checked npm @('--prefix', 'frontend', 'run', 'build')
    Invoke-Checked go @('test', './...')
    Invoke-Checked go @('vet', './...')
    Invoke-Checked go @('run', 'golang.org/x/vuln/cmd/govulncheck@v1.9.0', './...')
    Invoke-Checked npm @('--prefix', 'frontend', 'audit', '--audit-level=low')
    foreach ($check in @('test', 'typecheck', 'lint', 'format:check')) { Invoke-Checked npm @('--prefix', 'frontend', 'run', $check) }

    $runtimeRoot = Join-Path $stage 'runtime/ffmpeg'
    New-Item -ItemType Directory -Path (Join-Path $runtimeRoot 'bin') -Force | Out-Null
    Copy-Item -LiteralPath $sourceFFmpeg.FFmpeg -Destination (Join-Path $runtimeRoot 'bin/ffmpeg.exe')
    Copy-Item -LiteralPath $sourceFFmpeg.FFprobe -Destination (Join-Path $runtimeRoot 'bin/ffprobe.exe')
    Copy-Item -LiteralPath (Join-Path $sourceFFmpeg.Root 'LICENSE') -Destination (Join-Path $runtimeRoot 'LICENSE.txt')
    Copy-Item -LiteralPath (Join-Path $sourceFFmpeg.Root 'README.txt') -Destination (Join-Path $runtimeRoot 'README.txt')
    Copy-Item -LiteralPath (Join-Path $sourceFFmpeg.Root 'presets') -Destination $runtimeRoot -Recurse
    $runtimeMetadata = [ordered]@{
        schemaVersion = 1
        provider = 'Gyan.dev'
        package = $ffmpegPackage
        version = $ffmpegVersion
        versionLine = $sourceFFmpeg.VersionLine
        license = 'GPL-3.0-only'
        correspondingSource = 'https://github.com/FFmpeg/FFmpeg/commit/946fcce07b'
        archiveURL = 'https://github.com/GyanD/codexffmpeg/releases/download/9.0.2/ffmpeg-9.0.2-full_build.zip'
        archiveSHA256 = $sourceFFmpeg.ArchiveSHA256
        ffmpegSHA256 = File-SHA256 $sourceFFmpeg.FFmpeg
        ffprobeSHA256 = File-SHA256 $sourceFFmpeg.FFprobe
        packagedAt = (Get-Date).ToUniversalTime().ToString('o')
    }
    $runtimeMetadata | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $runtimeRoot 'BUNDLED_FFMPEG.json') -Encoding utf8NoBOM
    $bundleArchive = Join-Path $stage 'ffmpeg-runtime.zip'
    Compress-Archive -Path (Join-Path $runtimeRoot '*') -DestinationPath $bundleArchive -CompressionLevel Optimal

    $lightBuild = Join-Path $stage 'build-light/PheebKit.exe'
    $bundledBuild = Join-Path $stage 'build-bundled/PheebKit.exe'
    & (Join-Path $workspace 'scripts/build-windows.ps1') -OutputPath $lightBuild
    if ($LASTEXITCODE -ne 0) { throw 'Lightweight native build failed.' }
    & (Join-Path $workspace 'scripts/build-windows.ps1') -OutputPath $bundledBuild -BundleArchive $bundleArchive -BundleVersion $ffmpegVersion
    if ($LASTEXITCODE -ne 0) { throw 'Bundled native build failed.' }

    Invoke-Checked go @('run', 'golang.org/x/vuln/cmd/govulncheck@v1.9.0', '-mode=binary', $lightBuild)
    Invoke-Checked go @('run', 'golang.org/x/vuln/cmd/govulncheck@v1.9.0', '-mode=binary', $bundledBuild)
    foreach ($binary in @($lightBuild, $bundledBuild)) {
        if ((Get-Item -LiteralPath $binary).VersionInfo.ProductVersion -ne $version.version) { throw 'Executable version does not match VERSION.' }
    }
    $taskSourceRevision = (git rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0 -or $taskSourceRevision -notmatch '^[a-f0-9]{40}$') { throw 'Cannot record matching source revision.' }
    $actualLightBuild = Assert-BuildProvenance $lightBuild $taskSourceRevision $false ''
    $actualBundledBuild = Assert-BuildProvenance $bundledBuild $taskSourceRevision $true (File-SHA256 $bundleArchive)

    $lightOutput = Join-Path $output "PheebKit-$($version.version)-windows-x64.exe"
    $bundledOutput = Join-Path $output "PheebKit-$($version.version)-windows-x64-ffmpeg.exe"
    Copy-Item -LiteralPath $lightBuild -Destination $lightOutput
    Copy-Item -LiteralPath $bundledBuild -Destination $bundledOutput

    $portablePayload = Join-Path $stage 'portable'
    Copy-SharedPayloadFiles $portablePayload $true
    Copy-Item -LiteralPath $lightBuild -Destination (Join-Path $portablePayload 'PheebKit.exe')
    New-Item -ItemType Directory -Force -Path (Join-Path $portablePayload 'runtime') | Out-Null
    Copy-Item -LiteralPath $runtimeRoot -Destination (Join-Path $portablePayload 'runtime') -Recurse
    Write-BuildManifest $lightBuild (Join-Path $portablePayload 'BUILD_INFO.json') $taskSourceRevision $false $runtimeMetadata
    @"
PheebKit $($version.version) - Windows x64 portable with FFmpeg

Extract the entire ZIP, then run PheebKit.exe. FFmpeg $ffmpegVersion and ffprobe
are included under runtime\ffmpeg. No FFmpeg download is required unless you
explicitly replace them in Settings. Settings remain in AppData.
Windows 10/11 x64 and Microsoft WebView2 Runtime are required.

The bundled FFmpeg is a GPL v3 Gyan.dev build. Its license, README, exact source
commit, archive digest and executable hashes are in runtime\ffmpeg. Source:
https://github.com/u-fw/PheebKit/tree/v$($version.version)

Project license: GPL-3.0-only. See LICENSE.txt and THIRD_PARTY_NOTICES.txt.
"@ | Set-Content -LiteralPath (Join-Path $portablePayload 'README.txt') -Encoding utf8
    $portableZip = Join-Path $output "PheebKit-$($version.version)-windows-x64-portable-ffmpeg.zip"
    Compress-Archive -Path (Join-Path $portablePayload '*') -DestinationPath $portableZip -CompressionLevel Optimal

    $installerPayload = Join-Path $stage 'installer'
    Copy-SharedPayloadFiles $installerPayload $true
    Copy-Item -LiteralPath $bundledBuild -Destination (Join-Path $installerPayload 'PheebKit.exe')
    Write-BuildManifest $bundledBuild (Join-Path $installerPayload 'BUILD_INFO.json') $taskSourceRevision $true $runtimeMetadata
    @"
PheebKit $($version.version) - Windows x64 setup with FFmpeg

PheebKit and FFmpeg $ffmpegVersion / ffprobe are contained in this single
installer. The embedded runtime is verified and materialized into the current
user's private cache on first launch. No administrator rights are required.
Windows 10/11 x64 and Microsoft WebView2 Runtime are required.

The bundled FFmpeg is a GPL v3 Gyan.dev build. Exact version, source commit,
license and hashes are embedded in PheebKit. Source:
https://github.com/u-fw/PheebKit/tree/v$($version.version)

Project license: GPL-3.0-only. See LICENSE.txt and THIRD_PARTY_NOTICES.txt.
"@ | Set-Content -LiteralPath (Join-Path $installerPayload 'README.txt') -Encoding utf8
    $setupName = "PheebKit-$($version.version)-windows-x64-setup-ffmpeg"
    Invoke-Checked $InnoCompiler @(
        '--quiet',
        "--define=AppVersion=$($version.version)",
        "--define=WindowsVersion=$($version.windowsVersion)",
        "--define=PayloadDir=$installerPayload",
        "--define=ReleaseDir=$output",
        '--define=ApplicationExe=PheebKit.exe',
        "--define=OutputBaseFilename=$setupName",
        (Join-Path $workspace 'build/windows/installer.iss')
    )
    $setup = Join-Path $output "$setupName.exe"
    if (-not (Test-Path -LiteralPath $setup)) { throw 'Installer compiler did not produce the expected file.' }
    git diff --exit-code
    if ($LASTEXITCODE -ne 0) { throw 'Build changed tracked sources; commit generated files and rebuild.' }
    $checksums = foreach ($file in @($lightOutput, $bundledOutput, $portableZip, $setup)) {
        '{0}  {1}' -f (File-SHA256 $file), [IO.Path]::GetFileName($file)
    }
    $checksums | Set-Content -LiteralPath (Join-Path $output 'SHA256SUMS.txt') -Encoding ascii
    Move-Item -LiteralPath $output -Destination $final
    Write-Host "Release artifacts prepared locally: $final"
} finally {
    $resolvedStage = [IO.Path]::GetFullPath($stage)
    $expectedRoot = [IO.Path]::GetFullPath($releaseRoot) + [IO.Path]::DirectorySeparatorChar
    if (-not $resolvedStage.StartsWith($expectedRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Refusing cleanup outside release workspace.' }
    if (Test-Path -LiteralPath $resolvedStage) { Remove-Item -LiteralPath $resolvedStage -Recurse -Force }
}
