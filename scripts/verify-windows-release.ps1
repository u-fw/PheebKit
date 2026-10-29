param([string]$PreviousReleaseDirectory)

Set-StrictMode -Version Latest
. (Join-Path $PSScriptRoot 'legacy.ps1')
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'Release verification requires native Windows.' }
$workspace = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$version = (Get-Content -LiteralPath (Join-Path $workspace 'VERSION') -Raw).Trim()
$release = Join-Path $workspace "release/$version"
$lightName = "PheebKit-$version-windows-x64.exe"
$bundledName = "PheebKit-$version-windows-x64-ffmpeg.exe"
$portableName = "PheebKit-$version-windows-x64-portable-ffmpeg.zip"
$setupName = "PheebKit-$version-windows-x64-setup-ffmpeg.exe"
$light = Join-Path $release $lightName
$bundled = Join-Path $release $bundledName
$portable = Join-Path $release $portableName
$setup = Join-Path $release $setupName
$registryPath = 'HKCU:/Software/Microsoft/Windows/CurrentVersion/Uninstall/{9D8FAD06-8029-4A58-9497-96C7DA878C63}_is1'
if (Test-Path -LiteralPath $registryPath) { throw 'An existing PheebKit installation is registered; refusing to alter it for verification.' }
$root = Join-Path $workspace ('.tools/installer-test-' + [guid]::NewGuid().ToString('N'))
$installed = Join-Path $root 'app'
$uninstaller = Join-Path $installed 'unins000.exe'
$toolsRoot = [IO.Path]::GetFullPath((Join-Path $workspace '.tools')) + [IO.Path]::DirectorySeparatorChar
if (-not [IO.Path]::GetFullPath($root).StartsWith($toolsRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Test paths must stay inside the workspace tools directory.' }
New-Item -ItemType Directory -Force -Path $root | Out-Null
$configHashes = @{}
foreach ($directory in @('PheebKit', $PheebKitLegacyConfigDirectory)) {
    foreach ($name in @('ffmpeg.json', 'history.json', 'preferences.json', 'editor.json')) {
        $path = Join-Path $env:APPDATA "$directory/$name"
        $configHashes[$path] = if (Test-Path -LiteralPath $path) { (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash } else { '' }
    }
}

function Run-Hidden([string]$Path, [string[]]$Arguments) {
    $process = Start-Process -FilePath $Path -ArgumentList $Arguments -WindowStyle Hidden -PassThru
    if (-not $process.WaitForExit(120000)) { throw "Verification process did not finish within 120 seconds: $Path (PID $($process.Id))" }
    return $process.ExitCode
}

function Confirm-ReleaseChecksums([string]$Directory, [string[]]$ExpectedNames) {
    $seen = @{}
    foreach ($line in Get-Content -LiteralPath (Join-Path $Directory 'SHA256SUMS.txt')) {
        if ($line -notmatch '^([a-f0-9]{64})  ([^/\\]+)$') { throw 'Invalid checksum file entry.' }
        $digest, $name = $Matches[1], $Matches[2]
        if ($name -notin $ExpectedNames -or $seen.ContainsKey($name)) { throw 'Unexpected or duplicate release checksum entry.' }
        if ((Get-FileHash -LiteralPath (Join-Path $Directory $name) -Algorithm SHA256).Hash.ToLowerInvariant() -ne $digest) { throw 'Release checksum mismatch.' }
        $seen[$name] = $true
    }
    if ($seen.Count -ne $ExpectedNames.Count) { throw 'Release checksum manifest is incomplete.' }
}

function Numeric-FileVersion($Info) {
    return [Version]('{0}.{1}.{2}.{3}' -f $Info.FileMajorPart, $Info.FileMinorPart, $Info.FileBuildPart, $Info.FilePrivatePart)
}

function Confirm-AppMetadata([string]$Path, [bool]$BundledFFmpeg) {
    $info = (Get-Item -LiteralPath $Path).VersionInfo
    if ($info.ProductVersion -ne $version -or $info.ProductName -ne '菲比工具箱 (PheebKit)' -or $info.OriginalFilename -ne 'PheebKit.exe') {
        throw 'Executable branding or version mismatch.'
    }
    $build = & $Path --build-info | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0 -or $build.schemaVersion -ne 1 -or $build.version -ne $version -or $build.sourceRevision -notmatch '^[a-f0-9]{40}$' -or $build.modified -ne $false -or $build.framework -ne 'wails-v2.17.0') {
        throw 'Executable provenance mismatch.'
    }
    $bundledProperty = $build.PSObject.Properties['bundledFFmpeg']
    if ($BundledFFmpeg) {
        if (-not $bundledProperty -or $bundledProperty.Value.kind -ne 'ffmpeg' -or $bundledProperty.Value.payloadSHA256 -notmatch '^[a-f0-9]{64}$' -or $bundledProperty.Value.distributionVersion -ne '9.0.2' -or $bundledProperty.Value.extractedAtFirstLaunch -ne $true) {
            throw 'Bundled executable payload metadata mismatch.'
        }
    } elseif ($bundledProperty) {
        throw 'Lightweight executable unexpectedly contains FFmpeg.'
    }
    return $build
}

function Set-Or-RemoveEnvironment([string]$Name, [AllowNull()][string]$Value) {
    [Environment]::SetEnvironmentVariable($Name, $Value, [EnvironmentVariableTarget]::Process)
}

function Invoke-IsolatedEnvironment([string]$Executable, [string]$IsolationRoot) {
    $appData = Join-Path $IsolationRoot 'appdata'
    $localAppData = Join-Path $IsolationRoot 'localappdata'
    New-Item -ItemType Directory -Force -Path $appData, $localAppData | Out-Null
    $names = @('PHEEBKIT_FFMPEG', 'PHEEBKIT_FFPROBE', 'FFMPEG_PATH', 'FFPROBE_PATH', 'APPDATA', 'LOCALAPPDATA', 'PATH')
    $saved = @{}
    foreach ($name in $names) { $saved[$name] = [Environment]::GetEnvironmentVariable($name, [EnvironmentVariableTarget]::Process) }
    try {
        foreach ($name in @('PHEEBKIT_FFMPEG', 'PHEEBKIT_FFPROBE', 'FFMPEG_PATH', 'FFPROBE_PATH')) { Set-Or-RemoveEnvironment $name $null }
        Set-Or-RemoveEnvironment 'APPDATA' $appData
        Set-Or-RemoveEnvironment 'LOCALAPPDATA' $localAppData
        Set-Or-RemoveEnvironment 'PATH' ''
        $stdout = Join-Path $IsolationRoot 'environment.json'
        $stderr = Join-Path $IsolationRoot 'environment.stderr.txt'
        $process = Start-Process -FilePath $Executable -ArgumentList '--environment-info' -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru
        if (-not $process.WaitForExit(60000)) { throw "Environment inspection did not finish: $Executable" }
        if ($process.ExitCode -ne 0) {
            $detail = if (Test-Path -LiteralPath $stderr) { Get-Content -LiteralPath $stderr -Raw } else { '' }
            throw "Bundled environment inspection failed with exit code $($process.ExitCode): $detail"
        }
        return (Get-Content -LiteralPath $stdout -Raw | ConvertFrom-Json)
    } finally {
        foreach ($name in $names) {
            if ($null -eq $saved[$name]) { Set-Or-RemoveEnvironment $name $null } else { Set-Or-RemoveEnvironment $name ([string]$saved[$name]) }
        }
    }
}

function Confirm-BundledEnvironment([string]$Executable, [string]$IsolationRoot, [string]$ExpectedDirectoryPrefix) {
    $environment = Invoke-IsolatedEnvironment $Executable $IsolationRoot
    if ($environment.ready -ne $true -or $environment.ffmpeg.source -ne 'bundled' -or $environment.ffprobe.source -ne 'bundled' -or $environment.ffmpeg.version -notmatch '^ffmpeg version 9\.0\.2-' -or $environment.ffprobe.version -notmatch '^ffprobe version 9\.0\.2-') {
        throw "Bundled FFmpeg was not selected or validated: $($environment | ConvertTo-Json -Compress)"
    }
    if ($ExpectedDirectoryPrefix -and -not ([IO.Path]::GetFullPath($environment.ffmpeg.path).StartsWith([IO.Path]::GetFullPath($ExpectedDirectoryPrefix), [StringComparison]::OrdinalIgnoreCase))) {
        throw 'Portable executable did not use its adjacent FFmpeg runtime.'
    }
    return $environment
}

function Confirm-RuntimeMetadata([string]$Directory) {
    $metadata = Get-Content -LiteralPath (Join-Path $Directory 'BUNDLED_FFMPEG.json') -Raw | ConvertFrom-Json
    if ($metadata.schemaVersion -ne 1 -or $metadata.package -ne 'gyan-9.0.2' -or $metadata.version -ne '9.0.2' -or $metadata.license -ne 'GPL-3.0-only' -or $metadata.correspondingSource -notmatch '^https://github\.com/FFmpeg/FFmpeg/commit/[a-f0-9]{10}$') {
        throw 'Bundled FFmpeg provenance metadata mismatch.'
    }
    foreach ($item in @(
        @{ Name = 'ffmpeg'; Expected = $metadata.ffmpegSHA256 },
        @{ Name = 'ffprobe'; Expected = $metadata.ffprobeSHA256 }
    )) {
        $path = Join-Path $Directory "bin/$($item.Name).exe"
        if ((Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant() -ne $item.Expected) { throw "Bundled $($item.Name) digest mismatch." }
    }
    foreach ($notice in @('LICENSE.txt', 'README.txt', 'presets')) {
        if (-not (Test-Path -LiteralPath (Join-Path $Directory $notice))) { throw "Bundled FFmpeg is missing $notice." }
    }
    return $metadata
}

function Remove-TestInstallation {
    if ((Test-Path -LiteralPath $uninstaller) -and (Test-Path -LiteralPath $registryPath)) {
        $registration = Get-ItemProperty -LiteralPath $registryPath -ErrorAction Stop
        if ([IO.Path]::GetFullPath($registration.InstallLocation.TrimEnd('\')) -ne [IO.Path]::GetFullPath($installed)) { throw 'Refusing to uninstall a registration outside the test target.' }
        $code = Run-Hidden $uninstaller @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', ('/LOG="' + (Join-Path $root 'uninstall.log') + '"'))
        if ($code -ne 0) { throw "Test uninstall failed: $code" }
    }
}

try {
    Confirm-ReleaseChecksums $release @($lightName, $bundledName, $portableName, $setupName)
    $releaseFiles = @(Get-ChildItem -LiteralPath $release -File | ForEach-Object Name | Sort-Object)
    $expectedFiles = @(@($bundledName, $lightName, $portableName, $setupName, 'SHA256SUMS.txt') | Sort-Object)
    if (($releaseFiles -join "`n") -ne ($expectedFiles -join "`n")) { throw 'Release directory has unexpected files.' }

    $lightBuild = Confirm-AppMetadata $light $false
    $bundledBuild = Confirm-AppMetadata $bundled $true
    if ($bundledBuild.bundledFFmpeg.payloadSHA256 -notmatch '^[a-f0-9]{64}$') { throw 'Bundled payload digest missing.' }

    $portableRoot = Join-Path $root 'portable'
    Expand-Archive -LiteralPath $portable -DestinationPath $portableRoot
    $portableApp = Join-Path $portableRoot 'PheebKit.exe'
    $portableBuild = Confirm-AppMetadata $portableApp $false
    $portableMetadata = Confirm-RuntimeMetadata (Join-Path $portableRoot 'runtime/ffmpeg')
    $portableEnvironment = Confirm-BundledEnvironment $portableApp (Join-Path $root 'portable-isolation') (Join-Path $portableRoot 'runtime/ffmpeg/bin')
    $portableManifest = Get-Content -LiteralPath (Join-Path $portableRoot 'BUILD_INFO.json') -Raw | ConvertFrom-Json
    if ($portableManifest.version -ne $version -or $portableManifest.executableSHA256 -ne (Get-FileHash -LiteralPath $portableApp -Algorithm SHA256).Hash.ToLowerInvariant() -or $portableManifest.PSObject.Properties['bundledFFmpeg']) {
        throw 'Portable BUILD_INFO.json does not describe the lightweight sidecar payload.'
    }

    $embeddedEnvironment = Confirm-BundledEnvironment $bundled (Join-Path $root 'embedded-isolation') $null
    $runtimeDirectories = @(Get-ChildItem -LiteralPath (Join-Path $root 'embedded-isolation/localappdata/PheebKit/runtime/ffmpeg') -Directory -Filter 'ffmpeg-9.0.2-*')
    if ($runtimeDirectories.Count -ne 1) { throw 'Embedded FFmpeg did not materialize exactly one versioned runtime.' }
    $embeddedMetadata = Confirm-RuntimeMetadata $runtimeDirectories[0].FullName
    $reusedEnvironment = Confirm-BundledEnvironment $bundled (Join-Path $root 'embedded-isolation') $null
    if ($reusedEnvironment.ffmpeg.path -ne $embeddedEnvironment.ffmpeg.path) { throw 'Embedded FFmpeg was not reused on second launch.' }

    $arguments = @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/SP-', '/NORESTART', '/NOCLOSEAPPLICATIONS', '/NOICONS', '/TASKS=""', '/LANG=english', ('/DIR="' + $installed + '"'))
    if ($PreviousReleaseDirectory) {
        $previous = [IO.Path]::GetFullPath($PreviousReleaseDirectory)
        $releasePrefix = [IO.Path]::GetFullPath((Join-Path $workspace 'release')) + [IO.Path]::DirectorySeparatorChar
        if (-not ($previous.StartsWith($releasePrefix, [StringComparison]::OrdinalIgnoreCase) -or $previous.StartsWith($toolsRoot, [StringComparison]::OrdinalIgnoreCase))) { throw 'Previous candidate must be inside this workspace release or tools directory.' }
        $priorSetups = @(Get-ChildItem -LiteralPath $previous -File | Where-Object Name -Match '^PheebKit-(.+)-windows-x64-(?:setup|setup-ffmpeg)\.exe$')
        $priorZips = @(Get-ChildItem -LiteralPath $previous -File | Where-Object Name -Match '^PheebKit-(.+)-windows-x64-(?:standalone|portable-ffmpeg)\.zip$')
        if ($priorSetups.Count -ne 1 -or $priorZips.Count -ne 1) { throw 'Previous candidate needs exactly one setup and one portable ZIP.' }
        if ($priorSetups[0].Name -notmatch '^PheebKit-(.+)-windows-x64-(?:setup|setup-ffmpeg)\.exe$') { throw 'Cannot parse prior setup version.' }
        $priorVersion = $Matches[1]
        if ($priorZips[0].Name -notmatch "^PheebKit-$([regex]::Escape($priorVersion))-windows-x64-(?:standalone|portable-ffmpeg)\.zip$") { throw 'Previous artifacts use different versions.' }
        $priorNames = @($priorSetups[0].Name, $priorZips[0].Name)
        $priorLight = "PheebKit-$priorVersion-windows-x64.exe"
        $priorBundled = "PheebKit-$priorVersion-windows-x64-ffmpeg.exe"
        $hasLight = Test-Path -LiteralPath (Join-Path $previous $priorLight) -PathType Leaf
        $hasBundled = Test-Path -LiteralPath (Join-Path $previous $priorBundled) -PathType Leaf
        if ($hasLight -ne $hasBundled) { throw 'Previous direct executable pair is incomplete.' }
        if ($hasLight) { $priorNames += @($priorLight, $priorBundled) }
        Confirm-ReleaseChecksums $previous $priorNames
        Expand-Archive -LiteralPath $priorZips[0].FullName -DestinationPath (Join-Path $root 'previous-payload')
        $priorBinary = Join-Path $root 'previous-payload/PheebKit.exe'
        $priorInfo = (Get-Item -LiteralPath $priorBinary).VersionInfo
        if ($priorInfo.ProductVersion -ne $priorVersion -or (Numeric-FileVersion $priorInfo) -ge (Numeric-FileVersion (Get-Item -LiteralPath $bundled).VersionInfo)) { throw 'Upgrade baseline must be an older matching PheebKit binary.' }
        $prior = Run-Hidden $priorSetups[0].FullName ($arguments + @('/LOG="' + (Join-Path $root 'previous-install.log') + '"'))
        if ($prior -ne 0) { throw "Previous installation failed: $prior" }
        'retain across actual version upgrade' | Set-Content -LiteralPath (Join-Path $installed 'keep-upgrade-file.txt')
    }
    $guardIndex = 0
    $beforeBlocked = if (Test-Path -LiteralPath (Join-Path $installed 'PheebKit.exe')) { (Get-FileHash -LiteralPath (Join-Path $installed 'PheebKit.exe')).Hash } else { '' }
    foreach ($runningMutex in @('Local\PheebKit-application-running', $PheebKitLegacyRunningMutex)) {
        $mutex = [Threading.Mutex]::new($false, $runningMutex)
        try {
            $blocked = Run-Hidden $setup ($arguments + @('/LOG="' + (Join-Path $root ('blocked-install-' + $guardIndex + '.log')) + '"'))
            $afterBlocked = if (Test-Path -LiteralPath (Join-Path $installed 'PheebKit.exe')) { (Get-FileHash -LiteralPath (Join-Path $installed 'PheebKit.exe')).Hash } else { '' }
            if ($blocked -eq 0 -or $beforeBlocked -ne $afterBlocked) { throw 'Installer failed to respect running-application guard.' }
        } finally {
            $mutex.Dispose()
        }
        $guardIndex++
    }
    $first = Run-Hidden $setup ($arguments + @('/LOG="' + (Join-Path $root 'install.log') + '"'))
    if ($first -ne 0) { throw "Installation failed: $first" }
    $registration = Get-ItemProperty -LiteralPath $registryPath
    if ($registration.DisplayVersion -ne $version -or $registration.DisplayName -notlike '菲比工具箱 (PheebKit)*') { throw 'Installer registration mismatch.' }
    if ($PreviousReleaseDirectory -and -not (Test-Path -LiteralPath (Join-Path $installed 'keep-upgrade-file.txt'))) { throw 'Version upgrade removed user-owned files.' }
    if ((Get-FileHash -LiteralPath $bundled).Hash -ne (Get-FileHash -LiteralPath (Join-Path $installed 'PheebKit.exe')).Hash) { throw 'Installer and bundled EXE differ.' }
    $installedBuild = Get-Content -LiteralPath (Join-Path $installed 'BUILD_INFO.json') -Raw | ConvertFrom-Json
    if ($installedBuild.version -ne $version -or $installedBuild.distribution -ne 'windows-x64-ffmpeg' -or $installedBuild.bundledFFmpeg.payloadSHA256 -ne $bundledBuild.bundledFFmpeg.payloadSHA256) { throw 'Installer BUILD_INFO.json mismatch.' }
    $installedEnvironment = Confirm-BundledEnvironment (Join-Path $installed 'PheebKit.exe') (Join-Path $root 'installed-isolation') $null
    if ($installedEnvironment.ffmpeg.version -ne $embeddedEnvironment.ffmpeg.version) { throw 'Installed and direct bundled tools report different versions.' }
    'user-owned data; retain during reinstall and uninstall' | Set-Content -LiteralPath (Join-Path $installed 'keep-user-file.txt')
    $second = Run-Hidden $setup ($arguments + @('/LOG="' + (Join-Path $root 'reinstall.log') + '"'))
    if ($second -ne 0 -or -not (Test-Path -LiteralPath (Join-Path $installed 'keep-user-file.txt'))) { throw 'Reinstall failed or removed user-owned files.' }
    foreach ($runningMutex in @('Local\PheebKit-application-running', $PheebKitLegacyRunningMutex)) {
        $mutex = [Threading.Mutex]::new($false, $runningMutex)
        try {
            $blocked = Run-Hidden $uninstaller @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART')
            if ($blocked -eq 0 -or -not (Test-Path -LiteralPath (Join-Path $installed 'PheebKit.exe'))) { throw 'Uninstaller failed to respect running-application guard.' }
        } finally {
            $mutex.Dispose()
        }
    }
    Remove-TestInstallation
    if ((Test-Path -LiteralPath $registryPath) -or (Test-Path -LiteralPath (Join-Path $installed 'PheebKit.exe'))) { throw 'Uninstall left application or uninstall registration behind.' }
    if (-not (Test-Path -LiteralPath (Join-Path $installed 'keep-user-file.txt'))) { throw 'Uninstall removed user-owned files.' }
    if ($PreviousReleaseDirectory -and -not (Test-Path -LiteralPath (Join-Path $installed 'keep-upgrade-file.txt'))) { throw 'Uninstall removed upgrade sentinel.' }
    foreach ($path in $configHashes.Keys) {
        $afterHash = if (Test-Path -LiteralPath $path) { (Get-FileHash -LiteralPath $path).Hash } else { '' }
        if ($configHashes[$path] -ne $afterHash) { throw 'Installation modified existing application configuration or history.' }
    }
    try {
        & (Join-Path $PSScriptRoot 'release-windows.ps1')
        throw 'Existing release was unexpectedly overwritten.'
    } catch {
        if ($_.Exception.Message -notmatch 'Release directory exists and will not be overwritten') { throw }
    }
    Write-Host "PASS: standalone/bundled EXEs, sidecar and embedded runtime, checksums, provenance, portable execution, install/reinstall/uninstall, running guards and configuration preservation. Logs: $root"
} finally {
    Remove-TestInstallation
}
