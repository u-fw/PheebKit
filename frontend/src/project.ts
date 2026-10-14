import type { Media, Project, Settings } from "./bridge";
import { trimError } from "./trim";

export const defaultSettings: Settings = {
  codec: "hevc",
  encoder: "auto",
  quality: 26,
  preset: "medium",
  resolution: 0,
  audio: "copy",
  mode: "quality",
  bitrateKbps: 4000,
  targetSizeMB: 100,
  rawArguments: [],
};

export type Operation =
  "compress" | "remux" | "extract-audio" | "extract-subtitle";
export const operationLabels: Record<Operation, string> = {
  compress: "压缩视频",
  remux: "转换封装",
  "extract-audio": "提取音频",
  "extract-subtitle": "提取字幕",
};

export function operationOf(settings: Settings): Operation {
  return (settings.operation || "compress") as Operation;
}

export function videoDisplaySize(stream: {
  width: number;
  height: number;
  displayRotation?: number;
}) {
  const rotation = (((stream.displayRotation ?? 0) % 360) + 360) % 360;
  return rotation === 90 || rotation === 270
    ? { width: stream.height, height: stream.width }
    : { width: stream.width, height: stream.height };
}

export function selectedTracks(
  media: Pick<Media, "streams">,
  kind: "audio" | "subtitle",
  settings: Settings,
) {
  const tracks = media.streams.filter((stream) => stream.codecType === kind);
  const selection =
    kind === "audio" ? settings.audioSelection : settings.subtitleSelection;
  const mode = selection?.mode ?? (kind === "audio" ? "all" : "none");
  if (kind === "audio" && settings.audio === "none") return [];
  return mode === "all"
    ? tracks
    : mode === "none"
      ? []
      : tracks.filter((stream) => selection?.indices?.includes(stream.index));
}

export function operationExtensions(settings: Settings): string[] {
  switch (operationOf(settings)) {
    case "remux":
      return ["mp4", "mkv", "webm"];
    case "extract-subtitle":
      return ["srt", "ass", "vtt"];
    case "extract-audio": {
      const extensions: Record<string, string> = {
        aac: "m4a",
        opus: "opus",
        mp3: "mp3",
        flac: "flac",
        wav: "wav",
      };
      return [extensions[settings.audio] ?? "mka"];
    }
    default:
      return settings.audio === "opus"
        ? settings.codec === "av1"
          ? ["mkv", "webm"]
          : ["mkv"]
        : settings.codec === "av1"
          ? ["mp4", "mkv", "webm"]
          : ["mp4", "mkv"];
  }
}

export function outputExtension(settings: Settings, preferred = ""): string {
  const extensions = operationExtensions(settings);
  return extensions.includes(preferred.toLowerCase())
    ? preferred.toLowerCase()
    : extensions[0];
}

export function suggestedOperationOutput(
  uri: string,
  settings: Settings,
): string {
  const suffix =
    {
      compress: "compressed",
      remux: "remux",
      "extract-audio": "audio",
      "extract-subtitle": "subtitle",
    }[operationOf(settings)] ?? "processed";
  const stem = uri.startsWith("content://")
    ? "media"
    : uri.replace(/\.[^./\\]+$/, "");
  return `${stem}-${suffix}.${outputExtension(settings)}`;
}

export function changeOperation(
  settings: Settings,
  operation: Operation,
  media: Media,
): Settings {
  if (operationOf(settings) === operation) return settings;
  const firstAudio = media.streams.find(
    (stream) => stream.codecType === "audio",
  );
  const firstSubtitle = media.streams.find(
    (stream) => stream.codecType === "subtitle",
  );
  return {
    ...settings,
    operation,
    trim: null,
    transform: null,
    volumeDB: 0,
    rawArguments: [],
    resolution: operation === "compress" ? settings.resolution : 0,
    audio:
      operation === "remux"
        ? "copy"
        : operation === "extract-audio"
          ? settings.audio === "none"
            ? "copy"
            : settings.audio
          : ["copy", "aac", "opus", "none"].includes(settings.audio)
            ? settings.audio
            : "aac",
    audioSelection:
      operation === "extract-audio"
        ? { mode: "selected", indices: firstAudio ? [firstAudio.index] : [] }
        : operation === "extract-subtitle"
          ? { mode: "none", indices: [] }
          : settings.audioSelection,
    subtitleSelection:
      operation === "extract-subtitle"
        ? {
            mode: "selected",
            indices: firstSubtitle ? [firstSubtitle.index] : [],
          }
        : operation === "extract-audio"
          ? { mode: "none", indices: [] }
          : settings.subtitleSelection,
  };
}

