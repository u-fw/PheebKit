# Editing and recovery — 0.2.0-beta.6

## Configurable local recovery

Settings → Editing and Recovery exposes these preferences. Bounds are validated in the UI and Go service.

| Preference | Default | Allowed range |
| --- | --- | --- |
| Automatic drafts | On | On / Off |
| Checkpoint interval | 15 seconds | 5–300 seconds |
| Inactive historical drafts | 5 | 1–20 |
| Historical draft retention | 14 days | 1–90 days |
| Recent project records | 10 | 0–50; zero disables/clears recording |
| Undo history | 50 steps | 10–200 |

The editor checks the focused single-file project's committed settings at the selected interval. A changed, dirty project is written to a session checkpoint; unchanged intent does not generate repeated writes. A checkpoint is not a `.ffproject` save and does not clear the unsaved-edits indicator. The last interval of edits can be lost in a crash. Invalid/uncommitted text, batch UI drafts, queued/running process state and media contents are not recovery snapshots. Failed saves are visible and retain the last valid checkpoint.

Each checkpoint is at most 1 MiB. `%APPDATA%/kompeg/editor.json` is a version-1 file using the existing cross-process lock, temporary write/flush/atomic replacement and 24 MiB storage limit. Stale request revisions cannot resurrect a cleared checkpoint. Corrupt or unknown-version records are preserved, reported and never silently reset. The file contains private local media/project paths and settings; it is not uploaded. Existing FFmpeg/default/preset/history storage remains compatible.

The Home screen lists historical drafts and recent `.ffproject` files. Opening or restoring re-probes the source; missing source files reuse the native relocation picker. Recovery starts with unsaved state and never executes a job. An old checkpoint remains available until explicitly deleted or pruned, including after it is opened. A successful explicit save clears the current session checkpoint through the normal clean-editor transition, while older recovery copies remain subject to retention. Clearing recent records does not delete project files.

Active process checkpoints are protected from deletion, restoration and age/count pruning; activity uses process liveness, consistent with queue history. Current plus other active checkpoints can exceed the configured inactive-draft count, with a hard 32-entry/24 MiB limit. A full store reports an error rather than deleting active work. Retention is applied on reads for display and on subsequent writes/startup cleanup for storage; there is no background service while the application is closed. Turning autosave off retains existing checkpoints. Reducing retention/count settings explicitly prunes eligible older inactive records.

## Undo and redo

The focused file retains bounded settings/output-path history. Applying a configuration, changing operation or switching a codec with its output extension is one action. Continuous edits to the same field within 500 ms are grouped. Editing after Undo discards the redo branch. Saving establishes the dirty baseline without discarding edit history; undoing away from it is unsaved again. Replacing the focused file resets history. Changing undo capacity takes effect on subsequent actions.

Buttons expose Undo/Redo. Ctrl+Z, Ctrl+Shift+Z and Ctrl+Y work outside text/number inputs, text areas, selects, editable content and open dialogs. Inputs retain their native text undo behavior. Advanced-parameter controls refresh when restoring committed state. History does not reverse submitted jobs, files, saved defaults, external side effects or transient invalid input text.

## Editable scene presets and audio parameters

Compatible Playback, Lightweight Sharing, Quality First and Custom Size are explicit starting points; applying one never starts processing. They fill codec, quality/mode, encoder speed, resolution cap and audio parameters, leaving each normal field editable. Smaller source dimensions are not upscaled by the built-in presets. Device codec support, HDR policy and normal preview/compiler validation still apply. Quality First is lossy and can increase size; Custom Size uses a 50 decimal-MB starting estimate, not a size guarantee. Modified settings can be saved under a custom preset name.

Optional schema-1 `settings.audioBitrateKbps` controls encoded AAC/Opus per selected audio track, 32–320 kbps. Zero/omission retains AAC 192 / Opus 128 defaults. Compression and AAC/Opus audio extraction honor it; target-size budgeting subtracts the chosen per-track rate multiplied by selected audio count. Copy, silent output, remux, subtitle extraction and other audio formats do not use it, and inactive values do not block those operations. Older strict project readers need updating for this additive field. Sample duration is directly editable as 1–30 whole seconds, bounded by the actual selected range.

## Evidence and limits

Beta.7 adds a real operating-system termination test: a separate headless test executable writes a session checkpoint, preferences/preset and recent-project record; the parent verifies its PID, kills that owned child, creates a new service and restores the now-inactive checkpoint. Retained settings and unsaved state are checked, and no job starts automatically. An additional fixture reads pre-editor schema-1 configuration/history without rewriting it. This complements the real FFmpeg chain below; it does not establish native renderer/IPC, shutdown or file-dialog behavior.

Go regressions cover stale-save rejection after clear, active/inactive recovery, retention/counts, disabled autosave, oversized/corrupt record preservation, recent deduplication and clear-without-file-deletion, and custom/default/inactive audio bitrate behavior. A real FFmpeg test creates video/audio, persists a checkpoint, simulates a previous exited process, re-probes through a new service, verifies no job was started, then explicitly starts and verifies the resulting H.264/AAC output using the retained 64 kbps setting. Real AAC/M4A extraction also passes postflight verification.

The frontend suite has 100 tests in 19 files. Disposable full-App browser fixtures verified recovery, editable 80 kbps scene settings, actual interval checkpoint requests, Undo to 192 kbps, keyboard Redo to 80 kbps, direct 7-second sample entry, explicit save clearing the current checkpoint and populating recents, invalid interval refusal and custom preference application. The 375px dark layout had no observed console errors. Fixture files, server and browser tab were removed. These are browser bridge fixtures, not native IPC, native file-dialog or real OS-crash acceptance. Windows 10, clean-machine tests and native click-through remain manual; Android remains deferred.
