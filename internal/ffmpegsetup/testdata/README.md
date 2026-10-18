# Archive fixtures

These archives were generated for PheebKit and contain four plain-text files:

- `release/bin/ffmpeg.exe`: `binary`
- `release/bin/ffprobe.exe`: `probe`
- `release/bin/avcodec.dll`: `sidecar`
- `release/LICENSE`: `license`

They are decoder fixtures, not executable programs. The 7z fixture was created with 7-Zip 26.03 (LZMA2, solid). The bzip2 TAR was generated with Python's standard-library tarfile/bz2. The RAR fixture uses RAR4 stored file records with CRC32 for headers and payloads and was independently checked with `7z t`. No third-party fixture content is included. ZIP, TAR, gzip and XZ fixtures are created within the Go tests.
