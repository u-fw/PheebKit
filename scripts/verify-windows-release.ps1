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
function Remove-TestInstallation {
    if ((Test-Path -LiteralPath $uninstaller) -and (Test-Path -LiteralPath $registryPath)) {
        $registration = Get-ItemProperty -LiteralPath $registryPath -ErrorAction Stop
        if ([IO.Path]::GetFullPath($registration.InstallLocation.TrimEnd('\')) -ne [IO.Path]::GetFullPath($installed)) { throw 'Refusing to uninstall a registration outside the test target' }
        $code = Run-Hidden $uninstaller @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', ('/LOG="' + (Join-Path $root 'uninstall.log') + '"'))
        if ($code -ne 0) { throw "Test uninstall failed: $code" }
    }
}
try {
    foreach ($line in Get-Content -LiteralPath (Join-Path $release 'SHA256SUMS.txt')) {
        if ($line -notmatch '^([a-f0-9]{64})  ([^/\\]+)$') { throw 'Invalid checksum file entry' }
        if ((Get-FileHash -LiteralPath (Join-Path $release $Matches[2]) -Algorithm SHA256).Hash.ToLowerInvariant() -ne $Matches[1]) { throw 'Release checksum mismatch' }
    }
    Expand-Archive -LiteralPath $zip -DestinationPath (Join-Path $root 'standalone')
    $standalone = Join-Path $root 'standalone/PheebKit.exe'
    if ((Get-Item -LiteralPath $standalone).VersionInfo.ProductVersion -ne $version) { throw 'Standalone version mismatch' }
    $branding = (Get-Item -LiteralPath $standalone).VersionInfo
    if ($branding.ProductName -ne '菲比工具箱 (PheebKit)' -or $branding.OriginalFilename -ne 'PheebKit.exe') { throw 'Executable branding mismatch' }
    $arguments = @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/SP-', '/NORESTART', '/NOCLOSEAPPLICATIONS', '/NOICONS', '/TASKS=""', '/LANG=english', ('/DIR="' + $installed + '"'))
    $mutex = [Threading.Mutex]::new($false, 'Local\kompeg-application-running')
    try {
        $blocked = Run-Hidden $setup ($arguments + @('/LOG="' + (Join-Path $root 'blocked-install.log') + '"'))
        if ($blocked -eq 0 -or (Test-Path -LiteralPath (Join-Path $installed 'PheebKit.exe'))) { throw 'Installer failed to respect running-application guard' }
    } finally { $mutex.Dispose() }
    $first = Run-Hidden $setup ($arguments + @('/LOG="' + (Join-Path $root 'install.log') + '"'))
    if ($first -ne 0) { throw "Installation failed: $first" }
    $registration = Get-ItemProperty -LiteralPath $registryPath
    if ($registration.DisplayVersion -ne $version) { throw 'Registered version mismatch' }
    if ($registration.DisplayName -notlike '菲比工具箱 (PheebKit)*') { throw 'Installer branding mismatch' }
    if ((Get-FileHash -LiteralPath $standalone).Hash -ne (Get-FileHash -LiteralPath (Join-Path $installed 'PheebKit.exe')).Hash) { throw 'Installer and ZIP contain different application binaries' }
    foreach ($file in @('LICENSE.txt', 'THIRD_PARTY_NOTICES.txt', 'README.txt')) {
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
    Write-Host "PASS: checksums, shared payload, versions, install/reinstall/uninstall, running guards and configuration preservation. Logs: $root"
} finally {
    # Keep test logs for review. Uninstall only this script's verified target.
    Remove-TestInstallation
}
