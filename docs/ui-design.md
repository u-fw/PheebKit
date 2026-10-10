# Shared UI design

The technical spike implements Home, Compress and Queue. Other kompeg tools are future scope. The UI uses shared React components, TypeScript models generated from Go and a small binding adapter.

Material Design 3 inspired tonal surfaces, teal accent and a consistent 4/8px spacing scale form the visual system. kompeg's wordmark and K/play vector mark are shared by the sidebar, favicon and Windows executable icon. A restrained mint/lilac canvas sits behind the glass navigation: sidebar opacity is 60% light / 62% dark, while chrome opacity is 80% light / 84% dark. Glass is limited to the sidebar, top app bar and mobile bottom navigation, with 26px backdrop blur and a subtle inset highlight. Cards use opaque tonal surfaces without repeated blur. All colors, spacing, radii, elevation and motion use semantic CSS tokens.

The desktop app bar is 56px, page gutters are 32px and page titles are 30px. Local system fonts keep startup offline. Navigation and body labels maintain clear type levels; redundant page eyebrows and mixed-language decoration have been removed. The desktop import panel uses a horizontal illustration/content composition; phones stack it vertically.

Filled primary buttons, pill navigation and checked segmented controls express selection and action hierarchy. Resolution, audio and preset use a shared Material combobox/listbox rather than an operating-system dropdown. Arrow keys, Home/End, typeahead, Enter, Escape, Tab and outside clicks are supported. The trigger retains focus while `aria-activedescendant` identifies the active option. Menus choose above/below placement and constrain height between the fixed top/bottom bars. Phone controls stack into one column; codec tiles and segmented groups adapt to enlarged text.

The vendor-prefixed backdrop declaration precedes the standard declaration. The previous reverse order caused production CSS optimization to retain only a WebKit property that the audited Chromium WebView did not recognize. Final native verification confirms the standard computed value is `blur(26px)`; transparency alone is not treated as blur proof.

`frontend/src/styles.css` is the stylesheet entrypoint. Tokens, base rules, shell, workspace, media, compression, select, execution, queue, settings and responsive rules live in separate files under `frontend/src/styles/`. Their declaration order is preserved; the split produces identical production assets.

Windows uses a frameless Wails window with an integrated glass app bar, native drag region and custom minimize/maximize/restore/close controls calling the Wails Window API. OS shadows and Windows 11 corner decorations remain enabled. Composition hosting is not enabled. Android/browser layouts do not show desktop window controls. The title is simply `kompeg`; implementation-stage labels and marketing headings have been removed from the working screens. See the [Wails frameless window documentation](https://v3.wails.io/features/windows/frameless/).

Desktop uses a sidebar. Widths up to 960px switch to a bottom navigation rather than shrinking the desktop sidebar, including phone landscape. Safe-area insets, 44+ px targets, focus outlines, semantic controls and reduced-motion settings support touch and keyboard use. Light and dark themes use local persistence; fonts use the local system stack so the application works offline.

Home begins with file selection or drag/drop. A real Go probe result supplies the Media Card; no example file is presented as an imported file. Compress exposes codec family, quality, resolution, audio and output. Advanced controls expose encoder-specific quality, preset and one argv element per raw-argument line. UI never constructs the executable command. Switching codec family resets a GPU-specific encoder choice to automatic preference, because that encoder may not encode the new family.

Execution parameter preview comes from the Go compiler. Raw parameters are appended to output options with protected input/output/process flags. Codec-private suboptions use a typed safe allowlist to prevent auxiliary file access. Preview is refreshed after a debounce. Loading and validation errors appear beside the start button. Size mode with copied audio has a source-aware inline explanation and an explicit switch-to-AAC action; audio selection is never silently changed. Decimal MB and binary MiB are labeled consistently.

Progress combines throttled Go events with a short UI batching interval. Queue entries are independent records with progress/FPS/speed/out_time, cancellation, bounded logs and actual result metrics. Pause/resume of the FFmpeg process is outside this spike; cancel is a real process cancellation operation.

Size mode is a one-pass bitrate budget with muxing allowance and is an estimate, not a hard byte-size guarantee. Quality mode shows content-dependent size rather than inventing a percentage forecast. New settings default to automatic GPU preference; the actual compiler-selected encoder and any CPU fallback reason are displayed in the summary. Settings lists detected adapters and separately reports each encoder's real runtime trial result. CPU can be selected explicitly.

Media artwork loads asynchronously after probe. Windows `IShellItemImageFactory` is preferred, using the system's thumbnail provider to approximate Explorer's result. If unavailable, FFmpeg extracts an attached picture or a video frame. Failures retain a clear placeholder without preventing compression. Artwork is bounded and local; arbitrary paths must first pass the normal media import/probe flow.

Settings shows FFmpeg/ffprobe's actual executable paths, detected versions and configuration sources. Explicit environment variables take priority over saved application paths and PATH discovery. Missing binaries show an actionable setup message in the workspace and can be selected from Settings. Path changes are rejected while jobs are queued or running.

Responsive browser verification demonstrates shared layout; it does not prove an Android APK or native binding works. See the verification report for actual observed evidence.
