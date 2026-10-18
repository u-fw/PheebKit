# Open-source readiness — baseline 2026-10-10, stabilization 2026-10-18

Current work targets `0.2.0` with a stable Wails v2.17.0 shell, explicitly chosen by the owner on 2026-10-18. The dated Beta records below remain historical evidence. The active scope, native observations and outstanding release gates are maintained in [stabilization](stability.md); final packaging requires exact-revision reviewed records. Current-window native GPU compression/import/export evidence does not substitute for clean Windows 10/11, CPU-only or hosted CI acceptance.

Beta.11 adds opt-in typed audio processing with real PCM/loudness/noise/rate/channel evidence, post-concat whole-output envelopes and actual Service queue/persistence checks. Dynamic single-pass normalization is distinguished from future measured two-pass mastering workflows. [Audio evidence](audio-processing.md) records measured fixtures and remaining native/manual limits; it is not a universal audio-quality guarantee.

Beta.9 adds conservative decode scanning/recovery and mandatory output decoding for recovery jobs. Real corrupted H.264 packets exercise deletion, black/silence replacement, edge intervals and refusal to publish a readable-but-corrupt remux. Service queue/lifecycle and frontend regressions supplement browser fixture checks. See [recovery evidence and bounds](media-recovery.md). This extends CPU-verified media workflows; new native/GPU combinations and remaining publication gates below still require separate acceptance.

Beta.8 adds ordered same-source editing/exports, configurable text subtitle burn-in and local capability/help discovery. Real CPU FFmpeg tests verify reversed color-frame order, duration/audio, -6 dB volume, silent joins, embedded/external burn, nonzero trim, subtitle offset, combined burn/join and changed-subtitle refusal. The frontend has 109 tests; disposable browser fixtures verify direct time editing, reorder/export intent, subtitle project persistence/Undo/Redo, invalid-font refusal, search/help/pagination and 375px dark layout. Native file-dialog/installed-window, clean-machine and hosted-CI acceptance remain separate. See [expansion contract and coverage](ffmpeg-expansion.md). Earlier dated checks below remain historical evidence, not an exhaustive FFmpeg capability claim.

Current branding is 菲比工具箱 / PheebKit. The previous beta.7 iteration added allowlisted local diagnostic preview/export, actual headless process-kill recovery checks, old-record compatibility checks and repeatable previous-tag upgrade acceptance. It also fixed fresh-checkout frontend bootstrap and Windows text checkout problems exposed by building beta.6 from its original tag. Beta.6 editing controls remain. See [diagnostics](diagnostics.md), [release acceptance](release-acceptance.md), [editor evidence](editor-recovery.md) and [workflows](workflows.md). The dated baseline below describes earlier verification. No remote push, hosted release or publication was performed. This review does not claim all bugs are absent or production stability.

## Repaired findings

Beta.7 local acceptance passed on 2026-10-14: pinned FFmpeg real-media integration, actual headless process-kill recovery, old-record reads, diagnostic privacy/snapshot protection, 101 frontend tests and production packaging from committed source with no cached frontend assets. The original beta.6 tag was rebuilt and successfully installed/upgraded to beta.7; configuration hashes/user files survived upgrade/reinstall/uninstall and running-process mutex guards held. Source/binary govulncheck and npm audit found no known vulnerabilities. These are local results; [release acceptance](release-acceptance.md) keeps hosted CI, clean Windows 10/11, native click-through, signing and public reporting-channel gates open.

| Finding | Correction | Evidence |
| --- | --- | --- |
| Codec-private parameters could write unmanaged side files | Typed positive allowlists reject file access, multi-pass and unknown suboptions | Compiler regressions and three real software encodes; sentinel unchanged, no unexpected files |
| Replaced source could retain stale probe metadata | Fingerprints propagate from probe to plan and queued execution; changed source requires re-import | Same-size/restored-time replacement and real media/queued-plan tests |
| Shared configuration could differ from the executing instance | Status reflects the instance's actual binaries; idle rescan validates and synchronizes shared changes | Two real service instances, active-job and invalid-config regressions |
| Old snapshots could overwrite newer queue events | Request revisions protect newer events, terminal state and progress | Event/snapshot/initialization/refresh regressions and native completed refresh |
| Frontend history grew indefinitely | Oldest terminal records are pruned while active jobs remain | Chronological 200-record limit tests |
| Inactive quality parameter blocked bitrate/size mode | Quality validation is conditional on quality mode | Unit regressions and native quality-99 → bitrate interaction |
| Architecture and UI descriptions were stale | GPU qualification, private-option policy and selection behavior documented accurately | Documentation compared to current implementation |

## Verified baseline

- Windows production Wails window, native bridge, real media artwork and GPU/CPU compilation.
- Real NVENC HEVC/AAC output inspected independently with ffprobe.
- Full Go integration/unit checks, vet and formatting; 38 frontend tests, typecheck, lint, formatting and production build.
- Native light/dark, phone/landscape, enlarged text, reduced motion and axe checks; details in [verification](verification.md).
- Lockfile preserved; npm registry audit returned zero known vulnerabilities. Basic tracked-file/key-pattern checks found no matches. Neither check guarantees complete supply-chain or security coverage.
- Current build is `bin/PheebKit.exe`. Generated binaries, dependency folders, local verification media, test configs and archived prototypes are ignored by Git. Historical screenshots show earlier kompeg builds; they are not current branding evidence.

## Before publishing

1. The owner selected `GPL-3.0-only`; the full license is in the repository's `LICENSE` and the scope is recorded in the README. FFmpeg is discovered locally or explicitly downloaded from upstream/mirror sources into a private directory; see [download policy](ffmpeg-downloads.md). It is not bundled in the executable. Distribution notices and corresponding-source policy must still be settled before distributing project-hosted copies or offline bundles of third-party binaries.
2. Choose the remote repository and publication identity. No Git remote is configured. Review author/contact metadata and any machine-specific verification notes before making history public.
3. Windows CI configuration now provisions publisher-pinned FFmpeg and runs real media checks, builds/packages candidates and verifies installer lifecycle. Manual dispatch accepts a reviewed previous-version tag for isolated build/upgrade acceptance. Hosted execution is still unverified. GPU qualification needs an appropriate test machine and cannot be inferred from a generic hosted CPU runner.
4. Keep the experimental scope clear: pinned Wails v3 Beta was approved only for this Technical Spike; Android is deferred. APK/device execution, HDR encoding, signing, clean-machine installation and multi-clip editing remain unverified or unimplemented. New native dialogs and sample comparison need native click-through. Local installer lifecycle checks are in [Windows distribution](windows-distribution.md). Windows 10 also needs a separate runtime smoke test before a broad binary-release claim.

An Android shared-package cross-build verifies source portability only. The full cross-platform acceptance criteria remain open until an APK launches on a device or emulator and proves the Go bridge.
