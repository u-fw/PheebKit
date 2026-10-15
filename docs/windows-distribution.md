# Windows distribution

Windows x64 is the active release target. Android work is deferred: its source is retained, but no APK is packaged or published. Application version is read from `VERSION`. Wails v3 Beta remains the explicitly approved experimental baseline; these are experimental candidates, not a production-stability claim.

## Artifacts

Each version produces exactly three local files under `release/<version>/`:

- `PheebKit-<version>-windows-x64-setup.exe`
- `PheebKit-<version>-windows-x64-standalone.zip`
- `SHA256SUMS.txt`

The installer and ZIP contain identical `PheebKit.exe`, project license, generated third-party notices, usage instructions and version-matched `RELEASE_NOTES.md`. There is no separate bare-EXE release. The ZIP is named standalone because settings/cache/downloaded FFmpeg remain in AppData rather than following the executable directory. FFmpeg and WebView2 are external, not bundled.

## Version source and upgrade identity

`VERSION` follows `major.minor.patch` with optional `alpha.N`, `beta.N` or `rc.N`. `node scripts/sync-version.mjs` synchronizes npm package/lock metadata, Windows resources and the diagnostic Go version. Vite reads the same file to display the version in Settings. Git release tags use `v<version>`.

Windows numeric resource revisions map alpha.N to N, beta.N to 10000+N, rc.N to 20000+N and final to 65535 (N <= 9999). This orders prereleases below their final release. Display versions retain the complete SemVer string.

The installer AppId is permanently `{9D8FAD06-8029-4A58-9497-96C7DA878C63}` across versions. Existing installations reuse their directory and uninstall registration. Application-running mutex guards preserve support for multiple windows and prevent installation/uninstallation while a window is alive. The installer does not forcibly close media processing. Files not owned by the installer and AppData settings are retained on uninstall.

## Build and package locally

Requires native Windows x64, PowerShell 7, stable Go 1.27.2 or newer, Node 24+, the pinned Wails CLI and Inno Setup 7.1.0. The release script runs pinned govulncheck and npm audit before building; network/advisory failures stop packaging. The module retains Go 1.25 language compatibility with an automatic Go 1.27.2 toolchain hint.

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

Packaging requires clean committed sources, builds real frontend assets before Go package loading, runs Go tests/vet, frontend tests/typecheck/lint/format and version tests, then performs one production Wails build. It checks the EXE/notes version, generates notices for actual Go/production frontend dependencies, packages both deliverables and computes SHA-256. Existing version output directories are never overwritten. Generated packages and tools are ignored by Git. No credentials, repository URL or publishing identity is invented.

The installer installs per user to `%LOCALAPPDATA%/Programs/PheebKit`, supports Simplified Chinese and English, creates Start Menu entries and offers an optional desktop shortcut. It detects Microsoft's documented WebView2 registry values. If missing, the finish page explains the requirement and offers the official download page; it does not silently install a runtime. Silent installation never launches the app or a browser. FFmpeg remains an explicit in-app setup operation and system PATH is unchanged.

## CI and publication

`.github/workflows/windows.yml` runs Windows verification/packaging for PRs, main, version tags and manual dispatch. Official actions are pinned to verified release commit SHAs. Tagged builds must match `VERSION`. Candidate files are uploaded as workflow artifacts; repository permissions are read-only and no GitHub Release is created/published. Hosted execution remains unverified until run in the owner's repository.

CI downloads the publisher-pinned test FFmpeg via `scripts/setup-test-ffmpeg.ps1` and runs real media tests. For cross-version checks, manual dispatch accepts `upgrade_from_tag`; `scripts/prepare-upgrade-baseline.ps1` builds that reviewed local tag in an isolated worktree, then `verify-windows-release.ps1 -PreviousReleaseDirectory <candidate-directory>` installs it before the new candidate. Default lifecycle checks do not by themselves establish cross-version upgrade compatibility. See [release acceptance](release-acceptance.md) for commands and gates.

