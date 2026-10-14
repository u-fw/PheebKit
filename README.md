# 菲比工具箱 · PheebKit

A Windows 10/11 x64 video-processing experimental application using Go Core, Wails v3, React and strict TypeScript. Windows uses real installed FFmpeg binaries. Android work is deferred; its unsupported mock/native-shell research remains in source and no APK is released.

Wails `v3.0.0-beta.27` and its experimental Android shell were explicitly approved for this spike. This repository is not a production release. [Environment audit](docs/environment.md) and [Android blockers](docs/android.md) distinguish installed tools, source support and actual runtime evidence.

Current branding is 菲比工具箱 / PheebKit. Historical screenshots in the verification archive show earlier kompeg builds.

Beta.5 adds output structure verification before publication, visible duration warnings, a failed/interrupted queue filter and explicit review before retry. Missing `.ffproject` sources can be relocated without rewriting the saved project; the recovered editor remains unsaved until an explicit save. See [workflow checks and limits](docs/workflows.md).

## Windows development

Requires Go 1.25+ with automatic selection of the pinned Go 1.27.2 security toolchain, Node 24+, WebView2 and FFmpeg/ffprobe with a supported software or usable hardware encoder. Release builds enforce Go 1.27.2 or newer and dependency vulnerability checks. See the [security/performance audit](docs/security-performance-audit.md). Install the exact CLI once:

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.27
npm --prefix frontend ci
wails3 generate bindings -ts -i -clean=true
npm --prefix frontend run build
wails3 build
.\bin\PheebKit.exe
```

`wails3 build` generates the Windows manifest/resource and embeds the production frontend. The manifest enables long paths; filesystem policy and the installed FFmpeg build still determine complete long-path support. For optional explicit binary selection:

```powershell
$env:KOMPEG_FFMPEG = 'D:\Tools\ffmpeg\bin\ffmpeg.exe'
$env:KOMPEG_FFPROBE = 'D:\Tools\ffmpeg\bin\ffprobe.exe'
```

Use `wails3 dev -config build/config.yml` for frontend development. Ordinary browser previews cannot execute native operations. Drag media files onto the workspace or use the multi-file picker (up to 32 files), review media details, configure processing, inspect the backend command preview and follow progress/results in Queue. Batch submission uses shared settings, independent track selections and distinct output names; all projects must pass preflight before any are enqueued.

FFmpeg discovery prefers `KOMPEG_FFMPEG` / `KOMPEG_FFPROBE`, then `FFMPEG_PATH` / `FFPROBE_PATH`, then application settings and PATH. Values can identify the executable or its directory. Missing/invalid binaries show a setup prompt; Settings lets you download a verified Windows x64 package or select `ffmpeg.exe` and verifies its sibling `ffprobe.exe`. Paths are saved in `%APPDATA%/kompeg/ffmpeg.json`, without modifying Windows environment variables. Explicit environment overrides must be corrected in the environment and require restarting the app. Reconfiguration is blocked while jobs are active. One-click downloads are disabled when both tools are already usable; select a local path to deliberately switch versions. Explicit overrides must be removed/corrected before downloads or path writes.

One-click setup offers Gyan 9.0.2 full stable builds and the older npmmirror 6.1.1 domestic fallback. The default tries gyan.dev first, then the matching GitHub release. Domestic fallback must be selected manually after a failure; it never silently downgrades. Versions, transfer progress, cancellation and retry are visible. Downloads use fixed SHA-256 digests, preserve upstream notices, and install privately under `%LOCALAPPDATA%/kompeg/tools`. See [download policy and verification](docs/ffmpeg-downloads.md).

Settings also accepts a custom HTTPS archive download link. Mirrors of the exact built-in Gyan full ZIP retain its pinned digest. Other Windows x64 builds require a publisher-provided SHA-256 and explicit source trust; PheebKit automatically finds a single FFmpeg/ffprobe pair, preserves sidecar DLLs, validates and configures it. ZIP, 7z, RAR, tar, tar.gz/tgz, tar.xz and tar.bz2/tbz2 are detected by file contents without requiring an external extractor. Download pages, source-only/Linux builds, encrypted or split archives and self-extracting executables are unsupported.

New compression settings default to GPU preference. NVENC, QSV and AMF are offered only after bounded real encoding trials; unavailable combinations fall back to CPU with a visible reason. Source artwork prefers Windows thumbnail providers and falls back to FFmpeg cover/frame extraction.

Compression defaults to the complete source duration. Select **剪辑并压缩** to retain one start/end range using Material hour/minute/second selectors, 0.1-second adjustment buttons or keyboard-accessible sliders. The same FFmpeg invocation trims and encodes once; progress, speed and size-mode bitrate budget use the selected duration. Audio uses AAC/Opus for accurate boundaries (Copy changes to AAC when enabling trim). See [trimming behavior and verification](docs/trimming.md).

Windows packaging produces an installer, a standalone ZIP and SHA-256 checksums from the same build. `VERSION` is the version source; Settings, EXE resources and package metadata stay synchronized. See [Windows distribution and verification](docs/windows-distribution.md) for packaging commands, local lifecycle checks and remaining clean-machine/signing checks. Publication remains a separate owner action.

## Task workflow and local preferences

Closing the window (including Alt+F4) checks the native job queue and FFmpeg setup. Active work opens a confirmation dialog; returning preserves work, while explicit exit cancels processing. Completed results offer Open File and Show in Folder; missing/moved files produce an actionable error. Reuse Settings re-probes the source, restores the editable intent and suggests a new output filename, without executing a saved command.

Trim preview offers source playback, selected-range playback, a seek slider, current-position boundary buttons and FFmpeg boundary frames. WebView2 codec support controls playback; unsupported formats retain FFmpeg position-frame navigation. Preview is a positioning aid rather than frame-exact edit playback.

The app saves recent job history (normally 200 records) and up to 20 named processing configurations under the existing `%APPDATA%/kompeg` directory. Interrupted jobs do not auto-resume. Defaults/presets exclude source/output paths, trim, specific track indexes and crop margins; retained history contains local file paths and diagnostics. Atomic JSON writes use a cross-process file lock. See [task workflow and verification](docs/workflows.md).

## Media tools — 0.2.0-beta.1

- Compression selects audio/subtitle tracks, mutes audio, retains compatible subtitles, crops even pixel margins, rotates/flips video and adjusts encoded audio volume. Crop coordinates follow the source's displayed orientation; the preview remains the original, and adjustments apply to output.
- Remux copies the primary motion-video stream and selected compatible audio/subtitle streams to MP4/MKV/WebM. It does not encode, trim or apply filters. Unsupported container/codec combinations fail before processing.
- Audio extraction saves one selected track as AAC/M4A, Opus, MP3, FLAC, WAV or a compatible copy in Matroska Audio. Text subtitle extraction supports SRT/ASS/WebVTT; OCR and subtitle burn-in are outside this release.
- Batch processing uses full source durations and excludes the active single-file crop/trim. Track choices are reviewed per file. Processing remains serial; waiting jobs can be reordered, and pausing scheduling leaves the current job running. Remaining time is an estimate from live speed.
- Short sample compression uses 10/20/30 seconds from the current range start, within that range. Samples are distinct output files and can be compared at corresponding positions in Queue. Copy audio becomes AAC for the sample; size budgets scale with duration. The 512px comparison is a positioning aid, not a full-resolution quality metric.
- Save/open `.ffproject` files and import/export `.pheebpreset` configurations through native dialogs. Opening/importing never executes tasks; sources are re-probed, existing output targets require a new choice, and duplicate imported preset names are rejected. See [processing policies and acceptance](docs/media-tools.md).

## Verification commands

```powershell
go test ./...
go vet ./...
$env:KOMPEG_INTEGRATION = '1'
go test ./internal/runner/windows -v
Remove-Item Env:\KOMPEG_INTEGRATION
npm --prefix frontend run test
npm --prefix frontend run typecheck
npm --prefix frontend run lint
npm --prefix frontend run format:check
wails3 build
```

`go run ./cmd/spike -input <absolute-path> -output <new-absolute-path.mp4>` verifies the same probe/compiler/runner slice without the UI and prints actual media/result JSON.

## Structure

```text
cmd/spike/                  real Windows headless verification command
internal/app/               Wails service, events, platform factories
internal/core/              shared Media, Project IR, plan, job and Runner
internal/probe/              ffprobe JSON decoder
internal/capabilities/       dynamic encoder registry
internal/compiler/          validation, argv and command preview
internal/project/           schema-versioned project persistence
internal/presets/           versioned preset definitions
internal/jobs/              independent job records and serial worker queue
internal/runner/windows/    real process, progress/log parsing and cancellation
internal/runner/android/    explicitly unsupported mock/native boundary
frontend/                   shared React UI, tokens and generated TS bindings
platform/android/           JNI/SAF native integration plan
platform/windows/           Windows adapter and distribution notes
build/windows/              manifest, icon and executable metadata
build/android/              pinned upstream task reference; toolchain required
docs/                       architecture, design and verification evidence
```

See [architecture](docs/architecture.md), [UI design](docs/ui-design.md), [runner](docs/ffmpeg-runner.md), [project format](docs/project-format.md), [Android FFmpeg spike](docs/android-ffmpeg-spike.md) and [actual verification](docs/verification.md).

The [open-source readiness review](docs/open-source-readiness.md) records repaired defects, verification and remaining publication decisions.

## License

PheebKit's original code is licensed under the GNU General Public License version 3 only (`GPL-3.0-only`). See [LICENSE](LICENSE) for the full terms. Third-party dependencies and assets retain their respective licenses. FFmpeg is currently discovered locally rather than bundled; packaging or downloading FFmpeg does not replace its own license or distribution obligations.

## Current scope

Home, Media Card and thumbnails, environment setup, qualified GPU/CPU selection, SDR compression with single-range trimming/preview and basic transforms, selected tracks, compatible remux/audio/text-subtitle extraction, batch submission and queue scheduling, sample comparison, project-file management, preset sharing, limited advanced argv, exit protection and persisted history form the current Windows slice. Multi-clip editing, HDR encoding, unrestricted Raw Command, all component scanners and FFmpeg process pause/resume remain later milestones. Windows FFmpeg is discovered rather than bundled; third-party redistribution needs its own source/notice review.

Android SDK/NDK/emulator and a connected device are absent on the audited machine. The pinned Android task assets also require a POSIX build host. APK build, Android Go binding and native FFmpeg execution are therefore not verified; responsive UI reuse alone does not satisfy those criteria.

The current brand is 菲比工具箱 / PheebKit. Settings in the legacy kompeg AppData directory, KOMPEG_* environment overrides, project schema and installer AppId remain compatible. Release files use PheebKit; earlier kompeg packages are historical candidates. The supplied brand image is preserved in frontend/public/pheebkit.png and converted to Windows icons with the pinned Wails CLI.