export function settingsForMedia(settings: Settings, media: Media): Settings {
  const operation = operationOf(settings);
  if (operation === "extract-audio" && !settings.audioSelection) {
    const stream = media.streams.find((track) => track.codecType === "audio");
    return {
      ...settings,
      audioSelection: {
        mode: "selected",
        indices: stream ? [stream.index] : [],
      },
      subtitleSelection: { mode: "none", indices: [] },
    };
  }
  if (operation === "extract-subtitle" && !settings.subtitleSelection) {
    const stream = media.streams.find(
      (track) => track.codecType === "subtitle",
    );
    return {
      ...settings,
      subtitleSelection: {
        mode: "selected",
        indices: stream ? [stream.index] : [],
      },
      audioSelection: { mode: "none", indices: [] },
    };
  }
  return settings;
}

export function applyPresetSettings(
  current: Settings,
  preset: Settings,
  media: Media,
): Settings {
  const requested = operationOf(preset);
  const audioSelection =
    preset.audioSelection &&
    ["all", "none"].includes(preset.audioSelection.mode)
      ? preset.audioSelection
      : current.audioSelection;
  const subtitleSelection =
    preset.subtitleSelection &&
    ["all", "none"].includes(preset.subtitleSelection.mode)
      ? preset.subtitleSelection
      : current.subtitleSelection;
  if (requested !== operationOf(current))
    return settingsForMedia(
      {
        ...preset,
        trim: null,
        audioSelection:
          requested === "extract-audio"
            ? null
            : requested === "extract-subtitle"
              ? { mode: "none", indices: [] }
              : audioSelection,
        subtitleSelection:
          requested === "extract-subtitle"
            ? null
            : requested === "extract-audio"
              ? { mode: "none", indices: [] }
              : subtitleSelection,
        transform:
          requested === "compress" && preset.transform
            ? { ...preset.transform, crop: null }
            : null,
        resolution: requested === "compress" ? preset.resolution : 0,
        volumeDB:
          requested === "compress" || requested === "extract-audio"
            ? preset.volumeDB
            : 0,
        rawArguments: requested === "compress" ? preset.rawArguments : [],
      },
      media,
    );
  return settingsForMedia(
    {
      ...preset,
      trim: current.trim,
      audioSelection,
      subtitleSelection,
      transform:
        requested === "compress"
          ? {
              rotation: preset.transform?.rotation ?? 0,
              flipHorizontal: preset.transform?.flipHorizontal ?? false,
              flipVertical: preset.transform?.flipVertical ?? false,
              crop: current.transform?.crop ?? null,
            }
          : null,
    },
    media,
  );
}