For publication, create the matching tag, draft a GitHub Release and attach these three files. Mark beta/rc builds as prereleases and include changed behavior, runtime requirements and known limitations. Once published, do not replace that version's assets: changes get a new version. Repository/tag/release publication still requires explicit owner authorization.

## Local verification and remaining acceptance

Beta.12 (2026-10-15) fixes the settings autosave control and text-input CSS scope. Production artifacts were built from clean source `0a932a64ab56dfc9308646017940d4a23d370edb`, Go 1.27.2, `vcs.modified=false`. The production gate passed the Go suite with integration enabled, vet, 118 frontend tests in 23 files, TypeScript/ESLint/Prettier, three version tests, Wails production build and source/binary govulncheck/npm audit with no known findings. Actual beta.11 → beta.12 upgrade, reinstall/uninstall, exact payload/checksum/version checks, running guards and AppData/user-file preservation passed. Logs: `.tools/installer-test-a0a05d08e63447cfbdbddd91b157cfef`. Browser checks and limits are recorded in [editor recovery](editor-recovery.md). The native application was not launched; native-window, clean Windows 10/11 and hosted CI acceptance remain external. Temporary fixtures/server/tabs were removed. Assets remain local and unsigned.

Beta.11 (2026-10-15) adds configurable audio processing and fixes decimal denoise entry. Its production artifacts were built from clean source `81cbf8d59048241b093b1cde74b36901ba6f79e2`, Go 1.27.2, `vcs.modified=false`. The gate passed real FFmpeg media tests/vet, 118 frontend tests in 23 files, TypeScript/ESLint/Prettier, three version tests, Wails production build and source/binary govulncheck/npm audit (no known findings). Actual beta.9 → beta.11 upgrade, identical installer/ZIP payloads, checksums/display versions, reinstall/uninstall, mutex protection and configuration/user-file preservation passed. Logs: `.tools/installer-test-0142260de9774d86ab59b7be554ff5ad`. No PheebKit window was launched. Native installed-window/GPU-audio combinations, clean Windows 10/11, hosted CI and signing remain external. [Audio evidence and limits](audio-processing.md) documents actual measurements and browser checks.

Beta.11 setup SHA-256: `21f7cbb1bd239839fac8787166419767efc364ba133dd936903489e3ba92bf80`; standalone ZIP: `1c61a9a94d481f989095632b93487b44601c3cc2eed5bace21cf6f2992a7a76e`. Beta.10 remains an unchanged superseded local candidate from `abe5c2ba5591038beae40bd24fe72e6d3c115f31`; it passed build gates but the follow-up UI test exposed zero-value input disappearing, so beta.11 is the delivered candidate. Neither version was remotely published.

Beta.9 (2026-10-15) was built from clean committed source `ce1570bfb529d97e72c2c7fa8551395dd2f37861`; EXE metadata reports Go 1.27.2, that revision and `vcs.modified=false`. The release gate passed real FFmpeg corruption/recovery and full-regression Go tests/vet, 113 frontend tests in 22 files, TypeScript/ESLint/Prettier, three version tests, Wails production build, source/binary govulncheck and npm audit (no known findings). Checksums, exact installer/ZIP payloads and display versions passed. Actual beta.8 → beta.9 install/upgrade/reinstall/uninstall, running mutex guards and configuration/user-file preservation passed; logs remain in `.tools/installer-test-2606d57de6dc4b489ef87470072a56ff`. No PheebKit window was launched. New native/GPU recovery flows, clean Windows 10/11, hosted CI and signing remain separate gates. [Recovery evidence](media-recovery.md) describes actual CPU/browser tests and limits.

Beta.9 setup SHA-256: `57306af0e2d2d163687cff7aa405ecc45732b919f8d7aea18bd2bab7290b3318`; standalone ZIP: `7f04916d0c08f999eed302fc82f69837f25eafeb32363577553f0aa58cee5c61`. The version's artifacts are retained unchanged; no remote repository or hosted release was created.

