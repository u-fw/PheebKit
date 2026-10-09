# Project and preset format

Projects use UTF-8 JSON in `.ffproject` files. The Go model is the source of truth; Wails generates the TypeScript interfaces used by the UI bridge.

```json
{
  "schemaVersion": 1,
  "id": "project-id",
  "inputs": [{
    "uri": "C:/Videos/movie.mkv",
    "displayName": "movie.mkv",
    "kind": "local-file"
  }],
  "output": {
    "uri": "C:/Videos/movie-hevc.mp4",
    "displayName": "movie-hevc.mp4",
    "kind": "local-file"
  },
  "settings": {
    "codec": "hevc",
    "quality": 26,
    "preset": "medium",
    "resolution": 0,
    "audio": "aac",
    "mode": "quality",
    "bitrateKbps": 6000,
    "targetSizeMB": 100,
    "rawArguments": []
  }
}
```

`resolution: 0` preserves the input dimensions; positive even heights express the output intent. Codec families are `h264`, `hevc`, `av1`. Audio is `copy`, `aac`, `opus`. Modes are `quality`, `bitrate`, `size`. CRF is codec-dependent. Target size uses decimal MB (1,000,000 bytes) with a one-pass video bitrate budget and muxing allowance; it is not an exact size promise.

Android inputs/outputs retain `kind: content-uri` and the original document URI. Do not save temporary cache paths as substitutes for the URI or persist per-session `ffkitsaf:` handles. The spike rejects URI execution until a native adapter is integrated.

No final command string is persisted. Probe data and Execution Plan are derived. Raw arguments remain separate argv elements. `internal/project` supplies size-limited, strict load/save functions; unknown versions and fields fail explicitly. A migration must be added before introducing schema version 2. The first UI does not yet expose project save/load or recent-project browsing.

Presets carry `schemaVersion`, `id`, `name` and the same settings object. The first package contains built-in quality presets. User preset import/export and migration UI are later scope.
