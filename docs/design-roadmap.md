# PheebKit design and feature roadmap

This document separates the interface running today from ideas that remain future work. The Windows application keeps its Go, stable Wails v2 and React architecture. It applies Jetpack Compose, Material Design 3 and Material You principles without pretending that a React renderer is a Compose runtime. A future Android client can use native Compose while sharing product rules and the Go media core where practical.

## Material foundation in the current client

- Official Material Color Utilities derive coordinated HCT tonal roles from a validated seed. Users can keep a brand/custom seed or follow the current Windows accent.
- Light, dark and system modes; comfortable and compact density; semantic surfaces; one restrained glass layer for app chrome and floating controls.
- Shared Material controls cover segmented choices, listbox selectors, switches, dialogs, progress and keyboard focus. The command palette is available with `Ctrl+K`.
- Pointer ripples and short spatial transitions provide state feedback. Reduced-motion and Windows forced-colours modes remove or replace those effects.
- Desktop navigation becomes a bottom navigation at narrow widths. Touch targets remain at least 44px, large text can reflow controls, and the bottom inset follows the measured navigation height rather than a hard-coded estimate.
- The app remains offline and local-first: no remote fonts, analytics, telemetry or translation service is required.

## Next design work

### P0: close release-quality UX gaps

1. **First-run guidance:** detect FFmpeg, GPU capabilities and language once, then show a short actionable checklist. A usable environment should lead directly to media import without setup noise.
2. **Material snackbar with Undo:** report low-risk reversible changes such as preset application, queue reordering and preference resets near the action that caused them. Destructive or irreversible actions still require confirmation.
3. **Taskbar and notification feedback:** synchronize job progress with the Windows taskbar; send a dismissible completion or failure notification only when the app is minimized or unfocused.
4. **Complete large-text and contrast matrix:** exercise 100–300% text, English/Chinese, portrait/landscape, light/dark, forced colours and both mouse and keyboard across Home, Editor, Batch, Queue and Settings.
5. **Focused empty and failure states:** every panel should explain what is missing, why an action is disabled and the next safe action. Errors should preserve entered settings and offer a retry path that does not duplicate jobs.

### P1: high-value media workflow design

1. **Plain-language hardware inspector:** show detected CPU, adapter, driver/build evidence and the currently successful NVENC/QSV/AMF combinations. Translate failures into actionable text such as “driver unavailable” or “format unsupported” instead of exposing only FFmpeg diagnostics.
2. **Outcome presets with estimates:** offer bounded presets for sharing, archive, messaging and maximum compatibility. Each preset shows expected codec, dimensions, audio, estimated size range and likely speed before it changes the editor.
3. **Filmstrip and waveform range editing:** add draggable start/end handles, keyboard nudging, chapter markers, zoom and a clear distinction between preview seeking and output-range intent.
4. **Recently used recipes and workspace memory:** restore the last inspector category, density, panel position and safe non-source-specific settings. Source paths, tracks and ranges never transfer silently between media.
5. **Contextual action bar:** expose sample, project, preset, recovery and comparison actions around the selected media or queue item while keeping one primary action visible.

### P2: larger product capabilities

- Multi-source timeline with transitions, overlays and per-clip effects.
- Subtitle editing, preview, timing and optional translation, with explicit privacy controls.
- HDR-aware color metadata, tone mapping and validation.
- Portable project archives that account for media retention, licensing and size.
- Optional capability packs for filters, codecs or fonts, installed only after checksum and license review.
- A native Android Compose shell after the Windows stable-release gates pass.

## Acceptance rules for new UI

- Use Material semantic roles and shared components before adding page-specific colors or controls.
- Keep glass limited to navigation, bars, overlays and floating surfaces so content contrast and rendering cost stay controlled.
- Every pointer action has a keyboard path and visible focus. No action relies on color, hover or animation alone.
- Preserve source identity, output intent and unsaved work when navigation, resizing, language or appearance changes.
- Treat estimates as estimates. Never invent progress, media metadata, hardware support or successful encoding.
- Verify actual Windows WebView2 behavior separately from responsive browser fixtures. Hosted Windows Server CI does not replace Windows 10/11 client acceptance.
