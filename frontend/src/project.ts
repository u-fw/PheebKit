import type { Media, Project, Settings } from "./bridge";

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

export function validateSettings(
  settings: Settings,
  outputPath: string,
  inputPath: string,
  media?: Pick<Media, "streams" | "durationSeconds">,
): string[] {
  const errors: string[] = [];
  if (!outputPath.trim()) errors.push("请选择输出文件。");
  if (outputPath === inputPath && inputPath)
    errors.push("输出文件不能覆盖源文件，请选择新路径。");
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
  return settings.mode === "size" &&
    settings.audio === "copy" &&
    media?.streams.some((stream) => stream.codecType === "audio")
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
