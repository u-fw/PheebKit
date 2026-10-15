# Windows release acceptance

## Beta.11 audio processing acceptance

Local real-media tests verify actual sample rates/channels, PCM envelopes/EQ/noise attenuation, measured single-pass loudness, multi-track concat and nonzero trim. Service queue and project/preset/draft/sample tests cover application boundaries; 118 frontend tests and real-component browser fixtures cover editing and failure states. Production installer/upgrade evidence is recorded in [distribution](windows-distribution.md) after packaging. See [audio processing](audio-processing.md) for exact measurements, known timeline limits and external native/GPU/clean-machine/CI gates.

## Beta.9 media recovery acceptance

The 2026-10-15 beta.9 gate passed complete local checks and actual beta.8 package upgrade, reinstall/uninstall, exact payload/checksum/version tests, mutex protection and AppData hash preservation. Actual corrupt-packet fixtures prove default deletion and black/silence replacement, complete decode success markers and refusal to publish a corrupt remux. A real Service scan and queued recovery prove the backend boundary. Browser component fixtures prove review/application/cancel/precision/Undo/Redo/error states. See [media recovery](media-recovery.md) and [distribution evidence](windows-distribution.md) for precise coverage and limits. New native/GPU recovery combinations, clean-machine/Windows 10, hosted CI and signing remain external; older dated entries below remain historical baselines.

This is a candidate checklist. Passing builds does not establish production stability or clean-machine compatibility. Android is deferred. Publication/push/signing requires the owner's explicit decision.

## Repeatable local checks

1. Use a clean committed checkout, Go 1.27.2+, Node 24+, the pinned Wails CLI and verified Inno compiler. Update `VERSION`, run `node scripts/sync-version.mjs`, write matching `RELEASE_NOTES.md`, then commit.
2. Build frontend assets before loading the root Go package: `npm --prefix frontend ci`, then `npm --prefix frontend run build`. The release script now performs this bootstrap itself. Source text checkout rules keep JS/HTML compatible with formatting checks on Windows.
3. Run `scripts/setup-test-ffmpeg.ps1` once into a fresh `.tools` destination. It verifies the publisher-pinned Gyan archive and extracted binary hashes, sets process-local `KOMPEG_FFMPEG`/`KOMPEG_FFPROBE`, and supplies the same paths to subsequent GitHub steps. No system PATH or application configuration is changed. Existing manifests are refused; this is not an unverified cache.
4. Set `KOMPEG_INTEGRATION=1`; run `go test ./... -count=1` and `go vet ./...`. Run frontend tests, typecheck, lint, formatting and version tests. The process-interruption test kills only its own headless test child; it proves checkpoint persistence and recovery without automatic task execution.
5. Run `scripts/release-windows.ps1`. It refuses dirty sources/existing version directories, audits dependencies, performs a production build and creates Setup, standalone ZIP and SHA-256 manifest. Both packages include identical license, notices and release notes. Inspect `go version -m bin/PheebKit.exe` and run the pinned govulncheck in binary mode.
6. For version updates, build a reviewed existing local tag: `scripts/prepare-upgrade-baseline.ps1 -Ref v0.2.0-beta.7`. It creates an isolated detached worktree, builds the original source and removes the clean worktree. It does not fetch or modify tags. Execute `scripts/verify-windows-release.ps1 -PreviousReleaseDirectory (Get-Content .tools/upgrade-baseline-path.txt -Raw).Trim()`. An existing immutable local package from the verified matching source tag can instead be supplied directly with `-PreviousReleaseDirectory`.

Installer verification checks archive digests, exact current payload equality, actual older-version install → current upgrade, registration/version, running-application mutex refusal, reinstall, uninstall, and preservation of user-owned files and all four existing AppData JSON files. It refuses an existing registered real installation, uses a workspace-owned directory and never launches PheebKit.

## Local evidence and separate manual gates

Beta.8 additionally passes real ordered segment/subtitle/independent-queue tests, 109 frontend checks and production packaging, with successful local beta.7 → beta.8 installer lifecycle/configuration preservation. Source and binary audits have no known findings. Exact source revision and local log path are in [Windows distribution](windows-distribution.md); the [feature acceptance matrix](ffmpeg-expansion.md) distinguishes CPU/browser evidence, native/GPU manual gates and future unimplemented workflows. Diagnostic FIFO now has equal-clock regression coverage.

The beta.7 local record includes publisher-pinned FFmpeg media integration, actual headless process termination/recovery, additive old-record compatibility, diagnostic privacy/snapshot regressions, 101 frontend tests and a clean-source beta.6 baseline build. The release script also succeeded with old frontend assets absent. Actual original-tag beta.6 → beta.7 upgrade, reinstall/uninstall, mutex refusal, payload/checksum comparisons and configuration/user-file preservation passed on the current host. Source/binary govulncheck and npm audit reported no known vulnerabilities. Exact build revision and local log location are recorded in [Windows distribution](windows-distribution.md). Diagnostic rules are in [diagnostics](diagnostics.md).

Before broadly distributing binaries, record the machine/OS/build and pass/fail evidence for these steps:

- Fresh Windows 10 and Windows 11 x64, normal user permissions, both existing and missing WebView2. Check explanatory missing-runtime wizard flow; no silent runtime installation.
- Launch from the installed directory and extracted ZIP. Test native media/open/save dialogs, Unicode/long paths, a read-only output directory, existing output refusal, and missing source relocation.
- Test no-FFmpeg setup/download cancellation and retry, a real CPU-only machine, and an actual qualified GPU. Check visible fallback and cancellation; hosted CPU checks do not qualify a GPU.
- Edit → checkpoint → close/forced termination → restart → explicit restore; verify the retained settings, no automatic execution, and an explicit output operation. The automated headless test is not native renderer/IPC or OS-shutdown coverage.
- Settings → diagnostic preview → save/cancel/expired preview. Compare saved bytes with preview and review attachments before sharing.
- Run the repository's hosted Windows workflow. For an update, dispatch it with a reviewed previous tag. The default workflow runs installer lifecycle but cross-version acceptance is optional and must be requested for the update candidate.
- Choose public repository identity/contact, configure private security reporting, review public history/assets and matching corresponding source, and decide signing. Keep unsigned/Beta status visible. An accessible local tag is not a public source release.

No hosted CI, clean-machine, signing or native-dialog result is inferred from local builds. Never replace already published assets for the same version; publish a new version for changes.
