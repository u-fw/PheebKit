# Shared UI design

The technical spike implements Home, Compress and Queue. Other kompeg tools are future scope. The UI uses shared React components, TypeScript models generated from Go and a small binding adapter.

Material Design 3 inspired tonal surfaces, teal accent, large radii and spacious typography form the visual system. kompeg's wordmark and K/play vector mark are shared by the sidebar, favicon and Windows executable icon. A mint/lilac canvas creates a visible layer behind 62% light / 68% dark glass. Glass is limited to the sidebar, top app bar and mobile bottom navigation, with 26px backdrop blur. Cards use opaque tonal surfaces without repeated blur. All colors, spacing, radii, elevation and motion use semantic CSS tokens.

Filled primary buttons, pill navigation and checked segmented controls express selection and action hierarchy. Resolution, audio and preset use a shared Material combobox/listbox rather than an operating-system dropdown. Arrow keys, Home/End, typeahead, Enter, Escape, Tab and outside clicks are supported. The trigger retains focus while `aria-activedescendant` identifies the active option. Menus choose above/below placement and constrain height between the fixed top/bottom bars. Phone controls stack into one column; codec tiles and segmented groups adapt to enlarged text.

The vendor-prefixed backdrop declaration precedes the standard declaration. The previous reverse order caused production CSS optimization to retain only a WebKit property that the audited Chromium WebView did not recognize. Final native verification confirms the standard computed value is `blur(26px)`; transparency alone is not treated as blur proof.

`frontend/src/styles.css` is the stylesheet entrypoint. Tokens, base rules, shell, workspace, media, compression, select, execution, queue, settings and responsive rules live in separate files under `frontend/src/styles/`. Their declaration order is preserved; the split produces identical production assets.

Desktop uses a sidebar. Widths up to 960px switch to a bottom navigation rather than shrinking the desktop sidebar, including phone landscape. Safe-area insets, 44+ px targets, focus outlines, semantic controls and reduced-motion settings support touch and keyboard use. Light and dark themes use local persistence; fonts use the local system stack so the application works offline.

Home begins with file selection or drag/drop. A real Go probe result supplies the Media Card; no example file is presented as an imported file. Compress exposes codec family, quality, resolution, audio and output. Advanced controls expose CRF, preset and one argv element per raw-argument line. UI never constructs the executable command.

Command Preview comes from the Go compiler. Raw parameters are appended to output options with protected input/output/process flags. Preview is refreshed after a debounce. Errors disable start and display the backend's reason.

Progress combines throttled Go events with a short UI batching interval. Queue entries are independent records with progress/FPS/speed/out_time, cancellation, bounded logs and actual result metrics. Pause/resume of the FFmpeg process is outside this spike; cancel is a real process cancellation operation.

Size mode is a one-pass bitrate budget with muxing allowance and is an estimate, not a hard byte-size guarantee. Quality mode shows content-dependent size rather than inventing a percentage forecast. CPU is the only execution policy verified during this spike; GPU device naming/selection needs later capability and execution tests.

Responsive browser verification demonstrates shared layout; it does not prove an Android APK or native binding works. See the verification report for actual observed evidence.
