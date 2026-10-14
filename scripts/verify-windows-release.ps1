param([string]$PreviousReleaseDirectory)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'Installer verification requires native Windows.' }
$workspace = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$version = (Get-Content -LiteralPath (Join-Path $workspace 'VERSION') -Raw).Trim()
$release = Join-Path $workspace "release/$version"
$setup = Join-Path $release "PheebKit-$version-windows-x64-setup.exe"
$zip = Join-Path $release "PheebKit-$version-windows-x64-standalone.zip"
$registryPath = 'HKCU:/Software/Microsoft/Windows/CurrentVersion/Uninstall/{9D8FAD06-8029-4A58-9497-96C7DA878C63}_is1'
if (Test-Path -LiteralPath $registryPath) { throw 'An existing kompeg installation is registered; refusing to alter it for verification.' }
$root = Join-Path $workspace ('.tools/installer-test-' + [guid]::NewGuid().ToString('N'))
$installed = Join-Path $root 'app'
$uninstaller = Join-Path $installed 'unins000.exe'
$toolsRoot = [IO.Path]::GetFullPath((Join-Path $workspace '.tools')) + [IO.Path]::DirectorySeparatorChar
if (-not [IO.Path]::GetFullPath($root).StartsWith($toolsRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Test paths must stay inside the workspace tools directory' }
New-Item -ItemType Directory -Force -Path $root | Out-Null
$configHashes = @{}
foreach ($name in @('ffmpeg.json', 'history.json', 'preferences.json', 'editor.json')) {
    $path = Join-Path $env:APPDATA "kompeg/$name"
    $configHashes[$path] = if (Test-Path -LiteralPath $path) { (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash } else { '' }
}

function Run-Hidden([string]$Path, [string[]]$Arguments) {
    $process = Start-Process -FilePath $Path -ArgumentList $Arguments -WindowStyle Hidden -PassThru
    if (-not $process.WaitForExit(60000)) { throw "Verification process did not finish within 60 seconds: $Path (PID $($process.Id))" }
    return $process.ExitCode
}
function Confirm-ReleaseChecksums([string]$Directory, [string[]]$ExpectedNames) {
    $seen = @{}
    foreach ($line in Get-Content -LiteralPath (Join-Path $Directory 'SHA256SUMS.txt')) {
        if ($line -notmatch '^([a-f0-9]{64})  ([^/\\]+)$') { throw 'Invalid checksum file entry' }
        $digest, $name = $Matches[1], $Matches[2]
        if ($name -notin $ExpectedNames -or $seen.ContainsKey($name)) { throw 'Unexpected or duplicate release checksum entry' }
        if ((Get-FileHash -LiteralPath (Join-Path $Directory $name) -Algorithm SHA256).Hash.ToLowerInvariant() -ne $digest) { throw 'Release checksum mismatch' }
        $seen[$name] = $true
    }
    if ($seen.Count -ne $ExpectedNames.Count) { throw 'Release checksum manifest is incomplete' }
}
function Numeric-FileVersion($Info) { return [Version]('{0}.{1}.{2}.{3}' -f $Info.FileMajorPart, $Info.FileMinorPart, $Info.FileBuildPart, $Info.FilePrivatePart) }
function Remove-TestInstallation {
    if ((Test-Path -LiteralPath $uninstaller) -and (Test-Path -LiteralPath $registryPath)) {
        $registration = Get-ItemProperty -LiteralPath $registryPath -ErrorAction Stop
        if ([IO.Path]::GetFullPath($registration.InstallLocation.TrimEnd('\')) -ne [IO.Path]::GetFullPath($installed)) { throw 'Refusing to uninstall a registration outside the test target' }
        $code = Run-Hidden $uninstaller @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', ('/LOG="' + (Join-Path $root 'uninstall.log') + '"'))
        if ($code -ne 0) { throw "Test uninstall failed: $code" }
    }
}
try {
    Confirm-ReleaseChecksums $release @([IO.Path]::GetFileName($setup), [IO.Path]::GetFileName($zip))
    Expand-Archive -LiteralPath $zip -DestinationPath (Join-Path $root 'standalone')
    $standalone = Join-Path $root 'standalone/PheebKit.exe'
    if ((Get-Item -LiteralPath $standalone).VersionInfo.ProductVersion -ne $version) { throw 'Standalone version mismatch' }
    $branding = (Get-Item -LiteralPath $standalone).VersionInfo
    if ($branding.ProductName -ne '菲比工具箱 (PheebKit)' -or $branding.OriginalFilename -ne 'PheebKit.exe') { throw 'Executable branding mismatch' }
    $arguments = @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/SP-', '/NORESTART', '/NOCLOSEAPPLICATIONS', '/NOICONS', '/TASKS=""', '/LANG=english', ('/DIR="' + $installed + '"'))
    if ($PreviousReleaseDirectory) {
        $previous = [IO.Path]::GetFullPath($PreviousReleaseDirectory)
        $releasePrefix = [IO.Path]::GetFullPath((Join-Path $workspace 'release')) + [IO.Path]::DirectorySeparatorChar
        if (-not ($previous.StartsWith($releasePrefix, [StringComparison]::OrdinalIgnoreCase) -or $previous.StartsWith($toolsRoot, [StringComparison]::OrdinalIgnoreCase))) { throw 'Previous candidate must be inside this workspace release or tools directory' }
        $priorZips = @(Get-ChildItem -LiteralPath $previous -Filter 'PheebKit-*-windows-x64-standalone.zip' -File)
        if ($priorZips.Count -ne 1 -or $priorZips[0].Name -notmatch '^PheebKit-(\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?)-windows-x64-standalone\.zip$') { throw 'Previous candidate needs exactly one versioned standalone ZIP' }
        $priorVersion = $Matches[1]
        $priorSetupName = "PheebKit-$priorVersion-windows-x64-setup.exe"
        Confirm-ReleaseChecksums $previous @($priorSetupName, $priorZips[0].Name)
        Expand-Archive -LiteralPath $priorZips[0].FullName -DestinationPath (Join-Path $root 'previous-payload')
        $priorBinary = Join-Path $root 'previous-payload/PheebKit.exe'
        $priorInfo = (Get-Item -LiteralPath $priorBinary).VersionInfo
        if ($priorInfo.ProductVersion -ne $priorVersion -or (Numeric-FileVersion $priorInfo) -ge (Numeric-FileVersion $branding)) { throw 'Upgrade baseline must be an older matching PheebKit binary' }
        $prior = Run-Hidden (Join-Path $previous $priorSetupName) ($arguments + @('/LOG="' + (Join-Path $root 'previous-install.log') + '"'))
        if ($prior -ne 0) { throw "Previous installation failed: $prior" }
        if ((Get-ItemProperty -LiteralPath $registryPath).DisplayVersion -ne $priorVersion) { throw 'Previous version registration mismatch' }
        if ((Get-FileHash -LiteralPath $priorBinary).Hash -ne (Get-FileHash -LiteralPath (Join-Path $installed 'PheebKit.exe')).Hash) { throw 'Previous installation payload mismatch' }
        'retain across actual version upgrade' | Set-Content -LiteralPath (Join-Path $installed 'keep-upgrade-file.txt')
    }
    $beforeBlocked = if (Test-Path -LiteralPath (Join-Path $installed 'PheebKit.exe')) { (Get-FileHash -LiteralPath (Join-Path $installed 'PheebKit.exe')).Hash } else { '' }
    $mutex = [Threading.Mutex]::new($false, 'Local\kompeg-application-running')
    try {
        $blocked = Run-Hidden $setup ($arguments + @('/LOG="' + (Join-Path $root 'blocked-install.log') + '"'))
        $afterBlocked = if (Test-Path -LiteralPath (Join-Path $installed 'PheebKit.exe')) { (Get-FileHash -LiteralPath (Join-Path $installed 'PheebKit.exe')).Hash } else { '' }
        if ($blocked -eq 0 -or $beforeBlocked -ne $afterBlocked) { throw 'Installer failed to respect running-application guard' }
    } finally { $mutex.Dispose() }
    $first = Run-Hidden $setup ($arguments + @('/LOG="' + (Join-Path $root 'install.log') + '"'))
    if ($first -ne 0) { throw "Installation failed: $first" }
    $registration = Get-ItemProperty -LiteralPath $registryPath
    if ($registration.DisplayVersion -ne $version) { throw 'Registered version mismatch' }
    if ($registration.DisplayName -notlike '菲比工具箱 (PheebKit)*') { throw 'Installer branding mismatch' }
    if ($PreviousReleaseDirectory -and -not (Test-Path -LiteralPath (Join-Path $installed 'keep-upgrade-file.txt'))) { throw 'Version upgrade removed user-owned files' }
    if ((Get-FileHash -LiteralPath $standalone).Hash -ne (Get-FileHash -LiteralPath (Join-Path $installed 'PheebKit.exe')).Hash) { throw 'Installer and ZIP contain different application binaries' }
    foreach ($file in @('LICENSE.txt', 'THIRD_PARTY_NOTICES.txt', 'README.txt', 'RELEASE_NOTES.md')) {
        if ((Get-FileHash -LiteralPath (Join-Path $root "standalone/$file")).Hash -ne (Get-FileHash -LiteralPath (Join-Path $installed $file)).Hash) { throw "Payload notice differs: $file" }
    }
    'user-owned data; retain during reinstall and uninstall' | Set-Content -LiteralPath (Join-Path $installed 'keep-user-file.txt')
    $second = Run-Hidden $setup ($arguments + @('/LOG="' + (Join-Path $root 'reinstall.log') + '"'))
    if ($second -ne 0 -or -not (Test-Path -LiteralPath (Join-Path $installed 'keep-user-file.txt'))) { throw 'Reinstall failed or removed user-owned files' }
    $mutex = [Threading.Mutex]::new($false, 'Local\kompeg-application-running')
    try {
        $blocked = Run-Hidden $uninstaller @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART')
        if ($blocked -eq 0 -or -not (Test-Path -LiteralPath (Join-Path $installed 'PheebKit.exe'))) { throw 'Uninstaller failed to respect running-application guard' }
    } finally { $mutex.Dispose() }
    Remove-TestInstallation
    if ((Test-Path -LiteralPath $registryPath) -or (Test-Path -LiteralPath (Join-Path $installed 'PheebKit.exe'))) { throw 'Uninstall left application or uninstall registration behind' }
    if (-not (Test-Path -LiteralPath (Join-Path $installed 'keep-user-file.txt'))) { throw 'Uninstall removed user-owned files' }
    if ($PreviousReleaseDirectory -and -not (Test-Path -LiteralPath (Join-Path $installed 'keep-upgrade-file.txt'))) { throw 'Uninstall removed upgrade sentinel' }
    foreach ($path in $configHashes.Keys) {
        $afterHash = if (Test-Path -LiteralPath $path) { (Get-FileHash -LiteralPath $path).Hash } else { '' }
        if ($configHashes[$path] -ne $afterHash) { throw 'Installation modified existing application configuration or history' }
    }
    try {
        & (Join-Path $PSScriptRoot 'release-windows.ps1')
        throw 'Existing release was unexpectedly overwritten'
    } catch {
        if ($_.Exception.Message -notmatch 'Release directory exists and will not be overwritten') { throw }
    }
    Write-Host "PASS: checksums, shared payload, versions, install/reinstall/uninstall, running guards and configuration preservation. Upgrade baseline: $(if ($PreviousReleaseDirectory) { $priorVersion } else { 'not requested' }). Logs: $root"
} finally {
    # Keep test logs for review. Uninstall only this script's verified target.
    Remove-TestInstallation
}
