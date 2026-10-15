import type { Media, Settings, SubtitleBurn } from "./bridge";
export const textSubtitleCodecs = new Set([
  "subrip",
  "srt",
  "ass",
  "ssa",
  "webvtt",
  "mov_text",
  "text",
]);
export function defaultSubtitle(streamIndex = 0): SubtitleBurn {
  return {
    source: "embedded",
    streamIndex,
    path: "",
    offsetSeconds: 0,
    font: "Microsoft YaHei",
    fontSize: 24,
    color: "#ffffff",
    outlineColor: "#000000",
    outline: 2,
    shadow: 0,
    alignment: 2,
    margin: 20,
    bold: false,
  };
}
export function subtitleError(
  settings: Settings,
  media?: Pick<Media, "streams">,
): string {
  const b = settings.subtitleBurn;
  if (!b) return "";
  if (!["embedded", "external"].includes(b.source))
    return "请选择文字字幕来源。";
  if (
    b.source === "embedded" &&
    media &&
    !media.streams.some(
      (s) =>
        s.index === b.streamIndex &&
        s.codecType === "subtitle" &&
        textSubtitleCodecs.has(s.codecName),
    )
  )
    return "请选择当前媒体的文字字幕轨道；暂不支持图片字幕。";
  if (
    b.source === "external" &&
    (!/^(?:[a-zA-Z]:[\\/]|\\\\|\/)/.test(b.path ?? "") ||
      !/\.(srt|ass|ssa|vtt)$/i.test(b.path ?? ""))
  )
    return "请选择本地 SRT / ASS / SSA / VTT 字幕文件。";
  if (!Number.isFinite(b.offsetSeconds) || Math.abs(b.offsetSeconds) > 86400)
    return "字幕偏移必须在 ±86400 秒内。";
  if (!/^[\p{L}\p{N} ._-]{1,64}$/u.test(b.font))
    return "字体名需要 1–64 个文字、数字、空格、点、下划线或连字符。";
  for (const [value, min, max] of [
    [b.fontSize, 8, 144],
    [b.outline, 0, 10],
    [b.shadow, 0, 10],
    [b.alignment, 1, 9],
    [b.margin, 0, 500],
  ])
    if (!Number.isInteger(value) || value < min || value > max)
      return "字幕字号、边框、阴影、位置或边距超出合法范围。";
  if (
    !/^#[a-fA-F0-9]{6}$/.test(b.color) ||
    !/^#[a-fA-F0-9]{6}$/.test(b.outlineColor)
  )
    return "字幕颜色需要 #RRGGBB。";
  return "";
}
