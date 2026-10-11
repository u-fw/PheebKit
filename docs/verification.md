# PheebKit technical spike verification

Current 0.2.0-beta.1 feature evidence/limits are in [media tools](media-tools.md); beta.4 workflows remain in [workflows](workflows.md). Entries below are dated historical evidence; earlier kompeg branding, binaries, screenshots and test counts describe earlier builds.

Recorded on 2026-10-09, native Windows 11 build 26200, PowerShell 7.6.5. Wails v3.0.0-beta.27 and experimental Android were explicitly approved for this spike. Windows 10 is a target, but the actual runtime evidence below was collected on Windows 11.

## Acceptance status

| Technical Spike criterion | Status | Evidence |
| --- | --- | --- |
| Windows Wails window starts; React calls Go | VERIFIED | Actual production WebView2 window; generated `GetPlatform()` binding returns `windows` |
| Android APK starts; React calls Go | BLOCKED BY MISSING REQUIRED INPUT | No SDK/NDK/emulator/device on Windows or audited Debian WSL; pinned tasks also need a supported POSIX build host and full staged Android assets |
| `GetPlatform()` on both platforms | Windows VERIFIED; Android IMPLEMENTED BUT EXTERNALLY UNVERIFIED | Shared service uses `runtime.GOOS`; no Android device call claimed |
| Runner interface; real Windows and Android mock | VERIFIED | Shared interface, build-constrained factories, real runner tests; Android mock returns unsupported |
| Video import → ffprobe JSON → Media Card | VERIFIED through native dropped-file pipeline | Chinese/space-containing source path, native IPC, real ffprobe, generated Go models and actual card; file picker API implemented, dialog selection not automated |
| Real Windows HEVC compression | VERIFIED | Go CLI and native GUI runs; outputs independently inspected with ffprobe |
| Progress/FPS/speed/out time/cancel | VERIFIED | Native GUI progress and cancel, runner integration; cancellation leaves no final output or owned staging directory |
| Shared Media Card/Compress on Android | Responsive source VERIFIED; APK rendering BLOCKED | Same components at phone viewport; this does not establish Android runtime behavior |

The Windows vertical slice is verified. The complete cross-platform Technical Spike remains open until the Android shell criteria pass. Native Android FFmpeg integration is still phase-gated behind that proof.

## Commands and checks

- `go test ./...` with the real integration opt-in enabled: passed.
- `go vet ./...`: passed. `gofmt -l` for project Go files: empty.
- Frontend Vitest, TypeScript, ESLint and Prettier checks: passed. Production Vite build: passed.
- `wails3 build`: actual Windows production executable build passed.
- `wails3 task build:dev` and `wails3 dev -config build/config.yml`: passed. The native development window loaded `@vite/client` and source modules, and an actual reversible source edit appeared through HMR before being restored.
- Android arm64 shared-package check (`GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build` for Core/compiler/probe/project/jobs/capabilities/presets/Android runner): passed. This excludes Wails cgo/JNI.
- `wails3 android overlay:gen -out artifacts/android/overlay.json -config build/config.yml`: passed; no APK implied.
- Android native task attempt: failed with exit 127 (`uname` / `tail` unavailable); SDK/NDK also missing. No APK produced.
- `git diff --check`: passed.

The integration switch is now `KOMPEG_INTEGRATION=1`. Earlier evidence was collected before the brand rename with its previous prefix; the rename was rechecked with the current prefix. Race tests were not run on Windows because a usable cgo C compiler was absent. The WSL GCC installation does not supply a Windows compiler.

## Real compression measurements

A synthetic 8-second, 640×360, 30 FPS H.264/AAC clip was created under `artifacts/媒体 测试/源视频 sample.mp4`. It is a functional fixture, not a representative quality/size benchmark.

| Run | Input bytes | Output bytes | Saved | Result |
| --- | ---: | ---: | ---: | --- |
| Shared Go CLI, HEVC + AAC | 1,694,077 | 804,548 | 52.508% | HEVC, 640×360, 30 FPS, 8 seconds; 0.8203 seconds elapsed, 9.75× average |
| Native GUI, HEVC + AAC | 1,694,077 | 813,367 | 51.988% | Independently verified HEVC/AAC, 640×360, 30 FPS, 8 seconds |

