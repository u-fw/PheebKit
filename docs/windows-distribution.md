# Windows distribution

Windows x64 is the active release target. Android work is deferred: its source is retained, but no APK is packaged or published. Application version is read from `VERSION`. Wails v3 Beta remains the explicitly approved experimental baseline; these are experimental candidates, not a production-stability claim.

## Artifacts

Each version produces exactly three local files under `release/<version>/`:

- `PheebKit-<version>-windows-x64-setup.exe`
- `PheebKit-<version>-windows-x64-standalone.zip`
- `SHA256SUMS.txt`

The installer and ZIP contain identical `PheebKit.exe`, project license, generated third-party notices and usage instructions. There is no separate bare-EXE release. The ZIP is named standalone because settings/cache/downloaded FFmpeg remain in AppData rather than following the executable directory. FFmpeg and WebView2 are external, not bundled.

## Version source and upgrade identity

`VERSION` follows `major.minor.patch` with optional `alpha.N`, `beta.N` or `rc.N`. `node scripts/sync-version.mjs` synchronizes npm package/lock metadata and Windows resources. Vite reads the same file to display the version in Settings. Git release tags use `v<version>`.

Windows numeric resource revisions map alpha.N to N, beta.N to 10000+N, rc.N to 20000+N and final to 65535 (N <= 9999). This orders prereleases below their final release. Display versions retain the complete SemVer string.

The installer AppId is permanently `{9D8FAD06-8029-4A58-9497-96C7DA878C63}` across versions. Existing installations reuse their directory and uninstall registration. Application-running mutex guards preserve support for multiple windows and prevent installation/uninstallation while a window is alive. The installer does not forcibly close media processing. Files not owned by the installer and AppData settings are retained on uninstall.

## Build and package locally

Requires native Windows x64, PowerShell 7, Go 1.25+, Node 24+, the pinned Wails CLI and Inno Setup 7.1.0.

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.27
npm --prefix frontend ci
./scripts/setup-inno.ps1
node scripts/sync-version.mjs
# Commit source/version changes before release packaging.
./scripts/release-windows.ps1
./scripts/verify-windows-release.ps1
```

Compiler provisioning downloads the versioned [official Inno release](https://github.com/jrsoftware/issrc/releases/tag/is-7_1_0), verifies its pinned asset SHA-256 and valid Authenticode publisher (Pyrsys B.V.), then uses the upstream `/PORTABLE=1` mode under `.tools`. No compiler file association or system installation is required. Alternatively pass `-InnoCompiler <absolute-path>` or `KOMPEG_ISCC`.

Packaging requires clean committed sources, runs Go tests/vet, frontend tests/typecheck/lint/format and version tests, then performs one production Wails build. It checks the EXE version, generates notices for actual Go/production frontend dependencies, packages both deliverables and computes SHA-256. Existing version output directories are never overwritten. Generated packages and tools are ignored by Git. No credentials, repository URL or publishing identity is invented.

The installer installs per user to `%LOCALAPPDATA%/Programs/PheebKit`, supports Simplified Chinese and English, creates Start Menu entries and offers an optional desktop shortcut. It detects Microsoft's documented WebView2 registry values. If missing, the finish page explains the requirement and offers the official download page; it does not silently install a runtime. Silent installation never launches the app or a browser. FFmpeg remains an explicit in-app setup operation and system PATH is unchanged.

## CI and publication

`.github/workflows/windows.yml` runs Windows verification/packaging for PRs, main, version tags and manual dispatch. Official actions are pinned to verified release commit SHAs. Tagged builds must match `VERSION`. Candidate files are uploaded as workflow artifacts; repository permissions are read-only and no GitHub Release is created/published. Hosted execution remains unverified until run in the owner's repository.

For publication, create the matching tag, draft a GitHub Release and attach these three files. Mark beta/rc builds as prereleases and include changed behavior, runtime requirements and known limitations. Once published, do not replace that version's assets: changes get a new version. Repository/tag/release publication still requires explicit owner authorization.

## Local verification and remaining acceptance

`0.2.0-beta.1` adds batch scheduling, track selection, remux/extraction, transforms, samples and project/preset sharing; policies and evidence are in [media tools](media-tools.md). `0.1.0-beta.4` added [task/preview/history workflows](workflows.md). Beta.3 introduced PheebKit branding/time selectors; beta.2 added single-range trimming. Earlier artifacts remain unchanged.

The current Windows host passed checksum validation, exact installer/ZIP payload comparison, display/version checks, per-user install, same-version reinstall, uninstall, preservation of a user-owned sentinel and existing FFmpeg configuration, plus blocked install/uninstall with the application mutex present. Verification uses a fresh workspace directory and refuses to run if an existing kompeg installation is registered. Logs remain under `.tools`.

This is not a clean-machine or Windows 10 test. New-version upgrade from a previously published package, missing-WebView2 wizard behavior, application launch from the installed location, signing and hosted CI execution remain unverified. Project candidates are unsigned. Native app click-through was previously blocked by automatic approval review and was not retried by this packaging task.

## Existing FFmpeg safeguards

One-click setup is for incomplete environments. A working FFmpeg/ffprobe pair disables download with a visible explanation, and native entry points repeat actual version checks before downloading. Explicit FFmpeg-related variables block downloads and configuration writes until corrected/removed and the app restarted. Ordinary PATH installations can still be deliberately switched through local-path selection. Existing active-job and duplicate-download guards remain in place.
