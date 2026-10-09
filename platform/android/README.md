# Android platform adapter

This directory documents the native integration boundary. It contains no compiled Kotlin/JNI backend or FFmpegKitNext AAR yet. The Android runner in the first technical spike reports unsupported FFmpeg operations explicitly.

Read [Android shell status](../../docs/android.md) before attempting a device build, and [Android FFmpeg spike](../../docs/android-ffmpeg-spike.md) for problems, evidence and the integration proposal. The current gate is a successful APK launch and a real React-to-Go `GetPlatform()` call returning `android`.

## Current build preparation

The root `Taskfile.yml` has no Android include, so Windows builds cannot evaluate the upstream Android tasks' POSIX host-detection commands. `build/android/Taskfile.yml` is retained as a reference, not a prepared Gradle app. Full pinned Gradle/Java/NDK build assets exist only in the ignored `_scaffold` inspection copy and must be generated/staged on the chosen supported host before APK assembly.

The Windows Android task attempt failed with exit 127 on unavailable `uname` / `tail`. SDK, NDK and emulator were not found; JDK 25.0.1 and adb 37.0.1 are available, but no device is attached. Android arm64 shared Go package compilation with `CGO_ENABLED=0` and Android main-overlay generation succeeded. Neither check produces a cgo `libwails.so`, APK or working native bridge.

The pinned CLI verifies the explicit task selector `wails3 task -taskfile <prepared-Taskfile.yml> <task-name>`. Generate the full pinned build assets into an isolated staging directory, review common task includes and working directories, then integrate Android assets without replacing the Windows configuration. Current root-task commands such as `wails3 task android:build` are unavailable until that preparation is completed. See the shell document for the ordered preparation and acceptance checks.

## Ownership

| Layer | Responsibility |
| --- | --- |
| Shared `internal/core` | Project/Source/Media models, versioned persistence and Runner contract |
| Shared compiler and jobs | Validation, argv execution plan, queue state and throttled events |
| `internal/runner/android` | Implement the core contract through a platform bridge |
| App-owned native bridge | JavaVM/thread attachment, reference lifetimes, job/session callback routing |
| Android Java/Kotlin adapter | FFmpegKitNext sessions, ContentResolver/SAF, output export and native capabilities |
| Foreground service | Background work, notification, timeout and shutdown cancellation |
| Wails Android host | WebView, Go runtime, shared frontend service/event transport |

Native code belongs here or in the Android build source tree when it can be compiled and exercised. Keep adapters small and app-owned; do not access Wails' private JNI helpers or scatter platform checks through common services.

## Artifact and storage rules

- Build a pinned FFmpegKitNext tag on a supported Linux/macOS host and consume its generated local Maven repository. No official ready-made Maven Central AAR exists.
- Target `arm64-v8a` for the native device spike; use a separate `x86_64` artifact for emulator testing when needed.
- Align native-stage minSdk with FFmpegKitNext's API 24 minimum. Record toolchain/ABI/options/licenses/checksums and verify final `.so` and package alignment.
- Persist the original `content://` reference and display metadata in Project IR. Grants, descriptors and temporary `ffkitsaf:` registrations remain native resources.
- Pass argv arrays to native execution. Use session IDs for cancel and callbacks; completion determines the terminal state.
- Validate the output after encoding/export. No fabricated progress, probe or success data substitutes for native acceptance.

WSL Debian is present but its toolchain and FFmpegKitNext build have not been verified. The pinned Wails beta.27 Android Taskfile supports Linux/Darwin host compilation, despite broader support described by rolling upstream documentation. These limitations and the acceptance matrix are recorded in the linked documents.
