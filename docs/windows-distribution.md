# Windows distribution

Windows x64 is the active release target. Android is deferred and no APK is packaged. Application version is read from `VERSION`. The owner approved stable Wails v2.17.0 migration on 2026-10-18. RC packages are verification candidates; final-version packaging additionally requires exact-revision acceptance evidence from [stabilization](stability.md).

## Artifacts

Each version produces exactly five local files under `release/<version>/`:

- `PheebKit-<version>-windows-x64.exe`: lightweight direct-run application; FFmpeg remains external.
- `PheebKit-<version>-windows-x64-ffmpeg.exe`: direct-run application containing the verified FFmpeg payload.
- `PheebKit-<version>-windows-x64-portable-ffmpeg.zip`: lightweight application plus adjacent `runtime/ffmpeg`.
- `PheebKit-<version>-windows-x64-setup-ffmpeg.exe`: optional per-user installer containing the bundled direct-run application.
- `SHA256SUMS.txt`: exact SHA-256 for every artifact above.

The two bundled forms share one staged runtime and its provenance. The direct EXE atomically materializes it into `%LOCALAPPDATA%/PheebKit/runtime/ffmpeg` on first use; the portable ZIP uses its adjacent directory without extraction into a shared cache. The installer contains the same bundled EXE and creates no second runtime copy during installation. Settings remain in AppData. WebView2 remains a Windows runtime requirement in every edition, so product text says “FFmpeg included” rather than “fully offline.”

The bundled runtime is the publisher-pinned Gyan 9.0.2 full build. It contains `ffmpeg.exe`, `ffprobe.exe`, preset metadata, FFmpeg's GPL v3 license, Gyan README, exact source commit, archive URL/digest and executable hashes. The app validates the embedded ZIP digest, metadata and both executable hashes before selecting the runtime.

## Version source and upgrade identity

`VERSION` follows `major.minor.patch` with optional `alpha.N`, `beta.N` or `rc.N`. `node scripts/sync-version.mjs` synchronizes npm package/lock metadata, Windows resources and the diagnostic Go version. Vite reads the same file to display the version in Settings. Git release tags use `v<version>`.

Windows numeric resource revisions map alpha.N to N, beta.N to 10000+N, rc.N to 20000+N and final to 65535 (N <= 9999). This orders prereleases below their final release. Display versions retain the complete SemVer string.

The installer AppId is permanently `{9D8FAD06-8029-4A58-9497-96C7DA878C63}` across versions. Existing installations reuse their directory and uninstall registration. Both current and legacy application-running mutex guards preserve support for multiple windows and prevent installation/uninstallation while a window is alive. The installer does not forcibly close media processing. Files not owned by the installer and AppData settings are retained on uninstall.

## Build and package locally