The native queue automatically reached `completed` without a manual refresh. A discovered React batching bug was fixed by snapshotting event updates before clearing the pending map and preventing a late queued Start response from overwriting newer event state.

A 180-second 1280×720 fixture was used for native cancel verification. The running window displayed 135 FPS, 4.47× speed and out time 00:00:11 before cancellation. Clicking Cancel transitioned through `cancelling` to `cancelled`; the requested `GUI native-cancel.mp4` and `.kompeg-*` staging directories were absent afterward. Earlier pre-rename evidence uses `.ffstudio-*`. Runner integration also retries the same output path after cancellation and protects against a destination created by another process.

## UI evidence

Actual WebView2 evidence is stored locally under the ignored `artifacts` directory. These are generated verification artifacts, not bundled product sample data:

- `windows-media.png`, `windows-compress.png`, `windows-queue-completed.png`: real media/configuration/result.
- `windows-queue-running.png`, `windows-queue-cancelled.png`: real progress and cancellation.
- `native-completed-events.json`, `native-cancel-events.json`: captured native job events.
- `windows-compression-result.json`: CLI input/output/result JSON.
- `windows-dev-hmr.png`: native development source-update proof.

The original pre-polish queue passed axe checks in light, dark and 375×812 phone viewport with zero reported violations; dark Media Card also passed. A duplicate complementary landmark on Compress was found and fixed with a named sidebar.

The kompeg refresh was then verified in the actual production WebView2 window:

- Window/page identity, Go module, executable metadata, generated bindings, favicon and K/play icon use kompeg. Windows file metadata reads `ProductName: kompeg`, `OriginalFilename: kompeg.exe`, version 0.1.0.
- Native computed `backdrop-filter` is `blur(26px)`. A production CSS issue that dropped the standard declaration was found and corrected by ordering the vendor-prefixed declaration first.
- Custom resolution/audio menus were operated with Arrow keys, Home/End and Enter. The chosen AAC setting appeared in the Go command preview and real HEVC/AAC output. Tab closed the menu and focused Audio; Escape closed it. Native DOM checks confirmed valid `aria-controls`/active-descendant references.
- A menu initially collided with the phone bottom navigation. The corrected 375×812 layout puts it between top bar bottom 65px and bottom navigation top 735px (observed menu 73–365.375px). It has a full-width phone field and scrollable options.
- Both 375×812 portrait and 812×375 landscape use bottom navigation without horizontal overflow. At 200% root text size, 375px viewport reports 360px document scroll width (the remainder is the scrollbar); the output summary and segmented groups adapt.
- Latest desktop light Compress axe audit: 42 passes, 0 violations, 0 incomplete. Dark Compress: 42 passes, 0 violations. Phone dark: 42 passes, 0 violations, 1 incomplete contrast check for the active bottom-navigation label because of overlapping translucent rendering. The label was visually checked; this automated result is not presented as an exhaustive accessibility certification. Open-menu audits also leave obscured background text checks incomplete; menu reference existence was checked directly.
- A fresh kompeg GUI encode automatically completed: 1,694,077 → 813,367 bytes, 51.9876% saved, 0.9672 seconds elapsed, 8.27× average. ffprobe confirmed HEVC/AAC, 640×360, 8 seconds.

Current review artifacts include `kompeg-home-light.png`, `kompeg-home-dark.png`, `kompeg-compress-light.png`, `kompeg-compress-dark.png`, `kompeg-mobile-dark.png`, `kompeg-mobile-menu-dark.png`, `kompeg-large-text.png`, `kompeg-queue-completed.png`, `kompeg-native-events.json`, and `kompeg-menu-layout-proof.json`. Earlier `windows-*` captures show the pre-refresh implementation. Ten frontend tests cover format/project helpers, select semantics, disabled options, navigation wrapping and typeahead; native interaction supplements those tests.

