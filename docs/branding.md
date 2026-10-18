# 菲比工具箱 / PheebKit

The owner selected the Chinese name **菲比工具箱** and English brand **PheebKit** on 2026-10-10. Version 0.1.0-beta.3 applies the name to the interface, browser/native title, Windows executable metadata, installer/shortcuts and release filenames.

`frontend/public/pheebkit.png` is the owner-supplied artwork, preserved byte-for-byte. Its SHA-256 is `163d1e2675600072a0399e0c06c22d0847333347ad3fc15c5e0aad4a7682e3fc`. The existing pinned Wails icon generator produces `build/windows/icon.ico` at 16, 32, 48, 64, 128 and 256 pixels. No new image service or dependency is used.

Current public names use `pheebkit` / `PHEEBKIT_*`, including the Go module, build constants and Windows manifest. New installs use `%APPDATA%/PheebKit` and `%LOCALAPPDATA%/Programs/PheebKit`. Installer AppId and isolated legacy data/env/mutex aliases remain compatible. See [migration rules](compatibility.md) and [interface languages](i18n.md). The current application is `bin/PheebKit.exe`; older immutable artifacts remain historical evidence.

The media summary labels YUV 4:2:0, bit depth and BT.709 in readable terms; full matrix/transfer/primaries values remain in the video's tooltip. `und`/missing track language is displayed as “语言未标注”, and absent subtitles as “未检测到内嵌字幕”. No claim is made about externally supplied subtitles or subtitles burned into the video.

Browser verification checked the supplied logo in the complete app shell and time selectors in a synthetic-media component fixture. Arrow/Enter/Escape, fractional-value retention, ±0.1 seconds and the final allowed second were operated. Light/dark and 375px / 200% text-size layouts were checked. A temporary fixture header's extra buttons initially overflowed; wrapping that test-only header confirmed the actual time-picker component has no horizontal overflow. Native click-through of beta.3 remains unverified.
