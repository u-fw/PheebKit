# FFmpeg runner contract

Core owns `Runner.Probe`, `Capabilities`, `Start` and `Cancel`. Execution plans contain argv and `Source` references, never an executable shell string. The application invokes runners from background bridge calls and the job worker; FFmpeg does not block React.

## Windows

The adapter starts `exec.CommandContext(binary, args...)`; no `cmd.exe /c`, PowerShell invocation or shell concatenation exists in the execution path. Preview is a separately escaped PowerShell display command. Both `KOMPEG_FFMPEG` and `KOMPEG_FFPROBE` can point to real binaries. Defaults search PATH; recognized Scoop shims resolve to the actual executable so cancellation owns FFmpeg rather than only a launcher.

Probe uses `-v error -show_format -show_streams -show_chapters -show_programs -of json`. Go decodes JSON and derives rational frame rate, bit depth and common HDR signaling. ffprobe exposes other structures but the first Media Card projects stream and format data only.

Capability scanning reads the installed build's version, configuration, encoder list and hardware backends. An encoder registry resolves supported CPU families (H.264, HEVC and AV1) to real available names. Other component scanners and hardware execution qualification belong to later scope.

Encoding uses `-progress pipe:1 -nostats`, parses key/value packets on stdout and collects stderr separately. The job engine throttles progress to 150 ms, log events to 250 ms and keeps bounded logs. Terminal state comes from process completion, not a `progress=end` packet. Percent is 0–100 in Go and normalized to 0–1 by the UI bridge.

The compiler maps the first motion video by its absolute stream index and optional audio streams. Attached cover art and timed thumbnails are excluded. Subtitles and data streams are not transferred by the compression spike. AAC/Opus encoding or audio copy is validated against the container. Explicit resolution choices are honored, including upscaling. Target size uses decimal MB and a one-pass bitrate budget with 2% muxing allowance; it is an estimate, not an exact size guarantee. HDR can be probed and displayed, but compression rejects HDR until tone mapping or metadata preservation has been verified.

Existing output paths and identical input/output paths are refused. Encoding writes to an owned `.kompeg-*` directory adjacent to the requested output. On success, Windows `MoveFile` publishes the result without replacing an existing destination, even if another process created it during encoding. Cancellation/failure removes only the runner's temporary directory; the requested output remains absent and can be retried. Preview shows the logical output path; the adapter substitutes the temporary path internally and preserves every other argv element.

Cancel interrupts the actual process context. Process pause/resume is not supported in this spike. Android has no executable process fallback; its mock reports unsupported and never emits fabricated media or success.

## Advanced parameters

The GUI sends raw arguments as one argv element per line, up to 32 option/value pairs. The spike accepts only `-profile:v`, `-level:v`, `-g`, `-bf`, `-maxrate`, `-bufsize`, `-x264-params`, `-x265-params`, `-svtav1-params`, and `-metadata`. Structural process, input, output and progress switches are protected, so this escape hatch cannot add a second input, change the managed output or redirect progress. The full unrestricted Raw Command workspace remains a later milestone. Shell punctuation stays within argv data.

## Verification

Run the Windows integration check from PowerShell:

```powershell
$env:KOMPEG_INTEGRATION = '1'
go test ./internal/runner/windows -v
```

The check creates its own synthetic media, probes actual JSON, compiles and executes HEVC, verifies the encoded output and checks cancellation. Core unit tests exercise invalid projects, unavailable encoders, URI rejection, argv boundaries, progress parsing and queue transitions. See `verification.md` for recorded results and limitations.
