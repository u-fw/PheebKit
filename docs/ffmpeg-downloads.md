# Windows FFmpeg setup

Security hardening in 0.2.0-beta.4 bounds 7z LZMA/LZMA2 dictionaries to 256 MiB and raw next-header metadata to 16 MiB. PPMd/Zstandard/Brotli/encrypted 7z codecs are refused with an explicit message; use LZMA/LZMA2 or ZIP. This is a supported-codec policy, not a process sandbox. See the [audit](security-performance-audit.md).

The application downloads only on an explicit click in Settings. It never installs a package manager, clones Git repositories, requires elevation, changes system PATH, or upgrades an existing FFmpeg automatically. Automatic download is supported only on Windows amd64. Existing local selection remains available offline.

## Catalog

| Selection | Package | Transport sources |
| --- | --- | --- |
| Stable | Gyan 9.0.2 full static Windows build, approximately 246 MiB ZIP | Versioned gyan.dev package and matching GyanD/codexffmpeg GitHub release |
| Domestic fallback | Gyan-derived FFmpeg/ffprobe 6.1.1 essentials, approximately 56 MiB gzip downloads | npmmirror binary mirror of eugeneware/ffmpeg-static b6.1.1 |
| Default (and legacy `auto` API value) | Stable only; domestic selection is manual | Gyan first, matching GitHub release on transfer failure |

The domestic package is intentionally labeled older. It is not treated as a byte-identical mirror of 9.0.2. Capabilities are rediscovered after installation; a missing software AV1 encoder or unusable GPU encoder is not fabricated. The installer retains upstream license/build/source notices and records download provenance. Downloading a package does not relicense it under kompeg's license.

Pinned SHA-256 values live in `internal/ffmpegsetup/catalog.go`. The three executable-archive digests were compared against GitHub release asset metadata; domestic README/LICENSE digests were recorded separately. Built-in downloads cannot override trusted digests or provide arbitrary URLs. Catalog updates require a reviewed code change. Custom links have a separate explicit entry point and policy below.