Requires native Windows x64, PowerShell 7, stable Go 1.27.3 or newer, Node 24+, the pinned Wails CLI, Inno Setup 7.1.0 and a verified `PHEEBKIT_FFMPEG` / `PHEEBKIT_FFPROBE` pair from the pinned archive. The release script runs pinned source govulncheck and npm audit before building, audits both actual production EXEs, then checks provenance, payload metadata and extraction before archiving; network/advisory failures stop packaging.

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
npm --prefix frontend ci
./scripts/setup-inno.ps1
node scripts/sync-version.mjs
# Commit source/version changes before release packaging.
./scripts/release-windows.ps1
./scripts/verify-windows-release.ps1
```

Compiler provisioning downloads the versioned [official Inno release](https://github.com/jrsoftware/issrc/releases/tag/is-7_1_0), verifies its pinned asset SHA-256 and valid Authenticode publisher (Pyrsys B.V.), then uses the upstream `/PORTABLE=1` mode under `.tools`. No compiler file association or system installation is required. Alternatively pass `-InnoCompiler <absolute-path>` or `PHEEBKIT_ISCC`.

Packaging requires clean committed sources, builds real frontend assets before Go package loading, runs Go tests/vet, frontend tests/typecheck/lint/format and version tests, then performs lightweight and bundled Wails builds with the same reviewed source revision. It checks each EXE's build metadata, rejects a payload digest/version mismatch, generates notices for both distribution modes, packages all four application forms and computes SHA-256. Existing version output directories are never overwritten. Generated payload archives, packages and tools are ignored by Git.

The installer installs per user to `%LOCALAPPDATA%/Programs/PheebKit`, supports Simplified Chinese and English, creates Start Menu entries and offers an optional desktop shortcut. It detects Microsoft's documented WebView2 registry values. If missing, the finish page explains the requirement and offers the official download page; it does not silently install a runtime. Silent installation never launches the app or a browser. FFmpeg is already embedded, and system PATH is unchanged.

## CI and publication

`.github/workflows/windows.yml` always verifies source, real FFmpeg processing, frontend quality and a production binary. Ordinary PR/main verification does not require stable-release evidence, even when `VERSION` is final. The tested event planner selects packaging separately:

| Event                              | Verification | Packaging and installer lifecycle             | Publication         |
| ---------------------------------- | ------------ | --------------------------------------------- | ------------------- |
| PR or main push, any version       | Always       | No                                            | No                  |
| Matching prerelease tag push       | Always       | Yes                                           | Prerelease          |
| Matching final tag push            | Always       | No; dispatch with reviewed evidence           | No                  |
| Manual `checks_only=true`          | Always       | No                                            | No                  |
| Manual candidate, no `release_tag` | Always       | Yes; final versions require `stable_evidence` | No                  |
| Manual matching `release_tag`      | Always       | Yes; final versions require `stable_evidence` | Prerelease or final |

Official actions remain pinned to verified release commit SHAs. The verification job has `contents: read`; only a dependent publish job receives `contents: write`. A separate attestation job receives only `attestations: write`, `contents: read` and `id-token: write`; it downloads the candidate artifact, attests all five files without executing them, and must succeed before publication starts. `actions/attest` v4.2.2 creates Sigstore-backed SLSA build provenance binding each file digest to this repository, workflow and source revision. Initial checkout uses the event SHA or selected existing tag, and publication checks out the exact verified commit. Before upload and again before publishing the draft, the remote lightweight/annotated tag must still resolve to that commit. The upload validator compares the checksum manifest with the digest emitted by the verification job, checks all four asset digests and exact filenames, and reads the portable ZIP's bounded `BUILD_INFO.json` and application hash without executing an artifact in the write-permission job. Publication refuses an existing release, creates a draft with `--verify-tag` and uploads without overwrite. GitHub's immutable-release setting and tag-protection rules are still separate repository-owner controls.

Packaged candidates remain available as workflow artifacts. Source revision `94665cb` passed [packaging run 40275376229](https://github.com/u-fw/PheebKit/actions/runs/40275376229) and [checks-only run 40275353795](https://github.com/u-fw/PheebKit/actions/runs/40275353795); rc.5 is published as a prerelease. Those runs predate the event-planning, isolated-upload and artifact-attestation changes, so rc.5 has no retroactive provenance claim; new source requires its own hosted result and attested candidate.

Verify a downloaded candidate or publication with GitHub CLI:

```powershell
gh attestation verify .\PheebKit-<版本>-windows-x64.exe --repo u-fw/PheebKit
```

Run the command against each downloaded file, including the ZIP and installer. Provenance establishes the workflow and source identity behind exact bytes; it does not replace the SHA-256 manifest, publisher code signing or the clean-machine acceptance matrix.

CI downloads the publisher-pinned test FFmpeg via `scripts/setup-test-ffmpeg.ps1` and runs real media tests. For cross-version checks, manual dispatch accepts `upgrade_from_tag`; `scripts/prepare-upgrade-baseline.ps1` builds that reviewed local tag in an isolated worktree, then `verify-windows-release.ps1 -PreviousReleaseDirectory <candidate-directory>` installs it before the new candidate. Default lifecycle checks do not by themselves establish cross-version upgrade compatibility. See [release acceptance](release-acceptance.md) for commands and gates.

Publication is fail-closed: a prerelease tag can publish automatically; for a final version, dispatch the existing matching tag through `release_tag`, leave `checks_only` false and supply reviewed exact-revision `stable_evidence`. The workflow publishes the five tested files without replacing an existing release. Beta/rc builds are marked as prereleases and include changed behavior, runtime requirements and known limitations. Changes after publication get a new version. Repository/tag/release publication still requires explicit owner authorization.

## Local verification and remaining acceptance

Beta.12 (2026-10-15) fixes the settings autosave control and text-input CSS scope. Production artifacts were built from clean source `0a932a64ab56dfc9308646017940d4a23d370edb`, Go 1.27.2, `vcs.modified=false`. The production gate passed the Go suite with integration enabled, vet, 118 frontend tests in 23 files, TypeScript/ESLint/Prettier, three version tests, Wails production build and source/binary govulncheck/npm audit with no known findings. Actual beta.11 → beta.12 upgrade, reinstall/uninstall, exact payload/checksum/version checks, running guards and AppData/user-file preservation passed. Logs: `.tools/installer-test-a0a05d08e63447cfbdbddd91b157cfef`. Browser checks and limits are recorded in [editor recovery](editor-recovery.md). The native application was not launched; native-window, clean Windows 10/11 and hosted CI acceptance remain external. Temporary fixtures/server/tabs were removed. Assets remain local and unsigned.

Beta.11 (2026-10-15) adds configurable audio processing and fixes decimal denoise entry. Its production artifacts were built from clean source `81cbf8d59048241b093b1cde74b36901ba6f79e2`, Go 1.27.2, `vcs.modified=false`. The gate passed real FFmpeg media tests/vet, 118 frontend tests in 23 files, TypeScript/ESLint/Prettier, three version tests, Wails production build and source/binary govulncheck/npm audit (no known findings). Actual beta.9 → beta.11 upgrade, identical installer/ZIP payloads, checksums/display versions, reinstall/uninstall, mutex protection and configuration/user-file preservation passed. Logs: `.tools/installer-test-0142260de9774d86ab59b7be554ff5ad`. No PheebKit window was launched. Native installed-window/GPU-audio combinations, clean Windows 10/11, hosted CI and signing remain external. [Audio evidence and limits](audio-processing.md) documents actual measurements and browser checks.

Beta.11 setup SHA-256: `21f7cbb1bd239839fac8787166419767efc364ba133dd936903489e3ba92bf80`; standalone ZIP: `1c61a9a94d481f989095632b93487b44601c3cc2eed5bace21cf6f2992a7a76e`. Beta.10 remains an unchanged superseded local candidate from `abe5c2ba5591038beae40bd24fe72e6d3c115f31`; it passed build gates but the follow-up UI test exposed zero-value input disappearing, so beta.11 is the delivered candidate. Neither version was remotely published.

Beta.9 (2026-10-15) was built from clean committed source `ce1570bfb529d97e72c2c7fa8551395dd2f37861`; EXE metadata reports Go 1.27.2, that revision and `vcs.modified=false`. The release gate passed real FFmpeg corruption/recovery and full-regression Go tests/vet, 113 frontend tests in 22 files, TypeScript/ESLint/Prettier, three version tests, Wails production build, source/binary govulncheck and npm audit (no known findings). Checksums, exact installer/ZIP payloads and display versions passed. Actual beta.8 → beta.9 install/upgrade/reinstall/uninstall, running mutex guards and configuration/user-file preservation passed; logs remain in `.tools/installer-test-2606d57de6dc4b489ef87470072a56ff`. No PheebKit window was launched. New native/GPU recovery flows, clean Windows 10/11, hosted CI and signing remain separate gates. [Recovery evidence](media-recovery.md) describes actual CPU/browser tests and limits.

Beta.9 setup SHA-256: `57306af0e2d2d163687cff7aa405ecc45732b919f8d7aea18bd2bab7290b3318`; standalone ZIP: `7f04916d0c08f999eed302fc82f69837f25eafeb32363577553f0aa58cee5c61`. The version's artifacts are retained unchanged; no remote repository or hosted release was created.

Beta.8 (2026-10-15) production packages were built from clean committed source `5d46ea2b2f84a591d25d25e02cc5f55f99d42fe5`; the EXE reports Go 1.27.2 and `vcs.modified=false`. The release gate passed real-media Go tests with `PHEEBKIT_INTEGRATION=1`, vet, 109 frontend tests/typecheck/lint/format, three version tests, source/binary govulncheck and npm audit (no known findings). Actual beta.7 package → beta.8 upgrade, exact installer/ZIP payload checks, checksum/version checks, reinstall/uninstall, mutex refusal and preservation of configuration/user sentinels passed. Logs remain in `.tools/installer-test-ea10d1ad12a244ee9b2ec850bf8fa5c7`. No app window was launched. Native subtitle picking, the new flows on GPU, installed-window interaction, clean Windows 10/11 and hosted CI remain separate acceptance items. See [feature evidence](ffmpeg-expansion.md).

The first beta.8 release attempt exposed equal Windows timestamps making diagnostic preview FIFO eviction nondeterministic; it stopped before producing a final version directory. The cache now uses a monotonic sequence for FIFO ordering and timestamps only for expiration. Equal-timestamp regression and 30 repeated diagnostic test runs passed before the successful rebuild. No failed candidate was published or overwritten.

Beta.6 adds local `editor.json` for draft/recovery preferences, checkpoints and recent projects. Installer/uninstaller preserve it alongside FFmpeg settings and history; lifecycle verification now includes it in configuration-hash comparisons. See [editing/recovery](editor-recovery.md) for the user-controlled retention and privacy policy.

`0.2.0-beta.1` adds batch scheduling, track selection, remux/extraction, transforms, samples and project/preset sharing; policies and evidence are in [media tools](media-tools.md). `0.1.0-beta.4` added [task/preview/history workflows](workflows.md). Beta.3 introduced PheebKit branding/time selectors; beta.2 added single-range trimming. Earlier artifacts remain unchanged.

On 2026-10-14, beta.7 built successfully from committed source `d4a12af1f468e4c5896f2c778fcba50ac13f3700` with the existing `frontend/dist` moved aside. The updated release script regenerated real assets before Go checks. Source/binary govulncheck and npm audit found no known vulnerabilities; Go checks, 101 frontend tests, typecheck, lint, formatting and three version tests passed. Production EXE metadata reports Go 1.27.2, the same revision and `vcs.modified=false`.

The current Windows host passed checksum validation, exact installer/ZIP payload comparison including release notes, display/version checks, original-tag beta.6 installation → beta.7 upgrade, same-version reinstall and uninstall. User-owned upgrade/reinstall sentinels and hashes of the four existing AppData JSON files were retained; install/uninstall were blocked with the application mutex present. The beta.6 baseline was rebuilt from its original tag in an isolated clean worktree. Verification used a fresh workspace directory and refused an existing registered real installation. This run's logs remain in `.tools/installer-test-6e16113e33dd4864a131f12db6935ade`; the temporary baseline worktree was removed.

This is not a clean-machine or Windows 10 test. The upgrade baseline is a locally rebuilt original tag, not an independently downloaded published package. Missing-WebView2 wizard behavior, application launch from the installed location, signing and hosted CI execution remain unverified. Project candidates are unsigned. Native app click-through was previously blocked by automatic approval review and was not retried by this packaging task.

## Existing FFmpeg safeguards

One-click setup is for incomplete environments. A working FFmpeg/ffprobe pair disables download with a visible explanation, and native entry points repeat actual version checks before downloading. Explicit FFmpeg-related variables block downloads and configuration writes until corrected/removed and the app restarted. Ordinary PATH installations can still be deliberately switched through local-path selection. Existing active-job and duplicate-download guards remain in place.

rc.3 uses current PheebKit public names and tests both current/legacy running guards plus hashes for both configuration roots. See [compatibility](compatibility.md).
