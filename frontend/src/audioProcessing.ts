import type { AudioProcessing, Settings } from "./bridge";
import { t } from "./i18n";

export const audioSampleRates = [
  8000, 11025, 12000, 16000, 22050, 24000, 32000, 44100, 48000, 64000, 88200,
  96000,
];
export function defaultAudioProcessing(): AudioProcessing {
  return {
    normalize: null,
    denoiseReductionDB: 0,
    noiseFloorDB: -50,
    equalizer: [],
    fadeInSeconds: 0,
    fadeOutSeconds: 0,
    sampleRate: 0,
    channels: "source",
  };
}
export function audioProcessingErrors(
  s: Settings,
  duration?: number,
  trackCount?: number,
): string[] {
  const p = s.audioProcessing;
  if (!p) return [];
  const errors: string[] = [];
  const between = (v: number, min: number, max: number) =>
    Number.isFinite(v) && v >= min && v <= max;
  if (
    s.operation &&
    s.operation !== "compress" &&
    s.operation !== "extract-audio"
  )
    errors.push("声音处理仅适用于压缩或音频提取。");
  if (s.audio === "copy" || s.audio === "none")
    errors.push("声音处理需要重新编码且保留音轨。");
  if (s.recovery) errors.push("请先恢复视频，再应用声音处理。");
  if (trackCount !== undefined && (trackCount < 1 || trackCount > 4))
    errors.push("声音处理需要选择 1–4 条音轨。");
  if (
    !between(p.denoiseReductionDB, 0, 97) ||
    (p.denoiseReductionDB > 0 && p.denoiseReductionDB < 0.01)
  )
    errors.push("降噪强度需要为 0（关闭）或 0.01–97 dB。");
  if (p.denoiseReductionDB > 0 && !between(p.noiseFloorDB, -80, -20))
    errors.push("噪声底需要为 -80 至 -20 dB。");
  if (
    p.normalize &&
    (!between(p.normalize.integratedLUFS, -70, -5) ||
      !between(p.normalize.truePeakDBTP, -9, 0) ||
      !between(p.normalize.rangeLU, 1, 50))
  )
    errors.push("响度目标、真峰值或响度范围超出允许范围。");
  if (p.sampleRate !== 0 && !audioSampleRates.includes(p.sampleRate))
    errors.push("请选择支持的输出采样率。");
  if (s.audio === "opus" && p.sampleRate !== 0 && p.sampleRate !== 48000)
    errors.push("当前 Opus 解码采样率固定为 48000 Hz，请选择自动或 48000 Hz。");
  if (s.audio === "mp3" && p.sampleRate > 48000)
    errors.push("MP3 输出采样率不能高于 48000 Hz。");
  if (!["", "source", "mono", "stereo"].includes(p.channels))
    errors.push("请选择保留、单声道或立体声。");
  const bands = p.equalizer ?? [];
  if (bands.length > 8) errors.push("均衡器最多 8 个频段。");
  if (
    bands.some(
      (b) =>
        !between(b.frequencyHz, 20, 20000) ||
        !between(b.q, 0.1, 20) ||
        !between(b.gainDB, -24, 24),
    )
  )
    errors.push("均衡器需要 20–20000 Hz、Q 0.1–20、增益 ±24 dB。");
  if (
    bands.some(
      (b) => b.gainDB !== 0 && b.frequencyHz >= (p.sampleRate || 48000) / 2,
    )
  )
    errors.push("有效均衡频率需要低于输出采样率的一半。");
  if (!between(p.fadeInSeconds, 0, 600) || !between(p.fadeOutSeconds, 0, 600))
    errors.push("淡入淡出需要为 0–600 秒。");
  if (
    duration !== undefined &&
    p.fadeInSeconds + p.fadeOutSeconds > 0 &&
    (!Number.isFinite(duration) ||
      duration <= 0 ||
      p.fadeInSeconds + p.fadeOutSeconds > duration + 1e-9)
  )
    errors.push("淡入与淡出的总时长不能超过已知输出时长。");
  return errors;
}
export function audioProcessingSummary(s: Settings): string {
  const p = s.audioProcessing;
  if (!p) return "";
  return [
    p.normalize ? t("动态响度归一化") : "",
    p.denoiseReductionDB > 0 ? t("降噪 {{0}} dB", [p.denoiseReductionDB]) : "",
    (p.equalizer ?? []).some((b) => b.gainDB !== 0) ? t("均衡器") : "",
    p.fadeInSeconds || p.fadeOutSeconds
      ? t("淡入/出 {{0}}/{{1}} 秒", [p.fadeInSeconds, p.fadeOutSeconds])
      : "",
    `${(p.sampleRate || 48000) / 1000} kHz`,
    p.channels === "mono"
      ? t("单声道")
      : p.channels === "stereo"
        ? t("立体声")
        : t("保留声道"),
  ]
    .filter(Boolean)
    .join(" · ");
}
