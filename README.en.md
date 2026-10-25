# PheebKit · 菲比工具箱

[简体中文](README.md) · [English](README.en.md)

[![Windows verification](https://github.com/u-fw/PheebKit/actions/workflows/windows.yml/badge.svg)](https://github.com/u-fw/PheebKit/actions/workflows/windows.yml)
[![License: GPL-3.0-only](https://img.shields.io/badge/license-GPL--3.0--only-blue)](LICENSE)

A local-first Windows media toolkit powered by FFmpeg. Compress, trim, remux and process tracks with editable parameters, actual command previews and independent queued jobs. Media stays on your computer.

Built with a Go core, **stable Wails v2.17.0**, React and strict TypeScript. Targets Windows 10/11 x64; Android is deferred. The current `0.2.0-rc.3` is a verification candidate, with no final installer release yet. See the [stabilization evidence and remaining checks](docs/stability.md).

## Real desktop screenshots

Unedited captures of the Windows 11 desktop candidate on 2026-10-18, using a self-generated eight-second H.264/AAC test video. The interface supports Simplified Chinese and English, with immediate language switching and locally saved preferences. [Capture record and verification limits](docs/images/README.md)

![Native media workspace with actual thumbnail, metadata and output summary](docs/images/desktop-workspace-rc3.png)

![Native settings with draft recovery controls and a custom switch](docs/images/desktop-settings-en-rc3.png)

The current source is iterating on a Material 3 / Material You workbench: light/dark modes, custom seed palettes, density, grouped parameters and a persistent output dock, plus looping, nominal frame steps, filmstrip thumbnails and bounded waveform previews. These are actual windows from the **2026-10-25 development build**; they have not replaced the verified rc.3 package above. [Progress and verification limits](docs/uiux-iteration.md)

![Development build: dark Material workbench and grouped parameters](docs/images/desktop-workbench-material-dev.png)

![Development build: theme seed, color mode and density controls](docs/images/desktop-appearance-material-dev.png)

## Features

- **Compression and trimming:** H.264, HEVC and AV1; quality, bitrate or target-size modes; full duration, one trimmed range, or up to 16 ordered ranges from one source, joined or exported separately. Enter times directly or use selectors and sliders.
- **GPU and CPU:** prefers GPU encoding after actual NVENC/QSV/AMF trials; falls back to CPU with a visible explanation when a combination is unavailable.
- **Picture and audio:** resolution, crop, rotation and flip; track selection, gain, loudness normalization, denoise, EQ, fades, sample rate and channel conversion.
- **Containers and subtitles:** compatible stream-copy remux; audio/text-subtitle extraction; embedded or external text subtitle burn-in with editable style and timing.
- **Jobs and projects:** batch import, reusable recipes and per-file preflight, serial scheduling, cancellation, waiting-job reordering and scheduling pause; sample comparisons, projects/presets, Undo/Redo, checkpoints and explicit recovery.
- **Damaged media:** bounded decode-window scans, reviewed deletion of failed intervals by default, or black/silent replacement to preserve duration. Recovery output must pass a complete decode. Missing content cannot be reconstructed.
- **FFmpeg setup:** reuse installed tools or explicitly download with progress, cancellation/retry and a custom HTTPS archive option requiring publisher SHA-256. Supports ZIP, 7z, RAR, tar and gzip/xz/bzip2 tar variants.

Multi-file timelines, HDR output, unrestricted commands/filter graphs, network/live capture and FFmpeg process pause/resume are outside this release. [Media policies](docs/media-tools.md) · [Audio controls](docs/audio-processing.md) · [Recovery contract](docs/media-recovery.md) · [Expansion scope](docs/ffmpeg-expansion.md)

## Installation and FFmpeg

Candidates contain Setup, a no-install ZIP and SHA-256 sums. Extract the entire ZIP before running; settings still live in AppData. Final downloads will appear in [GitHub Releases](https://github.com/u-fw/PheebKit/releases); Actions artifacts are verification candidates.

Microsoft WebView2 Runtime and usable FFmpeg/ffprobe are required. Application packages do not bundle FFmpeg. Settings can download missing tools or select local executables; downloads are disabled while both tools are already usable. The default uses the FFmpeg-recommended Gyan stable build, trying gyan.dev and then the same-version GitHub release. The older domestic fallback requires explicit selection and never silently downgrades. Custom archives require an explicit source-trust confirmation and publisher SHA-256. [Download and validation policy](docs/ffmpeg-downloads.md)

Discovery order: `PHEEBKIT_FFMPEG` / `PHEEBKIT_FFPROBE` → `FFMPEG_PATH` / `FFPROBE_PATH` → saved application settings → PATH. Values may identify an executable or its directory. Correct explicit overrides in the environment and restart the app. System PATH is never modified.

Public environment variables use `PHEEBKIT_*`. New settings live in `%APPDATA%/PheebKit`; private downloads use `%LOCALAPPDATA%/PheebKit/tools`. An isolated compatibility layer reuses existing data and old overrides without deleting files. See [namespace migration and compatibility](docs/compatibility.md). Third-party version, copyright and filename output remains unchanged.

## Development and verification

Requires Windows x64, PowerShell 7, Node 24+ and Go 1.26+ with automatic selection of the pinned Go 1.27.2 toolchain. Release builds enforce Go 1.27.2+. Use the exact Wails CLI and retain the lockfiles:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
npm --prefix frontend ci
wails generate module
node scripts/normalize-bindings.mjs
npm --prefix frontend run build
go test ./...
go vet ./...
npm --prefix frontend run test
npm --prefix frontend run typecheck
npm --prefix frontend run lint
npm --prefix frontend run format:check
./scripts/build-windows.ps1
```

Real-media tests require verified FFmpeg tools and `PHEEBKIT_INTEGRATION=1`. Packaging checks source and final-binary vulnerabilities, frontend quality and clean sources. Installer acceptance checks install/reinstall/uninstall, running guards, digests and user-data preservation. Hosted Windows Server checks do not establish Windows 10/11 client UI acceptance. [Verification steps](docs/release-acceptance.md) · [Windows distribution](docs/windows-distribution.md)

Key directories: `internal/core` defines media/projects/plans; `internal/compiler` validates and compiles arguments; `internal/runner/windows` manages actual processes; `internal/app` provides the native service boundary; `frontend` supplies the interface. [Architecture](docs/architecture.md) · [Project format](docs/project-format.md) · [UI design](docs/ui-design.md)

## Contributing and license

Original code is licensed under **GNU GPL v3 only (`GPL-3.0-only`)**. See [LICENSE](LICENSE). Third-party dependencies/assets retain their own licenses. Downloading or distributing FFmpeg does not replace its own license, source and notice obligations.

[Contributing](CONTRIBUTING.md) · [Private security reporting](SECURITY.md) · [Release notes](RELEASE_NOTES.md) · [Issues](https://github.com/u-fw/PheebKit/issues)
