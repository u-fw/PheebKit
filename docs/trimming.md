# Optional trimming with compression

## Active Material preview iteration (2026-10-25)

The source preview is available for the full video, not only after enabling trim. Selection playback can loop, and previous/next controls seek by the source's declared average frame duration (0.1-second controls when the rate is unknown). This is approximate for variable-frame-rate sources and does not promise exact decoded-frame cuts. Stepping pauses playback; seeking alone never changes project boundaries. Source and selected-range playback remain separate actions.

Filmstrip thumbnails load only on request, at up to eight evenly spaced positions. Work is serial and bounded by the existing frame-extraction deadline/gate; leaving the editor stops the remaining requests. A failed cell still provides a time-position button. Waveform preview explicitly selects a source audio track and reads at most the next 120 seconds from the current preview position, returning up to 512 mono peaks. It has a 15-second total deadline and bounded output, is not a loudness/quality measurement, and does not alter selected export tracks. Hidden/stale UI results are discarded. Multi-segment ordering supports visible move buttons and Alt+ArrowUp/ArrowDown while a segment is focused. The player remains mounted when the active range is adjusted.

The [active iteration record](uiux-iteration.md) separates current automated/native evidence and remaining checks from the historical beta evidence below.

## Beta.8: ordered ranges and subtitles

The original single-range flow below remains. **多片段拼接** additionally stores an ordered `settings.segments` list (1–16 ranges), mutually exclusive with `trim`. Every range must be finite, within known source duration and at least one video frame. Repeated/overlapping ranges are intentional repetitions. Add/delete/up/down controls work without dragging; the existing direct-entry time picker edits the selected range. Range changes position its original-source preview at the selected start.

One joined output uses bounded independent input seeks of the same source, timestamp resets, one concat filter graph and a single encode. Decoder threads are limited per input, complex-filter threads are limited to one, and at most four audio tracks are retained. Progress/size budgets use summed selected duration. Soft subtitle retiming across joins is refused; the UI offers explicit soft-track removal, and text burn-in can follow each source interval. The separate-export button creates unique numbered single-range projects and uses existing atomic batch admission; size budgets are proportionally divided. No existing output is overwritten.

Samples retain the first listed range only, capped at its end, with a proportional share of the whole edit's size budget. Multi-file batch/default presets omit source-specific ranges/burn references. Projects/checkpoints/undo retain them. See [expansion details and evidence](ffmpeg-expansion.md). This is same-source range editing, not lossless cuts or a multi-file transition timeline.

## Original single-range behavior

The compression page defaults to the entire source. **剪辑并压缩** keeps one continuous interval of the input and encodes it in the same invocation. This original mode has no second intermediate encode; the beta.8 multi-range mode is described above. Neither mode is a lossless stream-copy cut.

Both boundaries use Material hour/minute/second selectors limited by the known source duration (hours may exceed 23). Fractional values from sliders are retained when selecting a unit, and ±0.1-second buttons allow finer adjustment. Both boundaries also have keyboard-accessible sliders with millisecond steps. Inline feedback rejects reversed, empty, out-of-duration and shorter-than-one-frame intervals. Unknown duration disables range selection. Full-source mode keeps existing behavior. Switching modes retains the editable range; importing a different source resets settings. Beta.2 had manual timecode fields; beta.3 replaces them with these selectors.

The optional `settings.trim` object contains `startSeconds` and `endSeconds`, relative to the source beginning. Missing/null means the full source. This additive schema-1 setting preserves loading of old project files; older strict readers will reject projects containing the new field, so exchange requires a reader supporting trim. No final command is stored in the project.

The compiler validates the range independently, then emits input `-ss` with `-accurate_seek` and output `-t` for the interval duration. The behavior follows [FFmpeg's seek/duration documentation](https://ffmpeg.org/ffmpeg.html#Main-options). Video is always encoded. Audio must use AAC/Opus when trimming: mixed stream-copy audio can retain packet-aligned prefixes, so enabling trim changes Copy to AAC and disables Copy until full-source mode is selected. The UI explains this behavior; it does not silently select another video encoder. Actual cut times are limited by frame/sample precision, not guaranteed millisecond-exact output durations.

Trimmed outputs omit source chapters because their original timestamps may no longer be meaningful. Subtitle/data streams remain excluded under the existing compression policy. Target-size budgeting, bitrate estimates, queue total duration and processing speed use `end - start`. Size mode remains a one-pass estimate. Original size remains the complete source size; the queue explicitly explains that reductions include discarded content.

## Verification on 2026-10-10

Beta.4 adds source/selected-range playback, current-position boundary buttons and FFmpeg position/boundary frames. Unsupported WebView2 playback falls back to frame navigation. End-boundary thumbnails show a nearby frame; the exact source end is clamped to the last readable frame. These previews do not promise decoded frame-exact playback cuts. New source imports use saved defaults when configured; media-specific trim resets. See [workflow verification](workflows.md).

- Compiler regressions cover seek placement, selected duration and size budget, unchanged whole-input behavior, invalid/unknown/non-finite boundaries, audio policy, shorter-than-frame rejection and raw parameter attempts to override managed timing.
- Project load/save preserves fractional ranges and old files without trimming. Queue tests verify ranges cannot be changed by mutating caller/snapshot objects.
- `PHEEBKIT_INTEGRATION=1 go test ./internal/runner/windows -run '^TestIntegrationTrimExactNonKeyframeRange$' -count=1 -v` passed using the installed FFmpeg. A six-second red/blue/green fixture with a long GOP was cut at non-keyframe 2.5–3.75 seconds. Independently probed output was H.264/AAC and 1.267 seconds (one-frame/container precision); decoding its first RGB frame confirmed blue, not the preceding red interval. Progress used 1.25 seconds.
- Browser component verification uses synthetic media, not native IPC: time entry, automatic AAC, disabled launch for reversed bounds, keyboard slider increments, full-range restoration and bitrate estimate were checked. Desktop light/dark and 375×812 / 812×375 layouts had no horizontal overflow. This does not establish native WebView2 click-through of the new feature.

Native launch verification remains pending; the previous automatic approval review blocked native app launching, and this task does not retry it through another mechanism. Android remains deferred.
