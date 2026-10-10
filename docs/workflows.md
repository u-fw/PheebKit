# PheebKit task workflows — 0.1.0-beta.4

## Behavior

1. Native window closing and application quitting check active backend jobs and environment setup. Active work opens a styled modal with Return and Cancel Work and Exit. Esc returns to the workspace, restoring trigger focus. Confirmation cancels the service contexts and waits for the job engine to finish before quitting. Idle windows close directly. Force-kill and OS shutdown are not interactive confirmation flows.
2. Completed jobs offer Open File and Show in Folder through Windows file associations/Explorer. The backend accepts only a known completed job's MP4/MKV/WebM output, checks that the file still exists and never interpolates a shell command. Reuse Settings is available for terminal jobs retaining Project intent. It re-probes the original media and returns to the compression form with a distinct suggested output path; the user starts a new task after reviewing settings. Technical errors remain available behind a readable explanation.
3. Trimming provides original-video and selected-range playback, keyboard-accessible location seeking, current-position boundary buttons and asynchronous boundary frames. The imported source is exposed through an opaque session URL with GET/HEAD/range support. Changed sources and unknown tokens are refused. Playback depends on WebView2's codec/container support; errors switch to FFmpeg position frames without requiring source transcoding. Frames are bounded to 512px and 2MiB, share two worker slots and a 12-second timeout. Boundary extraction is debounced and stale replies are ignored. Navigating away pauses playback. Playback stopping uses browser time updates and is a locating aid, not frame-exact playback.
4. Local defaults and up to 20 named presets save encoding, resolution, audio and advanced argv settings. Trim/source/output are excluded. Same-name saves update, and presets can be applied/deleted. Persisted settings pass ordinary frontend/backend validation when used; unavailable GPU choices are not fabricated. History retains editable settings, paths, status and bounded logs, normally the newest 200 jobs. Live jobs from other processes are preserved even if that temporarily exceeds 200. Active records belonging to dead process IDs are shown as interrupted and never auto-executed. Reusing a task compiles new arguments from current media/capabilities.

## Storage

`%APPDATA%/kompeg/preferences.json` and `history.json` use version 1 and retain compatibility with existing FFmpeg configuration. Updates hold an OS file lock across read/merge/write, write a temporary file, flush it and atomically replace the owned JSON file. Reads/writes have a 24MiB bound. Malformed/unknown-version files are preserved, with a visible error rather than silent resetting. Local history contains file paths and diagnostics; it is not uploaded. Uninstall keeps AppData.

History is persisted at state transitions, including queued/running/completed/failed/cancelled. Progress is live in memory; an interrupted record can therefore show only the last persisted transition's progress. PID liveness conservatively excludes another running process's tasks from the current instance; PID reuse can delay classification until that process exits. This is restartable task history, not process pause/resume.

## Verification

| Acceptance | Evidence / status |
|---|---|
| Active jobs/downloads block accidental close; explicit exit cancels | Go lifecycle tests, including live cancellable runner; native hooks inspected in the pinned Wails source. Native Alt+F4/dialog bridge click-through remains unverified. |
| Result/reuse actions and useful failures | Browser component checks for missing output and reuse dispatch; backend refuses unknown/active jobs. Windows association/Explorer launching still needs native manual verification. |
| Playback, seeking and boundary buttons | Browser component fixture plays a real six-second H.264 video, stops selected playback at 5.000s and sets a fractional start from current position. An unsupported-media fixture switches to frame navigation and still updates boundaries; too-short ranges show validation. Fixture adapters are test-only. |
| Arbitrary-codec position images and source protection | Real FFmpeg red/blue position and final-frame integration, invalid/non-finite positions; imported-media capability/range/changed-source HTTP tests. Native WebView2 asset playback remains unverified. |
| History/default persistence | Atomic concurrent update, corruption preservation, history merge/recovery/bounds and preset update/delete/reload Go tests. Active restored jobs are never enqueued. |
| Keyboard and responsive UI | Modal safe initial focus, Esc and restored trigger focus; Material preset keyboard selection; 375px dark/200% font shows no component overflow. |

The former native-launch automatic approval rejection was not retried through another launch mechanism. This release's browser component checks and headless FFmpeg tests do not claim native end-to-end coverage. Android remains deferred.
