# PheebKit 0.2.0 stabilization

The release scope is Windows 10/11 x64 local media processing. Freeze new feature work while closing the acceptance matrix. Android, network/capture, HDR, arbitrary multi-source timelines and unrestricted FFmpeg authoring are outside this release. Missing future features do not become silent stable-release promises.

On 2026-10-18 the owner explicitly chose migration from Wails v3 Beta to stable Wails v2.17.0. Go and npm v3 runtimes are removed. The existing Go media engine, React interface, schema-1 files, AppData paths and installer identity are retained. The native adapter supplies typed dialogs/events/quit; the renderer binds only the reviewed 59-method `DesktopAPI`. Generated TypeScript interfaces describe JSON timestamps as strings and nullable pointers/slices without weakening strict checks.

The local target is `0.2.0-rc.3`, the verification candidate for `0.2.0`. RC does not claim production acceptance. Final-version packaging is fail-closed: `scripts/release-windows.ps1 -StableEvidence <reviewed-file.json>` requires records for all checks below, matching the exact `VERSION` and clean Git revision. There is no force/skip-stability switch. Passing an attestation validator validates its shape and revision, not the truth of external reports; maintainers must inspect actual evidence.

## Acceptance records

| Record key | Required evidence |
| --- | --- |
| windows10-clean | Clean Windows 10 x64, normal user, OS/build and installed/extracted app flows |
| windows11-clean | Clean Windows 11 x64, normal user, installed/extracted app flows |
| webview2-missing | Missing-runtime explanation and successful subsequent launch after explicit runtime setup |
| native-media-workflows | Real native import/save dialogs, Unicode paths, compression/trim/audio/subtitle/project and no-overwrite/error flows |
| native-recovery-exit | Dirty edits, close confirmation, interruption/restart and explicit draft recovery; no automatic job execution |
| native-download-cancel-retry | Incomplete environment, explicit setup, progress, cancellation/retry and ready-environment guard |
| cpu-only | Actual CPU-only environment and cancellation/fallback behavior |
| gpu | Actual qualified GPU encoding, output verification and cancellation |
| installer-upgrade | Previous-version upgrade/reinstall/uninstall, payload/digest/version checks, running guard and user-data preservation |
| hosted-ci | Hosted Windows checks for the exact source revision; use workflow_dispatch checks_only before final packaging |
| source-and-assets-review | Public source/history/assets/license review, repository/contact and matching corresponding-source plan |

Evidence JSON has `schemaVersion: 1`, `version`, `sourceRevision` and `checks`. Each named check must contain `passed: true`, a meaningful `evidence` reference, `verifiedBy` and `verifiedOn` (YYYY-MM-DD). Do not fill unperformed checks with success. Keep private screenshots, source paths, logs and diagnostic attachments out of public records. A final-version workflow dispatch accepts the reviewed JSON via its `stable_evidence` input; candidate packages need no stable attestation and stay explicitly prerelease.

Signing is a separate owner decision, not an automatic mark of functional stability. Candidate assets are unsigned. The owner subsequently authorized public repository creation and source push to [u-fw/PheebKit](https://github.com/u-fw/PheebKit); private vulnerability reporting is enabled. Stable binary publication still requires the acceptance records above.

## Local evidence — 2026-10-18

The stable Wails build boots on the current Windows 11 Pro build 26300 workstation. Native multi-file dialog opens/cancels and imports a generated `测试 & clip.mp4`; the real renderer → Desktop IPC → service → qualified RTX 4070 NVENC workflow completes an eight-second HEVC/AAC job. The queue receives progress/completion and postflight structure verification. An independent FFmpeg full decode succeeds with duration 8.000 seconds. Native maximise/restore synchronizes the custom titlebar state. These observations do not establish clean-machine, Windows 10 or CPU-only compatibility.

The migration passes 120 frontend tests and the uncached local Go media integration suite. Dedicated regressions cover the 59-method renderer boundary, missing remembered dialog directories, direct Wails v2 event payloads, drop listener cleanup, and rejection of absent/failed/stale stable evidence. Final gate/package and further native results are recorded in the release acceptance and distribution documents when performed.

The first hosted run, [39101680862](https://github.com/u-fw/PheebKit/actions/runs/39101680862), exposed three PATH-dependent integration tests and a same-size source edit missed with coarse filesystem change times. The rc.2 fixes provision the PATH tests from the verified tools without modifying system PATH, and fully hash sources up to 4 MiB. Larger sources retain bounded sampling and its documented limits. The candidate also adds a side-effect-free `PheebKit.exe --build-info` query; local production execution confirms the actual linked version, revision, modified status and framework. Packaging and installer checks compare this response with the payload manifest. rc.1 is retained as a superseded local artifact.

Final binary auditing additionally found five known HTTP/2 advisories in the Wails production-tag dependency graph (`golang.org/x/net v0.56.0`), which ordinary untagged source analysis did not cover. The minimum official fixed version is v0.60.0; its verified module requirements raise the Go language baseline to 1.26 and update the related x/crypto, x/sys and x/text dependencies. Release builds still require Go 1.27.2+. Both packaging and checks-only CI now audit the actual built EXE. The unpublished pre-fix rc.2 package is archived outside release outputs; no vulnerable binary is published as a release.

The following hosted run reached real-media tests and isolated a copied-tool startup failure in the shared-configuration case. Validation now preserves context cancellation/deadline causes and includes bounded stderr, rather than hiding the reason behind an exit code. The finite environment validation budget is 30 seconds for cold launches. Focused real shared-configuration regressions pass locally. User-visible legacy brand messages are corrected while compatibility identifiers remain unchanged. The [README screenshots](images/README.md) show the actual Windows application and self-generated test media; rc.3 adds the actual [runtime language selector](i18n.md).

## rc.3 naming and language evidence — 2026-10-18

Public Go/env/build names now use PheebKit; read-only legacy aliases and persisted identity bytes are isolated and regression-tested. The shared catalog currently has 1,223 entries. All 126 frontend tests, source coverage/placeholder/namespace guards, TypeScript, lint, real uncached FFmpeg Go integration and vet pass locally. Native language switching, English after restart, English picker title/filter, qualified GPU encode and independent complete output decode pass on the existing Windows 11 workstation. Current [captures](images/README.md) describe their exact scope. The application-owned English close explanation and composed range/track hints found during native testing were corrected before packaging. Clean-client and CPU-only acceptance records remain open.
