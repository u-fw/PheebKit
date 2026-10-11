# Media tools — 0.2.0-beta.1

The existing Go compiler/runner and React Material/glass components are reused. No dependency or framework was added/upgraded. Windows remains active; Android is deferred.

## Import and processing

Import up to 32 videos/audio files with multi-select or drop. Sequential analysis shows progress and retains successful imports when individual files fail. Stop Analysis stops subsequent probes after the current one finishes. Choose a source to inspect/edit a single task. Native `media:probed` notifications no longer replace the focused source during unrelated batch/comparison probes.

The processing picker exposes compression, remux, audio extraction and text subtitle extraction. Switching operation clears trim, filters, volume and advanced video parameters with explanatory text. A task has one input/output; a batch contains independent projects.

Audio selection defaults to all; subtitles default to none, preserving older project behavior. Explicit choices use global ffprobe indexes and validate type/existence/uniqueness. Labels include language, codec and source title; MP4 handler names are read as titles when no title tag exists.

Compression encodes the primary motion-video stream and selected audio/subtitles. MP4 retains the existing audio-copy policy and text subtitle conversion to `mov_text`; Matroska supports the explicit copy table and text fallback; WebM compression remains AV1/Opus with compatible text subtitles. Unsupported bitmap/text or container combinations fail preflight. Subtitles are embedded tracks, not burned into pixels. Copy audio cannot apply volume or accurate trim; the UI offers an explicit switch to AAC.

Remux copies primary motion video and selected compatible tracks without a video encoder/GPU. MP4 copy is conservatively limited to H.264/HEVC/AV1/MPEG-4 video, allowed audio and existing `mov_text`; WebM accepts VP8/VP9/AV1, Opus/Vorbis and existing WebVTT. Matroska uses the explicit subtitle copy table. HDR copying is permitted; HDR compression is unsupported. Chapters/global metadata are retained where supported. Covers, additional video streams, attachments and data are excluded; this is not an all-stream archival copy.

Audio extraction requires one selected track: AAC/M4A, Opus, MP3, FLAC, signed 16-bit WAV or compatible copying, including `.mka`. Encoded audio can trim in the backend; the current extraction UI exposes full duration and volume. FLAC/WAV conversion cannot restore detail already lost in the source. Text subtitle extraction emits SRT/ASS/VTT; conversion may discard styling. Image subtitles require OCR, which is not implemented.

## Transforms and samples

Transforms apply even-pixel crop to the source's displayed orientation, then clockwise rotation, horizontal/vertical flips and scaling. Display matrix/legacy rotation is parsed so phone videos agree with preview. Non-quarter-turn display rotations are refused for manual transforms. Crop must leave even width/height of at least two pixels. Preview shows the original, while samples/results show actual changes. Volume is -60…20 dB (zero preserves volume) and requires encoded audio; gain can clip peaks, with no automatic loudness normalization claimed.

Samples are regular cancellable jobs with distinct output filenames beside the proposed result. Backend duration is 1–30 seconds (10/20/30 in UI), starting at the selected range's beginning and bounded by its end. Settings are cloned; Copy audio is explicitly AAC for sample accuracy, leaving the full task unchanged. Size-mode budgets scale by sample-duration/selected-duration. Samples are retained outputs, not silently deleted files or guarantees of whole-video size/quality.

Completed samples expose corresponding-position comparison. Source/output are re-probed; the original is checked against the job fingerprint before/after probing, and changed sources are refused. FFmpeg frames reuse the bounded worker pool, images are at most 512px, and stale replies are ignored. Full-resolution details/audio require opening the sample file.

## Batch and queue

Batch uses common operation/encoding/rotation/volume settings for full durations, excluding the focused file's trim and crop. Each file has its own audio/subtitle choices; stream indexes are not copied blindly. Directory, suffix and included files are editable. Duplicate basenames receive distinct names, including collisions with naturally numbered source names.

All projects compile before atomic admission. Source/format/track/path failures, existing outputs, active path reservations or insufficient queue capacity reject the entire batch. Up to 32 pending jobs run serially. External creation of an output during processing is protected by final no-overwrite publication.

Scheduling pause stops new starts; current work continues. Waiting tasks have keyboard/button ordering and ranks. Cancelling waiting jobs releases capacity/reservations. Increasing queue revisions reject stale status replies. Pause/order are session state; restored history never auto-executes. ETA uses valid running `(total-current)/speed` and remains an estimate.

## Projects and shared presets

`.ffproject` is strict schema-1 UTF-8 JSON, at most 1 MiB, retaining single-file trim/track/crop intent. Atomic save protects source/output aliases. Open re-probes the source, assigns a new project identity and clears an existing output target for a new choice. Missing files return an error; relinking/recent-project UI is future work. Opening never executes.

`.pheebpreset` contains one schema-1 named configuration, at most 1 MiB. Paths, trim, selected indexes and crop are omitted; generic all/none track policies, rotation, audio/volume and encoding can remain. Import validates safe advanced argv, rejects unknown/trailing data and duplicate names, preserves defaults/current intent and never executes. Apply remains explicit; the 20-preset bound remains.

Optional schema-1 additions retain old defaults; older strict readers need updating. Existing AppData/environment keys, installer identity and GPL-3.0-only license remain unchanged.

## Acceptance evidence

| Acceptance | Evidence |
|---|---|
| Selection/remux/extraction | Compiler checks plus real runner four-track fixture; language/title/text readback, packet SHA-256 equality, AAC copy, WAV trim/measured -6 dB, SRT/ASS/VTT. |
| Transform/orientation | Lossless pixel comparison for crop/rotation/flip and phone display orientation; geometry/rotation readback. |
| Batch/scheduling | Atomic capacity/ID/path failure, original output preservation, actual serial reorder/pause/current-task continuation, queued/preparing cancellation, history bounds and deep clones. |
| Project/preset IO | Re-probe/no enqueue, intent restoration, existing-target reset, missing source, alias protection, unsafe argv and duplicate-name refusal, default preservation. |
| Samples | Real service → queue → FFmpeg → output probe/frame comparison, proportional budget, distinct output, changed-source refusal. |
| Frontend | 90 tests in 15 files, Typecheck/lint/format and production build. Test-only bridge browser fixtures verify keyboard operation switching, global track selection, Copy/volume prevention, sample and three-file batch payloads, and 375px dark/200% text component layout; not full App/native IPC evidence. |

Native multi-file/folder/save dialogs, Windows WebView2 comparison/default file associations and clean-machine acceptance remain manual checks. Prior native launching was rejected by automatic approval review and has not been retried through another mechanism. No Android package, signed binary or hosted release is claimed.

Policies were checked against official [FFmpeg commands](https://ffmpeg.org/ffmpeg.html), [filters](https://ffmpeg.org/ffmpeg-filters.html) and real output inspection. Tables describe the application's supported policy, not every possible FFmpeg combination.