Beta.8 (2026-10-15) production packages were built from clean committed source `5d46ea2b2f84a591d25d25e02cc5f55f99d42fe5`; the EXE reports Go 1.27.2 and `vcs.modified=false`. The release gate passed real-media Go tests with `KOMPEG_INTEGRATION=1`, vet, 109 frontend tests/typecheck/lint/format, three version tests, source/binary govulncheck and npm audit (no known findings). Actual beta.7 package → beta.8 upgrade, exact installer/ZIP payload checks, checksum/version checks, reinstall/uninstall, mutex refusal and preservation of configuration/user sentinels passed. Logs remain in `.tools/installer-test-ea10d1ad12a244ee9b2ec850bf8fa5c7`. No app window was launched. Native subtitle picking, the new flows on GPU, installed-window interaction, clean Windows 10/11 and hosted CI remain separate acceptance items. See [feature evidence](ffmpeg-expansion.md).

The first beta.8 release attempt exposed equal Windows timestamps making diagnostic preview FIFO eviction nondeterministic; it stopped before producing a final version directory. The cache now uses a monotonic sequence for FIFO ordering and timestamps only for expiration. Equal-timestamp regression and 30 repeated diagnostic test runs passed before the successful rebuild. No failed candidate was published or overwritten.

Beta.6 adds local `editor.json` for draft/recovery preferences, checkpoints and recent projects. Installer/uninstaller preserve it alongside FFmpeg settings and history; lifecycle verification now includes it in configuration-hash comparisons. See [editing/recovery](editor-recovery.md) for the user-controlled retention and privacy policy.

`0.2.0-beta.1` adds batch scheduling, track selection, remux/extraction, transforms, samples and project/preset sharing; policies and evidence are in [media tools](media-tools.md). `0.1.0-beta.4` added [task/preview/history workflows](workflows.md). Beta.3 introduced PheebKit branding/time selectors; beta.2 added single-range trimming. Earlier artifacts remain unchanged.

On 2026-10-14, beta.7 built successfully from committed source `d4a12af1f468e4c5896f2c778fcba50ac13f3700` with the existing `frontend/dist` moved aside. The updated release script regenerated real assets before Go checks. Source/binary govulncheck and npm audit found no known vulnerabilities; Go checks, 101 frontend tests, typecheck, lint, formatting and three version tests passed. Production EXE metadata reports Go 1.27.2, the same revision and `vcs.modified=false`.

The current Windows host passed checksum validation, exact installer/ZIP payload comparison including release notes, display/version checks, original-tag beta.6 installation → beta.7 upgrade, same-version reinstall and uninstall. User-owned upgrade/reinstall sentinels and hashes of the four existing AppData JSON files were retained; install/uninstall were blocked with the application mutex present. The beta.6 baseline was rebuilt from its original tag in an isolated clean worktree. Verification used a fresh workspace directory and refused an existing registered real installation. This run's logs remain in `.tools/installer-test-6e16113e33dd4864a131f12db6935ade`; the temporary baseline worktree was removed.

This is not a clean-machine or Windows 10 test. The upgrade baseline is a locally rebuilt original tag, not an independently downloaded published package. Missing-WebView2 wizard behavior, application launch from the installed location, signing and hosted CI execution remain unverified. Project candidates are unsigned. Native app click-through was previously blocked by automatic approval review and was not retried by this packaging task.

## Existing FFmpeg safeguards

One-click setup is for incomplete environments. A working FFmpeg/ffprobe pair disables download with a visible explanation, and native entry points repeat actual version checks before downloading. Explicit FFmpeg-related variables block downloads and configuration writes until corrected/removed and the app restarted. Ordinary PATH installations can still be deliberately switched through local-path selection. Existing active-job and duplicate-download guards remain in place.
