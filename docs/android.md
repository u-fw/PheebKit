# Android technical spike

Android development and APK distribution are deferred by the owner as of 2026-10-10. The findings below are retained for a future milestone; current release packaging targets Windows x64 only.

Status recorded on 2026-10-09 (Asia/Singapore). Shared Go packages cross-compiled for Android arm64 with cgo disabled, and Android main-overlay generation succeeded. An Android build attempt failed before native compilation. No Android APK, device launch or frontend-to-Go call has been verified on this machine. Native FFmpeg integration follows the shell spike; it is not implemented by this milestone.

## Version and support boundary

The repository deliberately pins `github.com/wailsapp/wails/v3 v3.0.0-beta.27` for a reproducible spike. The user explicitly approved Wails v3 Beta and experimental Android support on 2026-10-09. The current upstream release is [beta.28](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.28); this project does not upgrade just because a newer release exists.

Wails documents a native Android WebView backed by the Go asset server, with service bindings and events sharing the desktop frontend. Android Go is compiled as a C shared library, `libwails.so`. Mobile remains outside the desktop beta compatibility promise. See [mobile overview](https://v3.wails.io/guides/mobile/) and the [official beta announcement](https://v3.wails.io/blog/wails-v3-beta/).

The current [Android guide](https://v3.wails.io/guides/mobile/android/) describes Windows build support, but the inspected [beta.27 Android Taskfile](https://github.com/wailsapp/wails/blob/v3.0.0-beta.27/v3/internal/commands/build_assets/android/Taskfile.yml) recognizes only `Darwin` and `Linux` in `compile:go:shared`; other hosts return `Unsupported host OS`. Its SDK discovery and compiler paths are POSIX-oriented. The pinned assets, rather than the rolling documentation, govern this spike. Native Windows Android compilation is therefore not established for this pin.

## Local environment

The table records the Windows environment audit and the separate read-only WSL host check described below.

| Component | Observed result | Android implication |
| --- | --- | --- |
| Host | Windows 11, PowerShell | Windows desktop build host |
| Go | 1.27.1 | Above the Wails documented Go 1.25 minimum |
| Node / npm | 24.11.1 / 11.6.2 | Available for the shared frontend |
| Wails | v3.0.0-beta.27 pinned | Beta and experimental mobile approved for this spike |
| Java / javac | Microsoft OpenJDK 25.0.1; `JAVA_HOME` points to a Minecraft runtime | Java tools available; Android Gradle execution not verified |
| adb | 37.0.1; no connected device | Platform-tools alone do not supply an Android build environment |
| Android SDK | Not found in the audit | SDK platform/build-tools cannot currently be used |
| Android NDK | Not found in the audit | No Android cgo compiler available |
| Android emulator | Not found in the audit | No emulator launch possible with the discovered tools |
| WSL | Debian 13.6 on x86_64 WSL2; read-only host audit completed | Linux host available; Android toolchain still not found |

Three read-only `wsl.exe -d Debian --exec sh -lc` checks each exited 0. The actual target is Debian 13.6 (trixie), Linux `6.18.33.2-microsoft-standard-WSL2`, x86_64, user `root` with home `/root`. The invoked shell is `/usr/bin/dash`; the default shell is `/bin/bash`.

Native Go, Java/javac, adb, sdkmanager, avdmanager, emulator and Nix were not found on the Linux PATH. `ANDROID_HOME`, `ANDROID_SDK_ROOT`, `ANDROID_NDK_HOME` and `JAVA_HOME` are unset; checked standard SDK/JDK/Go directories and installed-package candidates did not reveal the missing toolchain. Native GCC 14.2.0 is available as `/usr/bin/cc`, and the Linux filesystem reports about 934 GiB free.

A native Node v24.21.0 binary exists at `/opt/edgeterm-node-v24.21.0/bin/node` and its npm package metadata reports 11.19.0; this bundle is outside PATH. The PATH-resolved `npm` instead comes from the Windows installer under `/mnt/d/Dev/Node.js` and reports 11.6.2. Windows `node.exe`, `go.exe`, `java.exe`, `javac.exe`, `adb.exe` and `wails3.exe` are visible through mounted Windows paths; they are not evidence of native Linux build tools. No downloads, installations or emulator/UI launches occurred. The WSL audit does not remove the SDK/NDK/JDK/Go preparation blocker or establish an Android/FFmpegKitNext build.

The ignored pinned scaffold uses Gradle 9.2.1 and Android Gradle Plugin 8.7.3. Gradle's [Java compatibility matrix](https://docs.gradle.org/9.2.1/userguide/compatibility.html) supports running Gradle on Java 25 starting with Gradle 9.1. That does not prove the complete Android plugin/build succeeds with the discovered game runtime. Check Java, Gradle and SDK tools in the chosen build environment before changing versions.

## Repository staging and actual attempt

The root `Taskfile.yml` intentionally has no Android include: including the upstream Android tasks caused their POSIX variables to run during ordinary Windows builds. `build/android/Taskfile.yml` is an upstream task reference only. The full Android Gradle project, Java host, Gradle wrapper, NDK source layout, helper scripts and common build assets are present only in the ignored `_scaffold` inspection copy; they have not been staged into the repository. A fresh checkout is not a ready Android Gradle application.

The Android task attempt exited with code 127 after `uname` / `tail` were unavailable in the native Windows task shell. The SDK, NDK and emulator were also absent from the environment audit. This attempt did not compile `libwails.so` or assemble an APK. Repeating the reference task on Linux without staging the missing assets would still leave the project incomplete.

The successful `GOOS=android`, `GOARCH=arm64`, `CGO_ENABLED=0` check establishes shared Go package compatibility only. A real Wails Android app requires cgo and the NDK. The generated local `artifacts/android/overlay.json` and `artifacts/android/gen/main_android.gen.go` add `RegisterAndroidMain(main)` for a later native build; their generation is not an APK or runtime test. These generated artifacts are outside the committed source baseline.

## Next executable shell milestone

1. Inspect Debian's OS, shell, Go, Node/npm, Java, Android SDK/NDK, disk space and device connectivity. Windows binaries and environment variables must not be mistaken for Linux toolchain installations.
2. Generate the full build assets with the pinned Wails beta.27 CLI into a separate staging directory on that host, then review and integrate the Android Gradle/Java/wrapper/scripts and required common tasks. The verified generator has `wails3 generate build-assets -dir <staging-directory>`; do not overwrite this repository's Windows build configuration with a new generic template. Bind the staged Android build to this repository's shared Go source and React assets, and keep Android task evaluation separate from the Windows root task.
3. Establish SDK platform 35, build-tools 35.0.0, NDK 26.3.11579264, platform-tools and either a reachable device or emulator. These are Wails shell requirements; FFmpegKitNext uses a separate native build baseline. Run `wails3 doctor` in that environment.
4. Inspect the prepared Taskfile and its working-directory assumptions before building. The pinned CLI help verifies `wails3 task -taskfile <prepared-Taskfile.yml> <task-name>` for an explicit task entrypoint; this flag does not prepare Gradle assets. With a fully staged root task that includes Android, the task names are `android:build`, `android:run` and `android:run:device`. A direct Android Taskfile uses unprefixed names such as `build`; its relative includes and command directory must also be prepared. The repository's current root Taskfile exposes none of these Android tasks.
5. Build the cgo shared library and assemble an APK, then launch it. Record device/ABI/API level and logcat. Demonstrate a real React service call to `GetPlatform()` returning `android` and a Go event delivered to React.
6. Confirm the same Media Card and Compress components render at phone size in the APK. A desktop browser's phone viewport checks responsive layout only.

No SDK download, emulator creation or WSL installation is claimed here. A narrowly scoped Windows build adapter is another possible route, but must cover NDK host tags, executable extensions, path discovery and Gradle invocation and then be tested. Switching Android shell is a separate architectural decision, not an automatic response to missing tools.

## Acceptance record

| Requirement | Evidence available | Current acceptance |
| --- | --- | --- |
| Wails Android host and JS-to-Go support | Official documentation and beta.27 source inspected | Source support confirmed; local runtime unverified |
| Shared Go Core compiles for Android arm64 | cgo-disabled cross-compile succeeded | Go package compile check passed; native runtime unverified |
| Android main overlay | Overlay generation succeeded | Initialization source generated; cgo shared-library build not run successfully |
| Shared React frontend | Shared components in the repository | Android APK build/rendering verification pending |
| `GetPlatform()` returns `android` | Required service contract | Device/emulator call not run |
| Android Gradle project prepared in repository | Only upstream task reference retained; full assets remain in ignored scaffold | Not prepared |
| Android APK builds | Attempt exited 127 on Windows `uname` / `tail`; SDK/NDK not found | Failed before native compile; no APK produced |
| APK starts on emulator/device | No device; emulator not found | Not run |
| Shared Media Card and Compress page in Android | Shared frontend source | APK rendering not run |
| Native probe, encode, progress, cancel, output | Research in [Android FFmpeg spike](android-ffmpeg-spike.md) | Deferred until the shell spike passes |

The first-spike Android runner reports unsupported native execution explicitly. It does not claim successful probe/encode or generate simulated job results. The frontend can share models and layout while native actions remain unavailable.

## Storage and lifecycle boundary

`Source` already distinguishes `local-file` from `content-uri`. Keep the original document reference in the versioned project model. Platform-owned URI grants, file descriptors and temporary `ffkitsaf:` registrations belong in the Android adapter.

Wails' built-in open picker imports SAF documents as cache copies. It does not provide the direct URI and export behavior required by the native FFmpeg milestone; save-file and directory dialogs currently return errors. Use a native document picker/export adapter for large media and Android output. The details and alternatives are recorded in [android-ffmpeg-spike.md](android-ffmpeg-spike.md).