export function validateSettings(
  settings: Settings,
  outputPath: string,
  inputPath: string,
  media?: Pick<Media, "streams" | "durationSeconds">,
): string[] {
  const errors: string[] = [];
  const operation = operationOf(settings);
  if (!(operation in operationLabels)) errors.push("请选择有效的处理任务。");
  if (!outputPath.trim()) errors.push("请选择输出文件。");
  if (
    outputPath.trim().replace(/\\/g, "/").toLowerCase() ===
      inputPath.trim().replace(/\\/g, "/").toLowerCase() &&
    inputPath
  )
    errors.push("输出文件不能覆盖源文件，请选择新路径。");
  if (media) {
    for (const kind of ["audio", "subtitle"] as const) {
      if (
        (operation === "extract-audio" && kind === "subtitle") ||
        (operation === "extract-subtitle" && kind === "audio")
      )
        continue;
      const selection =
        kind === "audio" ? settings.audioSelection : settings.subtitleSelection;
      if (selection?.mode === "selected" && !selection.indices?.length)
        errors.push(
          `请选择至少一条${kind === "audio" ? "音轨" : "字幕轨"}，或改为不保留。`,
        );
      if (
        selection?.indices?.some(
          (index) =>
            !media.streams.some(
              (stream) => stream.index === index && stream.codecType === kind,
            ),
        )
      )
        errors.push(
          `所选${kind === "audio" ? "音轨" : "字幕轨"}不属于当前媒体，请重新选择。`,
        );
    }
    if (
      operation === "extract-audio" &&
      selectedTracks(media, "audio", settings).length !== 1
    )
      errors.push("提取音频需要选择一条音轨。");
    if (
      operation === "extract-subtitle" &&
      selectedTracks(media, "subtitle", settings).length !== 1
    )
      errors.push("提取字幕需要选择一条字幕轨。");
  }
  const volume = settings.volumeDB ?? 0;
  const audioBitrate = settings.audioBitrateKbps ?? 0;
  if (
    (operation === "compress" || operation === "extract-audio") &&
    (settings.audio === "aac" || settings.audio === "opus") &&
    (!Number.isInteger(audioBitrate) ||
      (audioBitrate !== 0 && (audioBitrate < 32 || audioBitrate > 320)))
  )
    errors.push("音频码率应为 32–320 kbps，或留空使用默认值。");
  if (!Number.isFinite(volume) || volume < -60 || volume > 20)
    errors.push("音量调整必须在 -60 至 20 dB 之间。");
  if (
    volume !== 0 &&
    settings.audio === "copy" &&
    (!media || selectedTracks(media, "audio", settings).length)
  )
    errors.push("调整音量需要重新编码音频，请改为 AAC、Opus 或其他音频编码。");
  if (
    (operation === "remux" || operation === "extract-subtitle") &&
    (settings.trim ||
      (settings.transform &&
        (settings.transform.rotation !== 0 ||
          settings.transform.flipHorizontal ||
          settings.transform.flipVertical ||
          settings.transform.crop)) ||
      volume !== 0 ||
      settings.rawArguments.length)
  )
    errors.push("此任务保持原始内容，请先清除剪辑、画面调整、音量和附加参数。");
  const transform = settings.transform;
  if (transform) {
    if (![0, 90, 180, 270].includes(transform.rotation))
      errors.push("旋转角度请选择 0、90、180 或 270°。");
    if (transform.crop) {
      const margins = Object.values(transform.crop);
      if (
        margins.some(
          (value) =>
            !Number.isInteger(value) ||
            value < 0 ||
            value > 16384 ||
            value % 2 !== 0,
        )
      )
        errors.push("裁剪边距必须为 0–16384 的偶数像素。");
      const video = media?.streams.find(
        (stream) =>
          stream.codecType === "video" &&
          !stream.attachedPicture &&
          !stream.timedThumbnails,
      );
      if (
        video &&
        (videoDisplaySize(video).width -
          transform.crop.left -
          transform.crop.right <
          2 ||
          videoDisplaySize(video).height -
            transform.crop.top -
            transform.crop.bottom <
            2)
      )
        errors.push("裁剪后宽度和高度至少需要保留 2 像素。");
    }
  }
  if (operation !== "compress" && settings.resolution !== 0)
    errors.push("调整分辨率仅适用于压缩任务。");
  if (operation !== "compress") return errors;
  if (settings.trim) {
    const video = media?.streams.find(
      (stream) =>
        stream.codecType === "video" &&
        !stream.attachedPicture &&
        !stream.timedThumbnails,
    );
    const rangeError = trimError(
      settings,
      media?.durationSeconds ?? 0,
      video?.frameRate,
    );
    if (rangeError) errors.push(rangeError);
    if (
      settings.audio === "copy" &&
      media &&
      selectedTracks(media, "audio", settings).length > 0
    )
      errors.push("剪辑需要将音频编码为 AAC 或 Opus，以保持起止时间准确。");
  }
  const maxQuality = settings.codec === "av1" ? 63 : 51;
  if (
    settings.mode === "quality" &&
    (!Number.isInteger(settings.quality) ||
      settings.quality < 0 ||
      settings.quality > maxQuality)
  )
    errors.push(`质量参数必须是 0–${maxQuality} 的整数。`);
  if (
    !Number.isInteger(settings.resolution) ||
    settings.resolution < 0 ||
    settings.resolution > 16384 ||
    settings.resolution % 2 !== 0
  )
    errors.push(
      "输出高度必须为不超过 16384 的正偶数，或设为 0 保持原始分辨率。",
    );
  if (
    settings.mode === "bitrate" &&
    (!Number.isFinite(settings.bitrateKbps) || settings.bitrateKbps <= 0)
  )
    errors.push("目标视频码率必须大于 0 kbps。");
  if (
    settings.mode === "size" &&
    (!Number.isFinite(settings.targetSizeMB) || settings.targetSizeMB <= 0)
  )
    errors.push("目标文件大小必须大于 0 MB。");
  if (
    settings.mode === "size" &&
    media &&
    (!Number.isFinite(media.durationSeconds) || media.durationSeconds <= 0)
  )
    errors.push("目标体积需要已知的视频时长，请换用质量优先模式。");
  const audioError = targetSizeAudioError(settings, media);
  if (audioError) errors.push(audioError);
  return errors;
}

export function targetSizeAudioError(
  settings: Settings,
  media?: Pick<Media, "streams">,
): string {
  return operationOf(settings) === "compress" &&
    settings.mode === "size" &&
    settings.audio === "copy" &&
    media &&
    selectedTracks(media, "audio", settings).length > 0
    ? "目标体积需要计算音频预算。请将音频改为 AAC 或 Opus，或使用质量优先模式保留原始音频。"
    : "";
}

export function makeProject(
  media: Media,
  settings: Settings,
  outputPath: string,
  id: string,
): Project {
  const leaf = outputPath.split(/[\\/]/).pop() || outputPath;
  return {
    schemaVersion: 1,
    id,
    inputs: [media.source],
    output: { uri: outputPath, displayName: leaf, kind: media.source.kind },
    settings,
  };
}

export function parseRawArguments(value: string): string[] {
  return value.split(/\r?\n/).filter((argument) => argument.length > 0);
}
