# Windows platform boundary

`internal/runner/windows` owns FFmpeg/ffprobe executable discovery, argv process execution, machine progress, stderr logs, cancellation and no-clobber output publication. Shared Core, compiler and job code do not invoke `os/exec`.

The Wails service uses native file/open/save dialogs. `build/windows` contains the WebView2 application icon, manifest (including long-path awareness) and executable metadata. `wails3 build` embeds the production frontend into `bin/kompeg.exe`.

FFmpeg is currently a separately installed dependency. Set `KOMPEG_FFMPEG` and `KOMPEG_FFPROBE` to absolute executable paths when PATH discovery is unsuitable. Recognized Scoop shims are resolved to their underlying executable. A release still needs installer design, signing, dependency packaging and a license audit of the chosen FFmpeg build.

`KOMPEG_DEBUG_PORT` enables WebView2 remote debugging only when explicitly set to a port in 1024–65535. It was used for native-window verification; ordinary launch leaves it disabled.
