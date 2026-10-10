# Open-source readiness — 2026-10-10

The Windows experimental slice has passed its current checks after the visual refinement and defect fixes. No remote repository, push, release or publication was performed. This review does not claim the absence of all bugs or production stability.

## Repaired findings

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
- Current build is `bin/kompeg.exe`. Generated binaries, dependency folders, local verification media, test configs and archived prototypes are ignored by Git. The showcase screenshot contains no imported user file.

## Before publishing

1. The owner must select a source license; the repository currently has no `LICENSE`. Distribution notices and FFmpeg packaging policy must be settled before distributing bundled third-party binaries. FFmpeg is currently discovered locally and is not bundled.
2. Choose the remote repository and publication identity. No Git remote is configured. Review author/contact metadata and any machine-specific verification notes before making history public.
3. Add automatic Windows build/unit/frontend checks in CI. Current checks are local; GPU/real-FFmpeg integration needs an appropriate test machine and cannot be inferred from a generic hosted runner.
4. Keep the experimental scope clear: pinned Wails v3 Beta was approved only for this Technical Spike; Android APK/device execution, HDR encoding, installer/signing and later editing/project-management features remain unverified or unimplemented. Windows 10 also needs a separate runtime smoke test before a broad binary-release claim.

An Android shared-package cross-build verifies source portability only. The full cross-platform acceptance criteria remain open until an APK launches on a device or emulator and proves the Go bridge.
