# Optional trimming with compression

The compression page defaults to the entire source. **剪辑并压缩** keeps one continuous interval of the input and encodes it in the same invocation; there is no second intermediate encoded file or order selector. This is range trimming, not a timeline editor, multi-segment join or lossless stream-copy feature.

Both boundaries use Material hour/minute/second selectors limited by the known source duration (hours may exceed 23). Fractional values from sliders are retained when selecting a unit, and ±0.1-second buttons allow finer adjustment. Both boundaries also have keyboard-accessible sliders with millisecond steps. Inline feedback rejects reversed, empty, out-of-duration and shorter-than-one-frame intervals. Unknown duration disables range selection. Full-source mode keeps existing behavior. Switching modes retains the editable range; importing a different source resets settings. Beta.2 had manual timecode fields; beta.3 replaces them with these selectors.

The optional `settings.trim` object contains `startSeconds` and `endSeconds`, relative to the source beginning. Missing/null means the full source. This additive schema-1 setting preserves loading of old project files; older strict readers will reject projects containing the new field, so exchange requires a reader supporting trim. No final command is stored in the project.

The compiler validates the range independently, then emits input `-ss` with `-accurate_seek` and output `-t` for the interval duration. The behavior follows [FFmpeg's seek/duration documentation](https://ffmpeg.org/ffmpeg.html#Main-options). Video is always encoded. Audio must use AAC/Opus when trimming: mixed stream-copy audio can retain packet-aligned prefixes, so enabling trim changes Copy to AAC and disables Copy until full-source mode is selected. The UI explains this behavior; it does not silently select another video encoder. Actual cut times are limited by frame/sample precision, not guaranteed millisecond-exact output durations.

Trimmed outputs omit source chapters because their original timestamps may no longer be meaningful. Subtitle/data streams remain excluded under the existing compression policy. Target-size budgeting, bitrate estimates, queue total duration and processing speed use `end - start`. Size mode remains a one-pass estimate. Original size remains the complete source size; the queue explicitly explains that reductions include discarded content.

## Verification on 2026-10-10

- Compiler regressions cover seek placement, selected duration and size budget, unchanged whole-input behavior, invalid/unknown/non-finite boundaries, audio policy, shorter-than-frame rejection and raw parameter attempts to override managed timing.
- Project load/save preserves fractional ranges and old files without trimming. Queue tests verify ranges cannot be changed by mutating caller/snapshot objects.
- `KOMPEG_INTEGRATION=1 go test ./internal/runner/windows -run '^TestIntegrationTrimExactNonKeyframeRange$' -count=1 -v` passed using the installed FFmpeg. A six-second red/blue/green fixture with a long GOP was cut at non-keyframe 2.5–3.75 seconds. Independently probed output was H.264/AAC and 1.267 seconds (one-frame/container precision); decoding its first RGB frame confirmed blue, not the preceding red interval. Progress used 1.25 seconds.
- Browser component verification uses synthetic media, not native IPC: time entry, automatic AAC, disabled launch for reversed bounds, keyboard slider increments, full-range restoration and bitrate estimate were checked. Desktop light/dark and 375×812 / 812×375 layouts had no horizontal overflow. This does not establish native WebView2 click-through of the new feature.

Native launch verification remains pending; the previous automatic approval review blocked native app launching, and this task does not retry it through another mechanism. Android remains deferred.
