# Windows platform boundary

`internal/runner/windows` owns FFmpeg/ffprobe executable discovery, argv process execution, machine progress, stderr logs, cancellation and no-clobber output publication. Shared Core, compiler and job code do not invoke `os/exec`.

The stable Wails v2 adapter uses native file/open/save dialogs. `build/windows` contains the application icon, long-path-aware manifest and executable metadata. `scripts/build-windows.ps1` embeds the production frontend into `bin/PheebKit.exe` and stamps exact revision/dirty state through supported linker flags.

FFmpeg is currently a separately installed dependency. Set `PHEEBKIT_FFMPEG` and `PHEEBKIT_FFPROBE` to absolute executable paths when PATH discovery is unsuitable. Recognized Scoop shims are resolved to their underlying executable. A release still needs installer design, signing, dependency packaging and a license audit of the chosen FFmpeg build.

The stable Wails v2 shell does not expose a production debug-port option. Native-window acceptance uses the real Windows interface. Historical v3 debug-port evidence applies only to those dated builds.
