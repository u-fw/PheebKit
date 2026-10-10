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
    "encoder": "auto",
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

`resolution: 0` preserves the input dimensions; positive even heights express the output intent. Codec families are `h264`, `hevc`, `av1`. `encoder: "auto"` prefers runtime-qualified GPU encoders, `"cpu"` selects software, and a specific encoder name requests that tested GPU encoder. Legacy files omitting `encoder` retain CPU behavior. This optional additive field stays in schema version 1; older strict readers will reject files containing it and need updating.

Audio is `copy`, `aac`, `opus`. Modes are `quality`, `bitrate`, `size`. The quality value maps to CRF on CPU, CQ on NVENC, ICQ on QSV and CQP on AMF; equivalent numeric values do not promise equal visual quality across encoders. Target size uses decimal MB (1,000,000 bytes) with a one-pass video bitrate budget and muxing allowance; it is not an exact size promise. Media with audio must use AAC/Opus in size mode to supply a known audio bitrate budget. For example, 100 MB is 95.37 MiB, the binary unit used in the size display.

Android inputs/outputs retain `kind: content-uri` and the original document URI. Do not save temporary cache paths as substitutes for the URI or persist per-session `ffkitsaf:` handles. The spike rejects URI execution until a native adapter is integrated.

Optional `trim: { "startSeconds": 30.5, "endSeconds": 70.5 }` retains a single range relative to the source beginning. Missing/null processes the entire input. Trim requires a known source duration and AAC/Opus for media containing audio. The compiler rejects invalid boundaries and known shorter-than-frame ranges; progress and target-size budgets use the selected duration. This additive schema-1 field is preserved by current load/save functions; older strict readers must be updated. See [trimming](trimming.md).

No final command string is persisted. Probe data and Execution Plan are derived. Raw arguments remain separate argv elements. `internal/project` supplies size-limited, strict load/save functions; unknown versions and fields fail explicitly. A migration must be added before introducing schema version 2. The first UI does not yet expose project save/load or recent-project browsing.

Presets carry `schemaVersion`, `id`, `name` and the same settings object. The first package contains built-in quality presets. User preset import/export and migration UI are later scope.
