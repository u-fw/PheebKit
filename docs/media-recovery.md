# Decode checks and conservative recovery

This workflow repairs playback by discarding or replacing failed intervals. It cannot reconstruct missing original pictures or promise detection of all visual corruption. FFmpeg may conceal damage without reporting an error; missing timestamps, variable frame rate and container damage can prevent reliable analysis. An unopenable container/header is not repaired by this workflow.

## Intent and priority

1. Scan the imported local video, show progress/cancel, bind the completed report to the source fingerprint.
2. Default to deleting failed windows and the corresponding audio. The timeline becomes shorter. Review the actual ranges before applying; applying is an editable project change, not execution.
3. Alternatively replace failed windows with black video and silence, preserving approximately the source duration. This mode explicitly normalizes retained audio to 48 kHz stereo. Freeze-frame interpolation and original audio salvage are not implemented.
4. Execute through the normal serial cancellable queue, retain the original file, write to private scratch space, verify structure/duration and decode the entire result before no-overwrite publication. Failed/cancelled verification must not publish a result.

## Detection policy and bounds

Scan independent configurable windows (GUI: 1/2/5/10/30 seconds, default five; adapter: integer 1–30) using CPU decoding, one decoder thread, accurate input seek, `err_detect=explode`, `xerror` and zero decoding-error tolerance. Include seek preroll/reference decoding: failures can implicate more normal content than the actual damaged frame. Merge adjacent failures. Known media decode failures and empty/short decode output become unsafe windows. Unknown process/configuration failures abort the scan rather than generate deletion advice. Progress is completed-window progress, not an ETA or a frame-perfect corruption map.

Only completed scans are actionable. Maximum six hours and 4320 windows per source, one motion video and four audio tracks, 128 merged failure ranges, one active scan per app instance, two-minute timeout per window and twelve-hour overall deadline. Shutdown cancels and joins the scan; active scans guard environment replacement and native close. Output composition has at most 16 pieces, and retains at least one complete video frame per piece and some original content; excess fragmentation refuses execution with an explanation. No source interval is silently dropped to fit a limit. All-failed scans cannot create an ostensibly recovered all-black movie.

Recovery requires compression/re-encoding, removes soft subtitles, and excludes simultaneous trim/multi-range/burn/raw arguments. Perform those edits on the recovered file. The encoder can still follow the existing qualified GPU/CPU selection; full decode checks use CPU. Subtitle retiming, corrupt container reconstruction, exact-frame localization and restoration of discarded content are outside this increment.

## Data and trust boundary

Schema-1 settings add optional `recovery: { mode: "delete" | "black", sourceFingerprint, damagedRanges }`. It is editable typed intent, not a certificate that an imported project was scanned. Compile validates all fields/ranges/source identity again; output decoding is mandatory for any recovery intent. Deep snapshots preserve it in projects, drafts, Undo/Redo and queue history. Portable presets and unrelated batch files omit it. Samples refuse recovery because a sample cannot establish full-file acceptance. `OutputVerification.fullDecodePassed` is set only after the output's complete strict decode succeeds; ordinary jobs still receive structure checks only.

No shell, network input, arbitrary filter expression, extra output path, source replacement or raw-argument bypass. Runtime output buffers, subprocess time and graph/input counts are bounded. No media is uploaded and no scan starts automatically on import.

## Acceptance evidence

| Criterion | Evidence |
| --- | --- |
| Actual corruption | Publisher-pinned FFmpeg 9.0.2 creates a 16-second H.264/AAC fixture. Packet payloads at 6.2–6.4 seconds are overwritten; scan identifies 5–10 seconds. The original healthy fixture yields no failures. |
| Delete strategy | Real encoding yields approximately 11 seconds with video/audio and `fullDecodePassed=true`. |
| Preserve strategy | Real encoding yields approximately 16 seconds; sampled replacement pixels are black and decoded PCM is silence; full decode passes. |
| Fail closed | Remuxing the corrupted source produces a readable container, but mandatory full decode rejects it and leaves no published output. Cancelled scans cannot complete. |
| Domain/safety | Invalid/overlapping/nonfinite/out-of-bounds ranges, no retained content, excessive fragmentation, incompatible operations/copy/raw options and snapshot aliasing have regressions. Operational decoder/configuration failures are not deletion advice. |
| Service lifecycle | Duplicate/stale cancellation, source report snapshots, cancel settlement, close/environment guards, closed-service admission and portable-preset exclusion are tested. |
| UI | Disposable real-component browser fixture verifies progress/cancel, completed-only application, default deletion, black strategy, automatic audio re-encoding, Undo/Redo, healthy/error no-advice states, configurable one-second windows and disabled controls while scanning. 375px light/dark layouts have no horizontal overflow; final fresh-load run has no console errors. Earlier fixture hot-reload root warnings were confined to the disposable entry point. Fixture/server/tab/viewport override removed. |
| Release | Beta.9 production release gate passed real media Go tests/vet, 113 frontend tests/typecheck/lint/format/build, version tests and vulnerability checks. Actual beta.8 → beta.9 upgrade, payload/checksum consistency, reinstall/uninstall and configuration preservation passed. Exact source/log/checksum evidence is in [Windows distribution](windows-distribution.md). Native installed-window/GPU-recovery/clean-machine/hosted-CI gates remain external. |

## Subsequent priorities

Recovery and [opt-in audio processing](audio-processing.md) are implemented. Next: image/GIF sequences and encoding controls (two-pass/rate-control details), followed by explicitly verified HDR/color management. Measured two-pass loudness and mixing remain audio extensions. Multi-source timelines/filter authoring and network/capture need separate input/security/resource designs. Broad FFmpeg coverage remains incremental; discovery/help is not implemented execution support.

Primary semantics: [FFmpeg command options](https://ffmpeg.org/ffmpeg.html), [format error detection](https://ffmpeg.org/ffmpeg-formats.html#Format-Options), [concat/source/audio filters](https://ffmpeg.org/ffmpeg-filters.html).