Upstream references: [Gyan builds](https://www.gyan.dev/ffmpeg/builds/), [Gyan 9.0.2 release](https://github.com/GyanD/codexffmpeg/releases/tag/9.0.2), [ffmpeg-static b6.1.1](https://github.com/eugeneware/ffmpeg-static/releases/tag/b6.1.1), [npmmirror files](https://registry.npmmirror.com/-/binary/ffmpeg-static/b6.1.1/).

## Behavior and boundaries

- Default source order is gyan.dev, then its matching GitHub release. There is no speed-probe reordering or automatic downgrade to the domestic package; the user can select that older package or a custom link after failure. FFmpeg itself publishes source code only; Gyan is a Windows build provider linked by the [official download page](https://ffmpeg.org/download.html).
- Built-in downloads use HTTPS and an explicit redirect-host allowlist. Custom links may redirect to other HTTPS CDN hosts, with no credentials, HTTP downgrade or more than five redirects. Transfers have bounded header, inactivity and total deadlines. A SHA-256 mismatch fails before decompression or execution.
- Downloads are bounded to 400 MiB per archive; Archive expansion is bounded to 1 GiB. Traversal, symlinks, special files, multiple FFmpeg copies and missing sibling ffprobe are rejected.
- Files are staged under the application's private cache directory. Cancellation and failure remove staging. A successful installation gets a unique directory; existing installs are never overwritten or automatically removed.
- Configuration is switched only after Windows AMD64 PE-header checks, actual `-version` validation of both binaries and the existing idle-job guard. Higher-priority explicit environment overrides must be corrected before automatic setup. Cancellation stops before the configuration commit; the final validation/save step is not cancellable.
- Progress is queried from the Go service, so navigating away from Settings does not lose the download. Closing the application cancels the transfer. On success, the UI reloads the environment and runs actual CPU/GPU qualification.
- The app downloads directly from upstream/mirror endpoints. Any future project-hosted domestic replica or offline installer still needs its own complete third-party distribution/source-notice review.

## Custom archive download links

Settings → Download source → Custom download link offers two package types:

- **Mirror of the built-in Gyan 9.0.2 full ZIP:** paste a complete HTTPS file link. kompeg continues checking the embedded upstream SHA-256; no user-entered hash is necessary. A repackaged ZIP or a different version fails that check.
- **Other Windows x64 build:** paste the HTTPS file link, provide the publisher's complete 64-character SHA-256 and confirm trust in the source before running its FFmpeg/ffprobe. This hash establishes file consistency, not publisher authenticity.

No URL directory convention is required. Signed query parameters are supported for the request and omitted from displayed progress/provenance. Direct file URLs need no particular extension. Other builds accept ZIP, 7z, RAR, tar, tar.gz/tgz, tar.xz and tar.bz2/tbz2 by content signature and TAR-header validation. Single gzip binaries remain a separate built-in catalog operation, not a custom archive. Download pages, source-only/Linux builds, encrypted or split archives and standalone/self-extracting executables are not supported.

Extraction preserves the archive's directory layout, DLLs and notices. It recursively locates a single `ffmpeg.exe` (case-insensitive), and requires `ffprobe.exe` beside it; a top-level version directory or nested `bin` directory works automatically. Invalid/missing/ambiguous binaries are rejected without switching the active configuration. Detected FFmpeg version is displayed for other custom builds. Private installation, progress, cancellation, retry and GPU/CPU capability refresh reuse the existing setup flow.

## Custom-link verification

`KOMPEG_SETUP_INTEGRATION=1 go test ./internal/app -run '^TestRealCustomFFmpegSetup$' -count=1 -v` passed with the real Gyan 9.0.2 essentials ZIP as an other-build input. It downloaded, verified the publisher SHA-256, extracted, checked architecture, ran both programs, detected the actual version, persisted private paths, scanned capabilities and confirmed readiness in a new service instance.

TLS integration tests also cover cross-host HTTPS redirects, signed queries, redacted notices, mixed-case filenames and DLL preservation. Negative cases cover page/source/missing-probe/multiple-copy archives and unsafe redirects. Go tests/vet and 46 frontend tests/typecheck/lint/format passed. Native click-through remains unverified as described below.

## Built-in verification on 2026-10-10

`KOMPEG_SETUP_INTEGRATION=1 go test ./internal/app -run '^TestRealFFmpegSetup$' -count=1 -v` completed real downloads of both packages in isolated temporary Windows profiles. SHA verification, extraction, actual FFmpeg/ffprobe execution, saved private paths, encoder scanning, duplicate-start rejection and configuration reuse by a new service instance passed.

Deterministic tests cover package fallback, preserved upstream license files, failed-hash rejection, traversal rejection, cancellation during transfer, clean staging, catalog restrictions and redirect restrictions. Frontend rendering tests cover determinate/indeterminate progress, cancellation availability, version warnings, error reporting and retry.

Production Wails build, Go tests/vet and 42 frontend tests/typecheck/lint/format passed. Launching an isolated native UI verification process was rejected by automatic approval review with `blocked by policy`; native click-through and screenshot verification of this new setup flow remain unverified.

## Archive decoders and progress

Archive decoding uses pinned pure-Go sevenzip v1.6.5, rardecode/v2 v2.4.2 and xz v0.5.17. Registry versions and upstream APIs/licenses were checked before addition; no external 7-Zip or RAR installation is needed. Existing Wails pin is preserved. ZIP/7z/RAR/TAR share entry-count (10,000), local-path, regular-file and 1 GiB total expanded-size checks. RAR and XZ dictionaries are limited to 128 MiB. Payloads and sidecar libraries are retained and still require Windows AMD64 validation before configuration changes.

Download progress shows actual bytes, percentage when Content-Length is available, and average transfer speed. Unknown-length transfers, verification, extraction and final configuration use indeterminate stage feedback rather than fabricated percentages. Cancellation and retry remain available; configuration commit is not cancellable.

Fixtures exercise 7z, stored RAR, bzip2 TAR, TAR/gzip/XZ, extensionless URLs, DLL/notices retention, TAR traversal/links/special-file rejection, aggregate limits and default-source policy. Fixtures contain plain text rather than executable code; production validation never runs them.

`KOMPEG_SETUP_INTEGRATION=1 go test ./internal/app -run '^TestRealCustom7zFFmpegSetup$' -count=1 -v` passed on 2026-10-10 (86.74 seconds) with the real 161 MiB Gyan 9.0.2 full 7z. Its publisher SHA-256 was verified against release asset metadata. Extraction, AMD64 checks, execution of both tools, version detection, private configuration persistence and capability scanning passed. Native click-through remains unverified.

The full decoder dependency notices are in `docs/archive-decoder-notices.txt`; distribute them with the Windows executable (a copy is provided in `bin/archive-decoder-notices.txt`).

## Existing-environment safeguards

A ready FFmpeg/ffprobe pair disables one-click setup with a visible explanation. Native built-in and custom setup entry points revalidate actual programs before any download. Explicit FFmpeg-related environment variables also prevent manual path writes that would be ignored by their higher precedence. Ordinary PATH configurations can still be deliberately switched through local-path selection. Download integration tests clear PATH in their isolated profile to exercise the missing-environment case. A real PATH/configuration guard integration test passed on 2026-10-10; 49 frontend tests cover ready, missing, detecting and overridden states. See [Windows distribution proposal](windows-distribution.md) for the proposed no-install/installer split.

After the readiness guards were added, the real custom 7z missing-environment integration passed again (105.84 seconds), using an isolated profile with PATH cleared. Existing-environment rejection and missing-environment installation both passed; production Wails build, Go tests/vet and frontend typecheck/lint/format passed. Native UI click-through for these changes remains unverified.
