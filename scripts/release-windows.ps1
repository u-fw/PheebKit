param([string]$InnoCompiler = $env:KOMPEG_ISCC)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'Windows x64 release packaging requires native Windows PowerShell 7.' }
if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -ne 'X64') { throw 'Only Windows x64 releases are currently supported.' }
$workspace = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
Set-Location -LiteralPath $workspace

function Invoke-Checked([string]$Program, [string[]]$Arguments) {
    & $Program @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Program failed with exit code $LASTEXITCODE" }
}

$versionInfo = node --input-type=module -e 'import { readVersion } from "./scripts/version.mjs"; console.log(JSON.stringify(readVersion()));'
if ($LASTEXITCODE -ne 0) { throw 'Invalid VERSION file' }
$version = $versionInfo | ConvertFrom-Json
$releaseRoot = Join-Path $workspace 'release'
$final = Join-Path $releaseRoot $version.version
$resolvedFinal = [IO.Path]::GetFullPath($final)
if (-not $resolvedFinal.StartsWith([IO.Path]::GetFullPath($releaseRoot) + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Release output must stay in the workspace release directory' }
if (Test-Path -LiteralPath $final) { throw "Release directory exists and will not be overwritten: $final" }
if (-not $InnoCompiler) {
    foreach ($candidate in @((Join-Path $workspace '.tools/inno-7.1.0/ISCC.exe'), 'C:/Program Files/Inno Setup 7/ISCC.exe')) {
        if (Test-Path -LiteralPath $candidate) { $InnoCompiler = $candidate; break }
    }
}
if (-not $InnoCompiler -or -not (Test-Path -LiteralPath $InnoCompiler)) { throw 'Install verified Inno Setup 7.1.0 or supply -InnoCompiler / KOMPEG_ISCC.' }
$InnoCompiler = (Resolve-Path -LiteralPath $InnoCompiler).Path
New-Item -ItemType Directory -Force -Path $releaseRoot | Out-Null
$stage = Join-Path $releaseRoot ('.stage-' + [guid]::NewGuid().ToString('N'))
$payload = Join-Path $stage 'payload'
$output = Join-Path $stage 'output'
New-Item -ItemType Directory -Path $stage, $payload, $output | Out-Null
try {
    Invoke-Checked node @('scripts/sync-version.mjs')
    $dirty = git status --porcelain --untracked-files=normal
    if ($LASTEXITCODE -ne 0 -or $dirty) { throw 'Release packaging requires committed sources and synchronized version metadata; commit changes before packaging.' }
    Invoke-Checked node @('--test', 'scripts/version.test.mjs')
    Invoke-Checked go @('test', './...')
    Invoke-Checked go @('vet', './...')
    foreach ($check in @('test', 'typecheck', 'lint', 'format:check')) { Invoke-Checked npm @('--prefix', 'frontend', 'run', $check) }
    Invoke-Checked wails3 @('build')
    $application = Join-Path $workspace 'bin/PheebKit.exe'
    $metadata = (Get-Item -LiteralPath $application).VersionInfo
    if ($metadata.ProductVersion -ne $version.version) { throw 'Executable version does not match VERSION' }
    Copy-Item -LiteralPath $application -Destination (Join-Path $payload 'PheebKit.exe')
    Copy-Item -LiteralPath (Join-Path $workspace 'LICENSE') -Destination (Join-Path $payload 'LICENSE.txt')
    Invoke-Checked node @('scripts/dependency-notices.mjs', (Join-Path $payload 'THIRD_PARTY_NOTICES.txt'))
    @"
PheebKit $($version.version) - Windows x64 experimental release

Extract the entire ZIP, then run PheebKit.exe. This is a no-install application;
settings and downloaded FFmpeg are stored in AppData, not beside this executable.
Windows 10/11 x64 and Microsoft WebView2 Runtime are required.
WebView2: https://developer.microsoft.com/en-us/microsoft-edge/webview2/#download-section
FFmpeg is external. Existing usable tools are reused; missing tools can be
downloaded explicitly or selected inside Settings. System PATH is not modified.
Android is deferred and no APK is included. Windows 10 and clean-machine checks
are separate release acceptance steps; a successful build is not evidence of them.

Project license: GPL-3.0-only. See LICENSE.txt and THIRD_PARTY_NOTICES.txt.
Corresponding source: the repository's matching v$($version.version) tag.
"@ | Set-Content -LiteralPath (Join-Path $payload 'README.txt') -Encoding utf8
    $zip = Join-Path $output "PheebKit-$($version.version)-windows-x64-standalone.zip"
    Compress-Archive -Path (Join-Path $payload '*') -DestinationPath $zip -CompressionLevel Optimal
    Invoke-Checked $InnoCompiler @('--quiet', "--define=AppVersion=$($version.version)", "--define=WindowsVersion=$($version.windowsVersion)", "--define=PayloadDir=$payload", "--define=ReleaseDir=$output", (Join-Path $workspace 'build/windows/installer.iss'))
    $setup = Join-Path $output "PheebKit-$($version.version)-windows-x64-setup.exe"
    if (-not (Test-Path -LiteralPath $setup)) { throw 'Installer compiler did not produce the expected file' }
    git diff --exit-code
    if ($LASTEXITCODE -ne 0) { throw 'Build changed tracked sources; commit generated files and rebuild.' }
    $checksums = foreach ($file in @($setup, $zip)) {
        '{0}  {1}' -f (Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant(), [IO.Path]::GetFileName($file)
    }
    $checksums | Set-Content -LiteralPath (Join-Path $output 'SHA256SUMS.txt') -Encoding ascii
    Move-Item -LiteralPath $output -Destination $final
    Write-Host "Release artifacts prepared locally: $final"
} finally {
    $resolvedStage = [IO.Path]::GetFullPath($stage)
    $expectedRoot = [IO.Path]::GetFullPath($releaseRoot) + [IO.Path]::DirectorySeparatorChar
    if (-not $resolvedStage.StartsWith($expectedRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Refusing cleanup outside release workspace' }
    if (Test-Path -LiteralPath $resolvedStage) { Remove-Item -LiteralPath $resolvedStage -Recurse -Force }
}
