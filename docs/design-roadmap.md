# PheebKit design and feature roadmap

This document separates the interface running today from ideas that remain future work. The Windows application keeps its Go, stable Wails v2 and React architecture. It applies Jetpack Compose, Material Design 3 and Material You principles without pretending that a React renderer is a Compose runtime. A future Android client can use native Compose while sharing product rules and the Go media core where practical.

## Compose and Material You fidelity boundary

- **Current Windows client:** React renders the interface inside the stable Wails v2 WebView2 shell. It uses Compose interaction patterns and Material 3 semantic roles, but it is not a Kotlin/Compose runtime and should not be described as one.
- **Visual and interaction parity:** HCT-derived tonal palettes, Windows-accent personalization, light/dark/system modes, density, state layers, ripples, tonal containers, adaptive navigation, keyboard focus, snackbars and reduced-motion behavior can follow Material 3 and Material You closely.
- **Platform integration:** Windows title controls, taskbar progress, notifications, dialogs and file thumbnails remain native adapter responsibilities. They do not require moving the desktop renderer to Compose.
- **Future Android client:** native Jetpack Compose is the preferred implementation for Android when that platform resumes. Shared product rules, data schemas and the media core should remain independent of React and Compose where practical.

## Material foundation in the current client

- Official Material Color Utilities derive coordinated HCT tonal roles from a validated seed. Users can keep a brand/custom seed or follow the current Windows accent.
- Light, dark and system modes; comfortable and compact density; semantic surfaces; one restrained glass layer for app chrome and floating controls.
- Shared Material controls cover segmented choices, listbox selectors, switches, dialogs, progress and keyboard focus. The command palette is available with `Ctrl+K`.
- Pointer ripples and short spatial transitions provide state feedback. Reduced-motion and Windows forced-colours modes remove or replace those effects.
- Reversible queue changes use an inverse-color Material snackbar with an Undo action. Hover or focus pauses expiry, Escape dismisses it, and failed Undo keeps the feedback visible with an accessible explanation.
- Desktop navigation uses an expanded sidebar, a Material 3 navigation rail and a compact bottom bar. Touch targets remain at least 44px, large text can reflow controls, and the bottom inset follows the measured navigation height rather than a hard-coded estimate.
- Home reports FFmpeg, verified GPU/CPU fallback and local-processing readiness before import. Missing FFmpeg changes the primary action to setup instead of letting a known-invalid probe start.
- The app bar exposes active-job progress and opens the queue from one compact control. The native host mirrors queue progress to the Windows taskbar and batches completion or failure into one background notification; clicking it restores the window and focuses the relevant queue item.
- The app remains offline and local-first: no remote fonts, analytics, telemetry or translation service is required.

## Next design work

### P0: close release-quality UX gaps

1. **Complete the text and contrast matrix:** exercise 100–300% text, English/Chinese, portrait/landscape, light/dark, forced colours and both mouse and keyboard across Home, Editor, Batch, Queue and Settings.
2. **Focused empty and failure states:** every panel should explain what is missing, why an action is disabled and the next safe action. Errors should preserve entered settings and offer a retry path that does not duplicate jobs.

The readiness summary and native job feedback are implemented in the current RC source. Their clean Windows 10/11 and minimized/unfocused notification click-through still belong to the separate release-acceptance matrix.

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
