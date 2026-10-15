# Configurable audio processing

## Scope and contract

Extend existing compression and audio extraction with optional typed `settings.audioProcessing`. Existing projects with this field absent retain their previous behavior. Controls are off until explicitly enabled; disabling processing retains the independent legacy volume setting.

Each selected output track is processed independently. For ordered video segments the chain operates after concatenation, so normalization and fades span the complete selected output timeline. Separate exports apply the chain independently to each resulting file. This increment does not mix tracks or add external audio.

Processing order: explicit working format/resampling and requested mono/stereo conversion → existing volume gain → FFT denoise → up to eight peaking EQ bands → optional dynamic single-pass loudness normalization → output-relative linear fades → explicit output rate/sample format. Configure integrated loudness -70…-5 LUFS, true peak -9…0 dBTP, LRA 1…50 LU; denoise reduction 0/off or 0.01…97 dB and noise floor -80…-20 dB; EQ 20…20000 Hz, Q 0.1…20, gain ±24 dB; fades 0…600 seconds with a known output duration and no overlap. Active EQ centers must be below the chosen output Nyquist frequency.

Output sampling defaults to 48000 Hz while processing is enabled, with explicit common rates from 8000 to 96000 Hz. Mono/stereo conversion is optional; otherwise preserve source layout. Opus output exposes the standardized 48000 Hz decoded clock, and MP3 refuses rates above 48000 Hz. Unknown/unavailable filters, incompatible copied/muted audio, missing selected tracks or invalid parameters fail preflight rather than silently skip processing. Typed numeric/enum fields never grant arbitrary filters, input paths, secondary outputs or network access.

Dynamic single-pass loudnorm is explicitly identified as such, not a two-pass linear or mastering-compliance guarantee. Normalization can change dynamics, heavy denoise/EQ can reduce quality, and lossy encoding/resampling can alter measured peak/loudness. Samples use the sample's own content; whole-output fades are disabled for samples and disclosed, avoiding artificial early fade-out or invalid durations. Original fade intent is validated before sample narrowing. Recovery remains a separate first pass: clear audio processing when applying recovery, then process the recovered file.

Fades follow the planned selected output timeline, not detected first/last audible samples. An audio track shorter than the container/video timeline can finish before a planned fade-out; duration differences remain output review warnings. Extract the audio first and process the resulting audio-only file when its own end must define the fade. Silence trimming/detection and source/container timestamp repair are separate capabilities.

## Persistence and acceptance

New settings use the existing schema-1 optional-field policy and deep snapshots for projects, drafts, presets, Undo/Redo and jobs. Portable audio controls can transfer between sources; source-specific ranges do not. Backend revalidates rate/track/duration/capabilities for every file. Output expectations verify explicitly requested/default-processed sample rate and mono/stereo counts before publication. No dependencies or platform framework changes are required.

| Criterion | Local evidence |
| --- | --- |
| Domain/safety | VERIFIED bounds/nonfinite/enum/filter/copy/operation/Nyquist/fade rejection, channel/rate verification and deep snapshot isolation. |
| Actual effects | VERIFIED with publisher-pinned FFmpeg 9.0.2: target -16 LUFS measured -16.04; 1 kHz attenuation -12.00 dB; seeded noise RMS ratio 0.100; PCM fade envelope; AAC 44.1 kHz, Opus 48 kHz, MP3 24 kHz and FLAC 96 kHz stereo outputs verified by probe. These are fixture measurements, not every-media quality guarantees. |
| Composition | VERIFIED real two-track video concat with normalization/denoise/fades, output midpoint envelope preventing per-segment fades, nonzero source trim and audio-only effects. Existing disabled/legacy-volume regressions pass. |
| Application/persistence | VERIFIED actual Service queue output at 16 kHz mono; project/draft/preset round trips, immutable queued samples, whole-output fades omitted only after original-intent validation. |
| UI | VERIFIED browser real-component fixture: direct numeric input, opt-in/Copy conflict action, normalization/denoise/EQ, invalid fade/Nyquist refusal, 8-band cap, Undo/Redo, rate/layout selectors. 375px dark and 812px landscape light have no horizontal overflow or console errors. A follow-up reproduced noise-reduction input disappearing at zero; controls now remain mounted, typing `0.5` character by character updates the model to 0.5, and blur removes leading zeros. An SSR regression protects visibility at zero. Fixture/server/tab/viewport removed. |
| Release | Production package and installer evidence is recorded in Windows distribution after verification. |

Evidence is recorded after executing these checks. Native installed-window/GPU combinations, clean Windows 10/11 and hosted CI remain external unless separately verified. Later work includes measured two-pass normalization, mixing/routing and per-track independent effect chains.

Primary reference: [FFmpeg audio filters](https://ffmpeg.org/ffmpeg-filters.html), particularly loudnorm, afftdn, equalizer, afade and aresample. Validate behavior against the selected local FFmpeg build.
