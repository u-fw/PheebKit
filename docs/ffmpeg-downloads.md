# Windows FFmpeg setup

The application downloads only on an explicit click in Settings. It never installs a package manager, clones Git repositories, requires elevation, changes system PATH, or upgrades an existing FFmpeg automatically. Automatic download is supported only on Windows amd64. Existing local selection remains available offline.

## Catalog

| Selection | Package | Transport sources |
| --- | --- | --- |
| Stable | Gyan 9.0.2 full static Windows build, approximately 246 MiB ZIP | Versioned gyan.dev package and matching GyanD/codexffmpeg GitHub release |
| Domestic fallback | Gyan-derived FFmpeg/ffprobe 6.1.1 essentials, approximately 56 MiB gzip downloads | npmmirror binary mirror of eugeneware/ffmpeg-static b6.1.1 |
| Automatic | Stable first, then domestic fallback on failure | Same packages and checksums as above |

The domestic package is intentionally labeled older. It is not treated as a byte-identical mirror of 9.0.2. Capabilities are rediscovered after installation; a missing software AV1 encoder or unusable GPU encoder is not fabricated. The installer retains upstream license/build/source notices and records download provenance. Downloading a package does not relicense it under kompeg's license.

Pinned SHA-256 values live in `internal/ffmpegsetup/catalog.go`. The three executable-archive digests were compared against GitHub release asset metadata; domestic README/LICENSE digests were recorded separately. Runtime downloads cannot override trusted digests or provide arbitrary URLs. Catalog updates require a reviewed code change.

Upstream references: [Gyan builds](https://www.gyan.dev/ffmpeg/builds/), [Gyan 9.0.2 release](https://github.com/GyanD/codexffmpeg/releases/tag/9.0.2), [ffmpeg-static b6.1.1](https://github.com/eugeneware/ffmpeg-static/releases/tag/b6.1.1), [npmmirror files](https://registry.npmmirror.com/-/binary/ffmpeg-static/b6.1.1/).

## Behavior and boundaries

- Stable-source ranking downloads at most 64 KiB per candidate under a five-second probe deadline. Actual transfer failure tries the remaining source; automatic mode then tries the domestic package. It does not claim that one source is fastest for every mainland ISP.
- HTTPS and an explicit redirect-host allowlist are used in production. Transfers have bounded header, inactivity and total deadlines. A SHA-256 mismatch fails before decompression or execution.
- Downloads are bounded to 400 MiB per archive; ZIP expansion is bounded to 1 GiB. Traversal, symlinks, special files, multiple FFmpeg copies and missing sibling ffprobe are rejected.
- Files are staged under the application's private cache directory. Cancellation and failure remove staging. A successful installation gets a unique directory; existing installs are never overwritten or automatically removed.
- Configuration is switched only after actual `-version` validation of both binaries and the existing idle-job guard. Higher-priority explicit environment overrides must be corrected before automatic setup. Cancellation stops before the configuration commit; the final validation/save step is not cancellable.
- Progress is queried from the Go service, so navigating away from Settings does not lose the download. Closing the application cancels the transfer. On success, the UI reloads the environment and runs actual CPU/GPU qualification.
- The app downloads directly from upstream/mirror endpoints. Any future project-hosted domestic replica or offline installer still needs its own complete third-party distribution/source-notice review.

## Verification on 2026-10-10

`KOMPEG_SETUP_INTEGRATION=1 go test ./internal/app -run '^TestRealFFmpegSetup$' -count=1 -v` completed real downloads of both packages in isolated temporary Windows profiles. SHA verification, extraction, actual FFmpeg/ffprobe execution, saved private paths, encoder scanning, duplicate-start rejection and configuration reuse by a new service instance passed.

Deterministic tests cover package fallback, preserved upstream license files, failed-hash rejection, traversal rejection, cancellation during transfer, clean staging, catalog restrictions and redirect restrictions. Frontend rendering tests cover determinate/indeterminate progress, cancellation availability, version warnings, error reporting and retry.

Production Wails build, Go tests/vet and 42 frontend tests/typecheck/lint/format passed. Launching an isolated native UI verification process was rejected by automatic approval review with `blocked by policy`; native click-through and screenshot verification of this new setup flow remain unverified.
