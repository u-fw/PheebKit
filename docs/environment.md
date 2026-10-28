# Development environment audit — 2026-10-09

The 2026-10-13 security update pinned Go 1.27.2 with Node 24; the later GO-2026-6809 correction supersedes that minimum with Go 1.27.3. The dated observations below describe the original spike environment.

## Current baseline — 2026-10-26

The production desktop shell is stable Wails v2.17.0 with React 19 and strict TypeScript. The module selects Go 1.27.3 and release packaging rejects older toolchains because of [GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809). Wails v3 Beta and its npm runtime are historical Technical Spike tooling only and are not active dependencies of the Windows application. Android remains deferred; a future Android client may use native Jetpack Compose while the Windows UI continues to express Material 3 / Material You through React.

Current local verification uses Node 24, the lockfile-installed frontend dependencies, Windows WebView2 and the repository-verified FFmpeg/ffprobe 9.0.2 pair. The current source passes 159 frontend tests in 34 files, 1,419 translation entries, project-scoped Go real-media integration, vet and source vulnerability checks. The dependency-triggered and weekly `.github/workflows/security.yml` audit repeats lockfile installation without lifecycle scripts, npm audit, pinned `govulncheck`, and pinned workflow linting. Clean Windows 10/11 and CPU-only machines remain release gates rather than assumptions.

Native Windows 11 Pro 25H2 (build 26200), PowerShell 7.6.5, amd64. Repository started empty with an unborn `main` branch; stabilization work is on `codex/stable-wails2`.

| Tool                         | Observed result                                                                                                                                                                                                                       |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Go                           | 1.27.1; Wails module requires Go 1.25+                                                                                                                                                                                                |
| Node / npm                   | 24.11.1 / 11.6.2                                                                                                                                                                                                                      |
| Wails existing               | v2.16.0                                                                                                                                                                                                                               |
| Wails spike installed        | v3.0.0-beta.27; explicitly approved for this spike                                                                                                                                                                                    |
| WebView2                     | 154.0.4258.62; Wails Doctor passes                                                                                                                                                                                                    |
| FFmpeg / ffprobe             | 9.0.2 full Gyan build; installed through Scoop                                                                                                                                                                                        |
| JDK / javac                  | Microsoft OpenJDK 25.0.1; JAVA_HOME points to Minecraft runtime                                                                                                                                                                       |
| adb                          | 37.0.1; no device attached                                                                                                                                                                                                            |
| Android SDK / NDK / emulator | Not found; ANDROID_HOME / ANDROID_SDK_ROOT unset                                                                                                                                                                                      |
| WSL                          | Read-only audit passed (exit 0): Debian 13.6 / x86_64 WSL2, dash; native GCC 14.2 and Node 24.21.0 found, SDK/NDK/emulator and native Go/JDK not found on PATH or checked standard locations; no Android/FFmpegKitNext build verified |
| Windows packaging            | NSIS, Windows SDK signing/MSIX tools absent; `.exe` spike does not need these                                                                                                                                                         |

Wails beta.28 exists upstream; beta.27 is intentionally pinned together with `@wailsio/runtime@3.0.0-beta.27` for this reproducible spike. No rolling `@latest` runtime dependency is used.

Frontend packages were checked against the official npm registry before installation. React/React DOM 19.3.0, Vite 8.3.4, plugin-react 6.1.2, TypeScript 6.0.3, Vitest 5.0.3 and ESLint 10.12.0 support the installed Node runtime. Current TypeScript 7.0.2 was deliberately not selected because typescript-eslint 8.71.1 declares support below 6.1.0. No toolchain upgrade was required.

The Android build is conditional on a usable toolchain. See [Android audit](android.md) for the exact native-build limitations and versioned sources.