## Scope boundaries

SDR CPU HEVC and NVIDIA NVENC are verified encode slices. Hardware evidence is specific to this RTX 4070 and FFmpeg build; AMD/Intel execution is not claimed. Target size is a one-pass estimate. HDR compression is explicitly rejected until tone mapping or metadata preservation has been verified. Probe still displays common HDR signaling.

Process pause/resume, unrestricted Raw Command, full component scanner, project/preset management UI, installer/signing and native Android encode remain outside the implemented slice. FFmpeg is installed separately. Long-path awareness and adapter publication paths are implemented, but all FFmpeg builds/filesystem policies are not exhaustively qualified.

## Windows usability follow-up

The target-size screenshot exposed a missing visible explanation: a source with audio plus Copy cannot provide the compiler's known audio budget. Native verification now shows the reason beside Audio and Start, preserves Copy until the explicit AAC action, and enables Start after choosing AAC. The display labels 100 decimal MB as approximately 95.37 binary MiB. Backend preview errors are visible beside Start, including existing-output errors.

- Native `EnumDisplayDevicesW` detects NVIDIA GeForce RTX 4070 and the GameViewer virtual adapter. H.264/HEVC/AV1 NVENC trials pass; AMF reports unavailable runtime DLL and QSV unavailable MFX sessions. These failed candidates are not selected. `artifacts/kompeg-gpu-capabilities.json` records real qualification.
- Fifteen real GPU codec/depth/rate-control combinations pass, including generated and independently probed 10-bit input. Qualification and all production encoders now force explicit scale/format conversion and strict pixel formats, preventing FFmpeg's automatic unsupported-format downgrade. A separate genuine CPU HEVC 10-bit encode with 720p resize also passes. CPU progress/cancel/retry and GPU tests pass.
- Native automatic HEVC/AAC output: 1,694,077 → 1,079,717 bytes, 36.265% saved, 0.5204 seconds, 15.37× average. ffprobe confirms HEVC/AAC, 640×360, 8 seconds. This fixture is a functionality check, not a comparative quality benchmark. GPU accelerates encoding; decode and scaling retain the CPU path.
- The native encode first exposed a Win32 publication failure for forward-slash paths. Paths now normalize separators/dot components before the extended-length prefix. Real publication tests cover forward slashes and Chinese names, retain no-overwrite behavior, and the repeated native GUI job completes with its actual file present.
- Eleven application tests cover environment precedence, missing/invalid configuration, persistence and hot replacement, active-job guard, actual rescan, Windows Shell thumbnail extraction, FFmpeg embedded-cover/motion-frame selection and bounded thumbnail concurrency/cancellation. Thumbnail work has two pipeline slots, a 12-second deadline and bounded COM workers.
- A separate native window launched with no FFmpeg in its isolated PATH shows the workspace setup prompt. Entering actual executable paths validates and saves them in an isolated test config, then loads GPU capabilities without restarting. Tests also cover explicit environment priority and actionable invalid-variable guidance. The file-picker API is implemented; native dialog selection is not automated.
- Native artwork displays a real 512-pixel video thumbnail; the attached-cover integration test independently receives `windows-shell` artwork. Home and Compress share asynchronous artwork. The extension displays MP4 rather than the MOV alias, and frame-rate text is rounded while retaining its precise tooltip.
- The Windows app uses the custom glass title bar and `kompeg` identity. Native minimize/maximize/restore/close operations pass; minimization was independently confirmed with Win32 `IsIconic`. Computed drag region is `drag` and blur is `blur(26px)`. Desktop and 375×812 dark/812×375 landscape layouts have no horizontal overflow. At 200% text size, 375px reports 365px scroll width (10px scrollbar).
- Final native Settings and dark phone Compress axe audits report 36 and 40 passes respectively, both with zero violations and zero incomplete checks. Window controls use a named group. These automated checks supplement the verified keyboard and native-window interactions.
- Fifteen frontend tests, typecheck, ESLint, Prettier, the real Wails production build, Go tests with integration enabled and `go vet` pass. Review artifacts include `kompeg-final-home.png`, `kompeg-final-compress-light.png`, `kompeg-final-mobile-dark.png`, `kompeg-size-copy-explanation.png`, `kompeg-missing-environment.png`, `kompeg-config-recovered.png`, `kompeg-gpu-queue-completed.png` and `kompeg-gpu-native-events.json`.

