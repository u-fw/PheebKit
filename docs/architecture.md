# FFmpeg Studio architecture

The first milestone is a technical spike, not the complete product. Targets are Windows 10/11 and Android. Wails v3 Beta and experimental Android support were explicitly approved for this spike on 2026-10-09; they are not a production stability claim.

```mermaid
flowchart TD
  UI[Shared React / TypeScript UI] --> Bridge[Wails Bridge]
  Bridge --> Services[Application Services]
  Services --> Core[Go Core: versioned Project IR / media / validation / jobs]
  Core --> Compiler[FFmpeg Execution Plan: argv + source references]
  Compiler --> Runner[Core Runner interface]
  Runner --> Windows[Windows: ffmpeg.exe / ffprobe.exe via os/exec]
  Runner --> Android[Android Adapter: native FFmpeg + SAF]
```

## Boundaries

- `internal/core` owns the shared models and runner interface. It imports neither Wails nor platform process APIs.
- `internal/probe` parses ffprobe JSON. `internal/capabilities` parses actual FFmpeg encoder capabilities.
- `internal/compiler` validates Project IR and produces argv. Command preview is display-only and is never executed through a shell.
- `internal/jobs` owns a queue of independent job records, cancellation, bounded logs and throttled progress events.
- `internal/runner/windows` owns process execution, stdout progress, stderr logs and Windows process options.
- `internal/runner/android` is an explicitly unsupported adapter during the first spike. No fabricated compression results are returned.
- `internal/app` connects services and events to Wails; platform factories use Go build constraints.
- `frontend` contains the shared Media Card, compression form and queue. The bridge converts generated bindings into the UI API.
- `platform/android` documents the native JNI/SAF boundary. It may replace the shell without rewriting Core.

## Media and execution

`Source` distinguishes local filesystem paths from `content://` document URIs. The original reference belongs in Project IR; temporary native SAF handles belong only in the runner. UI changes Project IR, asks Go to compile a plan, and starts that same plan through the job engine.

The Windows spike uses installed software encoders discovered from FFmpeg. Listing an NVENC encoder does not prove that a GPU can execute it; automatic hardware selection needs a later runtime test. H.265 software encoding is the first verified slice.

## Persistence and compatibility

`.ffproject` JSON and presets carry `schemaVersion: 1`. Unknown versions fail explicitly. Raw arguments are separate argv elements and are an escape hatch, not the persisted primary model. The first spike does not reverse-parse edited commands into GUI state.

## Verification limits

Windows gets real process, probe, encode and cancel tests. Android execution and Go binding cannot be called verified without an APK build and device/emulator run. A responsive browser viewport alone verifies layout sharing, not an Android native runtime.
