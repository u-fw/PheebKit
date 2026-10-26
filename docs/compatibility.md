# Naming migration and compatibility

Current branding is **菲比工具箱 / PheebKit**. Since 0.2.0-rc.3, the Go module is `pheebkit`, the frontend version constant is `__PHEEBKIT_VERSION__`, public environment variables use `PHEEBKIT_*`, and the Windows manifest identity is `studio.pheebkit.app`. The integration flag is `PHEEBKIT_INTEGRATION=1`; its former spelling is no longer used.

| Current identity | Existing installation behavior |
| --- | --- |
| `PHEEBKIT_FFMPEG`, `PHEEBKIT_FFPROBE` | Old `KOMPEG_FFMPEG` / `KOMPEG_FFPROBE` remain runtime aliases. Priority: current variable → legacy alias → generic `FFMPEG_PATH` / `FFPROBE_PATH` → saved configuration → bundled runtime → PATH. Explicit overrides still block redundant downloading. |
| `%APPDATA%/PheebKit` | Used for new installations. If absent and `%APPDATA%/kompeg` exists, the original directory is reused in place. No data is moved, copied or deleted. If both exist, the current directory wins. |
| `%LOCALAPPDATA%/PheebKit/tools` | New private downloads. Previously configured valid tools remain usable at their existing paths. |
| `pheebkit-theme`, `pheebkit-language` | New preferences use these keys. Theme reads the old `kompeg-theme` fallback; subsequent writes use the current key. |
| `Local\PheebKit-application-running` | Native windows also expose the old mutex. The installer checks both, preventing upgrades or removal while an old or new window is alive. |
| Installer AppId `{9D8FAD06-8029-4A58-9497-96C7DA878C63}` | Unchanged so upgrades reuse registration and installation directory. Uninstall retains user-owned files and configuration. |

`internal/compat`, `frontend/src/legacy.ts`, `scripts/legacy.ps1` and `build/windows/legacy.iss` isolate old identities. Tests and this document name them to prove compatibility. Persisted source-fingerprint domain bytes also remain unchanged: these are opaque data identifiers, not displayed branding. Changing them would invalidate saved source-bound recovery decisions.

Historical verification records and immutable older artifacts retain their original names. An existing checkout directory does not need renaming. `scripts/check-locales.mjs` rejects old environment names outside the compatibility boundary and runs with local/hosted frontend tests.
