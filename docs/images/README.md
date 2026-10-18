# Desktop capture record — 2026-10-18

Current rc.3 captures are unedited screenshots of the native Wails v2.17.0 application on Windows 11 Pro build 26300:

- `desktop-workspace-rc3.png`: Chinese media-processing workspace with an actual thumbnail/probed metadata for a self-generated eight-second 640×360, 30 fps H.264/AAC source named `测试 & clip.mp4`.
- `desktop-settings-en-rc3.png`: English editing/recovery settings with the custom autosave switch.
- `desktop-language-en-rc3.png`: English appearance/language selector and actual qualified GPU summary.

These screenshots contain no personal media or native dialog paths. Earlier rc.2 and `workspace.png` captures are retained as historical evidence and are not the current README images.

Native rc.3 testing exercised Settings lazy loading, Chinese → English → Chinese switching, persisted English after restart, English native file-picker title/filter, Unicode media import, existing-output refusal, conflict-free filename suggestion, real RTX 4070 HEVC/AAC queued encoding, completion feedback, explicit draft recovery and unsaved-close confirmation. Independent full decode passed; the output duration is 8.000 seconds. The eight-second synthetic source can grow when encoded at the selected quality; the queue correctly reports increased size rather than claiming savings.

The screenshots alone do not establish every workflow, clean Windows 10/11, CPU-only or GPU-driver compatibility. Current candidate images are development-build evidence; exact committed installer provenance and lifecycle results are recorded separately in [distribution](../windows-distribution.md). See [language behavior](../i18n.md) and [release gates](../stability.md).
