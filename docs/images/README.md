# Desktop capture record — 2026-10-18

## Material iteration captures — 2026-10-25

`desktop-batch-material-dev.jpg` is an unedited 2026-10-26 capture of the actual native development window. It shows two owned sources passing real per-file GPU preflight, independent names/track policies, the sticky parameter inspector and reachable queue action. Both jobs were explicitly admitted afterward; independent probes/full decodes confirmed two eight-second HEVC/AAC outputs. This image is not an installer or clean-client acceptance record. The original screenshot API returned JPEG, which is retained without conversion.

`desktop-comparison-material-dev.jpg` is an unedited capture from the same date, after fixing comparison navigation across lazy loading and metadata expansion. The actual native player shows the owned original at its two-second offset and the real three-second H.264/AAC sample. Native playback, stop-at-end, static frames and split reveal were exercised; closing restores the initiating card and keyboard order. Like the batch image, this is development workflow evidence rather than packaged/clean-client evidence.

`desktop-appearance-material-dev.png` and `desktop-workbench-material-dev.png` are unedited captures of the actual Wails v2 development EXE at `.tools/uiux-native/PheebKit.exe`, on this Windows workstation. They show English with the Iris purple seed in dark mode. The workbench uses the same owned eight-second test video described below. No personal media was used. The executable reports modified source and is not the released rc.3 payload, even though the version has not yet been advanced during this iteration.

Native testing confirmed repeated full-range loop playback, paused 30 FPS stepping from about 0.194 to 0.227 seconds, eight real extracted thumbnails, and a real audio waveform window from about 0.227 to 8 seconds. Switching to Settings, changing the seed/theme/language and returning retained the source, preview position, eight thumbnails and waveform. New output-message layout/focus padding is a later source edit and requires fresh interaction verification. Full release/client acceptance is separate.

## Earlier rc.3 candidate captures

Current rc.3 captures are unedited screenshots of the native Wails v2.17.0 application on Windows 11 Pro build 26300:

- `desktop-workspace-rc3.png`: Chinese media-processing workspace with an actual thumbnail/probed metadata for a self-generated eight-second 640×360, 30 fps H.264/AAC source named `测试 & clip.mp4`.
- `desktop-settings-en-rc3.png`: English editing/recovery settings with the custom autosave switch.
- `desktop-language-en-rc3.png`: English appearance/language selector and actual qualified GPU summary.

These screenshots contain no personal media or native dialog paths. Earlier rc.2 and `workspace.png` captures are retained as historical evidence and are not the current README images.

Native rc.3 testing exercised Settings lazy loading, Chinese → English → Chinese switching, persisted English after restart, English native file-picker title/filter, Unicode media import, existing-output refusal, conflict-free filename suggestion, real RTX 4070 HEVC/AAC queued encoding, completion feedback, explicit draft recovery and unsaved-close confirmation. Independent full decode passed; the output duration is 8.000 seconds. The eight-second synthetic source can grow when encoded at the selected quality; the queue correctly reports increased size rather than claiming savings.

The screenshots alone do not establish every workflow, clean Windows 10/11, CPU-only or GPU-driver compatibility. Current candidate images are development-build evidence; exact committed installer provenance and lifecycle results are recorded separately in [distribution](../windows-distribution.md). See [language behavior](../i18n.md) and [release gates](../stability.md).