## Visual refinement and pre-publication fixes — 2026-10-10

- The Windows title bar is 56px; page gutters/type hierarchy and mint/lilac glass layers are consistent across screens. Desktop import uses a horizontal layout, phones stack vertically. The real production window is shown in `docs/images/workspace.png`.
- Quality validation now applies only to quality mode. Native interaction sets quality to 99, changes to bitrate mode and confirms Start is enabled with no irrelevant quality error; returning to valid quality allows a real GPU encode.
- Queue events, startup/refresh snapshots and enqueue replies share revision-aware reconciliation. Regressions cover stale running snapshots after completed events, pending batched events, overlapping refreshes and cancellation completion. Terminal history is bounded to 200 records while active jobs are retained. Native manual refresh after completion retains completed state.
- Typed positive allowlists protect x264/x265/SVT-AV1 private suboptions. Regression cases reject file access, side-file/multi-pass options, unknown keys and parsing escapes. Three real CPU encodes with safe suboptions independently probe as H.264/HEVC/AV1 at 128×96, leave only expected files and preserve a sentinel file.
- Source-fingerprint tests cover edits outside sample ranges with restored modification time, same-size replacement, mutation during probe and media replacement after plan creation. A real equal-size 1-second/2-second media replacement requires re-import; a queued changed source is rejected before any FFmpeg process/staging output. Explicit re-import updates media/thumbnail keys and restores execution.
- Two real service instances verify environment consistency: external configuration changes preserve the old instance's actual displayed paths, pending jobs block synchronization, idle rescan reloads validated paths, and invalid shared config retains the prior working runner. Frontend rescan reads actual environment after both successful and failed scans; five asynchronous tests cover ordering, failure and generation guards.
- Go tests with `KOMPEG_INTEGRATION=1`, `go vet`, gofmt, frontend 38 tests, TypeScript, ESLint, Prettier, the pinned Wails Windows production build and `git diff --check` pass. Shared Android arm64 packages, including source fingerprints, cross-build successfully; no APK is implied.
- Final native Home audit: 33 passes; light and dark-phone Compress: 40 passes each. All three report zero violations and zero incomplete checks. Dark 375×812, 812×375 landscape and 200% root text have no horizontal overflow. Reduced-motion emulation confirms `animation-name: none`; the native drag region and 26px blur remain active.
- Final native NVENC HEVC/AAC encode produces 1,079,717 bytes from 1,694,077 bytes (36.265% saved); ffprobe independently confirms HEVC/AAC, 640×360, 8 seconds. This synthetic clip checks functionality, not quality/size performance across videos.
- `npm audit` reports zero known vulnerabilities for the pinned lockfile. A basic tracked-file/common-key-pattern scan finds no matching credential files or key strings; this is not an exhaustive security certification. No dependencies were upgraded.

Artifacts include `kompeg-refined-compress-light.png`, `kompeg-refined-compress-dark.png`, `kompeg-refined-mobile-dark.png`, `kompeg-refined-large-text.png`, `kompeg-refined-queue-completed.png` and the corresponding axe JSON under the ignored `artifacts/` directory. The old `ffmpegstudio-native.exe` is archived under `artifacts/legacy-builds/`; `bin/` contains only the current `kompeg.exe`.

## Next milestone

Prepare a supported Android build host with full pinned Wails assets, SDK/NDK and a device or emulator; build/launch an APK and prove `GetPlatform() == android` plus Go events. Then implement the app-owned native FFmpeg/SAF adapter following [the Android FFmpeg spike](android-ffmpeg-spike.md), with arm64 probe, H.264/HEVC, progress, cancellation and output export.
