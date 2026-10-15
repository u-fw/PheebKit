# FFmpeg capability expansion

The direction is broad FFmpeg coverage through the existing local Go compiler/runner, with progressively exposed typed controls. No single installed FFmpeg contains every upstream feature: build options, external libraries, hardware, drivers and operating system matter. Listing a capability is not a successful runtime qualification or an implemented GUI workflow.

## This increment

Beta.10 additionally implements [typed audio processing](audio-processing.md) for compression/extraction, including single-pass loudness, denoise/EQ/fades/rate/channels. Track mixing and measured two-pass normalization remain future work.

Beta.9 additionally implements [conservative media scans/recovery](media-recovery.md), with mandatory complete decode verification of recovery outputs. Ordinary jobs still use metadata/structure checks. The beta.8 evidence below remains the preceding expansion's dated baseline.

- Inspect the selected local FFmpeg's encoders, decoders, filters, muxers/demuxers, protocols, bitstream filters, pixel/sample formats, channel layouts, colors, devices and hardware backends. Search results and read the binary's own option help. Fixed query categories/name validation, bounded output/time/cache, no shell, no execution of help text. Configuration replacement cannot race an inspection.
- Retain/reorder up to 16 ranges from one source, concatenate through an accurately timestamped filter graph and encode once. Bounds/one-frame minimum checked per range; repeated/overlapping ranges deliberately repeat content. Maximum four selected audio tracks for this graph bounds resource use. Independent export uses the existing all-or-none queue preflight. Soft subtitle retiming across concatenation is not silently approximated: remove soft tracks or burn them into video.
- Burn one embedded text subtitle track or local SRT/ASS/SSA/WebVTT file. Configure offset, font, size, colors, outline, shadow, margins and position. Stage a bounded ASS file using a forced local text demuxer or existing restricted source input; render only the generated workspace file, never interpolate an arbitrary filename into a filter. Source subtitles are fingerprinted before/after execution. No fonts/network download, OCR or bitmap subtitle rendering is implied.
- Preserve project/checkpoint/undo/queue snapshots. Portable presets and unrelated batch files omit source-specific ranges and subtitle references. Samples use the first retained segment and identify that policy. Existing schema-1 files remain readable; older strict readers reject new optional fields rather than silently losing edits.

## Invariants and acceptance

No shell, uncontrolled filter expressions, secondary output files, network inputs or overwrite bypass. Actual GPU trials remain authoritative for the supported encoding workflows; a catalog listing never marks a GPU usable. Filter absence produces actionable preflight errors. Cancellation includes subtitle preparation; only fully verified results publish. Existing FFmpeg completion/structure checks apply.

Acceptance requires real compiler → FFmpeg → ffprobe tests for reordered ranges (video/audio duration and content order), separate queue submission, embedded/external subtitle rendering and offset with nonzero trim, Unicode/punctuation paths, mutated subtitle refusal, unavailable filters, invalid styles/ranges, snapshot isolation, persistence round-trip, and capability queries against the actual binary. UI tests must cover direct time editing/reordering, subtitle settings, search/help and narrow dark layout. Native dialogs/installed-window interaction remain separately unverified where unavailable.

References: [FFmpeg generic options](https://ffmpeg.org/ffmpeg.html#Generic-options), [concat filter](https://ffmpeg.org/ffmpeg-filters.html#concat), [subtitles filter](https://ffmpeg.org/ffmpeg-filters.html#subtitles-1). The local binary's help is checked when implementing version-sensitive options.

## Verification record — 2026-10-15

| Criterion | Evidence / status |
| --- | --- |
| Same-source ordered/repeated ranges, timing/budget | Unit validation/budget checks; real red/blue source reordered into a two-second result, decoded frames checked for actual order. VERIFIED locally. |
| Audio and combined operations | Real audio decoded to PCM confirms -6 dB gain; joined duration, silent output, embedded subtitle burn combined with reordered ranges pass. VERIFIED on CPU. |
| Independent output / state integrity | UI bridge fixture checks numbered paths and per-range trim/subtitle intent; actual service/queue creates two independently verified real-media exports. Atomic-admission rejection, immutable snapshots, project round-trip/checkpoint/preset exclusions pass. VERIFIED locally. |
| Subtitle source/style/offset | Real external Unicode/punctuation filename and embedded track render; white-pixel checks confirm nonzero-trim timing and delayed disappearance; changed subtitle source refuses output. A playlist disguised as SRT is refused with zero HTTP requests; empty/no-dialogue preparation and canceled preparation cannot publish. Invalid fonts/styles/paths/filters/ranges have regressions. VERIFIED locally. |
| Discovery/help | Actual publisher-pinned Gyan FFmpeg 9.0.2 returns 13 catalogs and detailed/full help; entries include 579 filters and 76 channel/layout entries. Name validation, canceled lock waits and parser legends tested. VERIFIED for this binary, not every historical/custom build. |
| GUI | Disposable Compress/ProjectActions/Explorer fixture verifies direct time input, reorder, separate exports, subtitle save and Undo/Redo, external picker result, invalid-font disabled execution, search/help/pagination. 375px dark view has no horizontal overflow or console errors; expected Wails browser-preview warning only. Fixture/server/tab/viewport override removed. VERIFIED as browser bridge fixtures. |
| Full regression | 109 frontend tests in 21 files, TypeScript, ESLint, Prettier, production Wails build and Go unit/integration/vet pass. Source/binary govulncheck and npm audit report no known findings. Actual beta.7 → beta.8 install upgrade/reinstall/uninstall, identical payloads and configuration preservation pass. Packaging evidence is in [Windows distribution](windows-distribution.md). |
| Native/manual gates | Native subtitle picker, installed-window click-through, Windows 10/clean machines and hosted CI are IMPLEMENTED BUT EXTERNALLY UNVERIFIED. New multi-range/subtitle combinations have CPU evidence; GPU qualification is retained, but these new combinations have no native GPU execution acceptance. |

## Coverage beyond this increment

Not complete: HDR/tone mapping, arbitrary multi-file timelines/transitions, comprehensive filter graph authoring, network/live/device capture, all codec/muxer workflows, two-pass measured normalization/audio mixing, multiple output profiles, optional full-decode verification for ordinary jobs and unrestricted command execution. These require additional typed intent, resource/privacy policies and real operation-specific verification. The capability explorer provides discovery/help for these areas, not a claim of operational GUI support. Android remains deferred; existing Wails Beta approval remains limited to the Technical Spike.
