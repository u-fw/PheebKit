# FFmpeg runner contract

Core owns `Runner.Probe`, `Capabilities`, `Start` and `Cancel`. Execution plans contain argv and `Source` references, never an executable shell string. The application invokes runners from background bridge calls and the job worker; FFmpeg does not block React.

Local source fingerprints are checked around probe, before preview, before queued execution and before output publication. Detected changes require explicit re-import; an old plan does not silently reuse stale stream indices, HDR status or size budgets. Files up to 4 MiB are hashed in full. Larger files combine identity/change timestamps with 192 KiB of content samples; unsampled changes on filesystems with weak timestamps are outside this detection guarantee. This is not a file lock. Historical references retain their original fingerprint semantics; new probes use v2 hashes.

## Windows

The adapter starts `exec.CommandContext(binary, args...)`; no `cmd.exe /c`, PowerShell invocation or shell concatenation exists in the execution path. Preview is a separately escaped PowerShell display command using the discovered executable path. Discovery prefers `PHEEBKIT_FFMPEG`/`PHEEBKIT_FFPROBE`, then `FFMPEG_PATH`/`FFPROBE_PATH`, saved application paths and PATH. Recognized Scoop shims resolve to the actual executable so cancellation owns FFmpeg rather than only a launcher. Environment status validates both binaries with bounded `-version` calls and supplies a configuration prompt if either is missing. Saved configuration is updated atomically; the queue's stable runner proxy allows idle reconfiguration without losing job history. Queued/running jobs block path changes.

Probe uses `-v error -show_format -show_streams -show_chapters -show_programs -of json`. Go decodes JSON and derives rational frame rate, bit depth and common HDR signaling. ffprobe exposes other structures but the first Media Card projects stream and format data only.

Capability scanning reads the installed build's version, configuration, encoder list and hardware backends. Native `EnumDisplayDevicesW` supplies adapter names. NVENC, AMF and QSV candidates present in the build are then runtime-qualified with real synthetic-frame encoding trials using the compiler's quality/bitrate parameter policy. Trials have bounded concurrency and timeouts, and test 8-bit plus HEVC/AV1 10-bit input. Availability records include tested pixel formats and failure reasons; a build wrapper alone is never treated as a usable GPU. New `encoder: auto` intent prefers qualified GPU candidates, then CPU; `cpu` is explicit software. Legacy empty selectors preserve CPU. Capability refresh repeats the trials.

CPU uses existing CRF/preset options. NVENC uses CQ/VBR with p3/p5/p7, QSV ICQ with `global_quality`, and AMF CQP/VBR with vendor presets. Numeric quality values are not equivalent visual-quality measurements. Only hardware actually present can be verified here; unsuccessful vendor trials remain unavailable rather than claiming support from package listings.

Thumbnails use the Windows Shell's `IShellItemImageFactory` first, retaining system thumbnail-provider selection similar to Explorer. Calls run on a locked COM worker thread with bounded callers/workers and proper interface/GDI cleanup. FFmpeg fallback maps an attached picture or the first motion video stream and emits a bounded local image. Thumbnail requests require a previously probed source; failure never substitutes invented artwork or blocks a compression job.

Encoding uses `-progress pipe:1 -nostats`, parses key/value packets on stdout and collects stderr separately. The job engine throttles progress to 150 ms, log events to 250 ms and keeps bounded logs. Terminal state comes from process completion, not a `progress=end` packet. Percent is 0–100 in Go and normalized to 0–1 by the UI bridge.

The compiler maps the first motion video by its absolute stream index and optional audio streams. Attached cover art and timed thumbnails are excluded from motion encoding. Subtitles and data streams are not transferred by the compression spike. AAC/Opus encoding or audio copy is validated against the container. Explicit resolution choices are honored, including upscaling. Target size uses decimal MB and a one-pass bitrate budget with 2% muxing allowance; it is an estimate, not an exact size guarantee. If audio exists, size mode requires AAC/Opus because Copy does not provide a known audio budget in this model. The UI explains this at the audio control and beside Start, with an explicit AAC correction action. HDR can be probed and displayed, but compression rejects HDR until tone mapping or metadata preservation has been verified.

Existing output paths and identical input/output paths are refused. Encoding writes to an owned `.pheebkit-*` directory adjacent to the requested output. On success, Windows `MoveFile` publishes the result without replacing an existing destination, even if another process created it during encoding. Cancellation/failure removes only the runner's temporary directory; the requested output remains absent and can be retried. Preview shows the logical output path; the adapter substitutes the temporary path internally and preserves every other argv element.

Before publication, beta.5 re-probes the temporary output with a cancellable 30-second timeout and bounded response capture. Compiler intent supplies expected motion-video/audio/subtitle counts and compression codec; invalid/unknown codecs, invalid video geometry or missing/unexpected tracks fail the task before publication. Duration differences and unknown audio/video duration produce review warnings, not false claims of corruption. Result history records observed structure; legacy records have no verification badge. This metadata check is not a complete decode scan; [workflow policy and evidence](workflows.md) describe the limits.

Cancel interrupts the actual process context. Process pause/resume is not supported in this spike. Android has no executable process fallback; its mock reports unsupported and never emits fabricated media or success.

## Advanced parameters

The GUI sends raw arguments as one argv element per line, up to 32 option/value pairs. The spike accepts only `-profile:v`, `-level:v`, `-g`, `-bf`, `-maxrate`, `-bufsize`, `-x264-params`, `-x265-params`, `-svtav1-params`, and `-metadata`. Structural process, input, output and progress switches are protected, so this escape hatch cannot add a second input, change the managed output or redirect progress. The full unrestricted Raw Command workspace remains a later milestone. Shell punctuation stays within argv data.

Codec-private parameter strings additionally require exact `key=value` entries from an encoder-specific typed allowlist. Unknown keys, file paths, auxiliary input/output options and multi-pass side files are rejected before execution; FFmpeg's `-n` alone does not protect those codec-managed files. Safe examples include `aq-mode=3` for x264, `log-level=error` for x265 and `tune=0` for SVT-AV1. The supported table lives in `internal/compiler/codec_parameters.go`.

## Verification

Run the Windows integration check from PowerShell:

```powershell
$env:PHEEBKIT_INTEGRATION = '1'
go test ./internal/runner/windows -v
```

The check creates its own synthetic media, probes actual JSON, compiles and executes HEVC, verifies the encoded output and checks cancellation. Core unit tests exercise invalid projects, unavailable encoders, URI rejection, argv boundaries, progress parsing and queue transitions. See `verification.md` for recorded results and limitations.
