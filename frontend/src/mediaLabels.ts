import { t } from "./i18n";
import type { Stream } from "./bridge";

export function streamLabel(stream: Stream): string {
  const languages: Record<string, string> = {
    und: t("语言未标注"),
    zho: t("中文"),
    chi: t("中文"),
    eng: t("英语"),
    jpn: t("日语"),
    kor: t("韩语"),
  };
  const language = stream.language
    ? languages[stream.language] || stream.language
    : t("语言未标注");
  return [
    stream.codecName.toUpperCase(),
    stream.codecType === "audio" && stream.sampleRate
      ? `${stream.sampleRate / 1000} kHz`
      : "",
    stream.codecType === "audio" && stream.channels
      ? t("{{0}} 声道", [stream.channels])
      : "",
    language,
    stream.title,
  ]
    .filter(Boolean)
    .join(" · ");
}

export function videoLabel(stream: Stream): string {
  const parts = [];
  if (stream.pixelFormat)
    parts.push(
      stream.pixelFormat.startsWith("yuv420p")
        ? t("YUV 4:2:0{{0}}", [
            stream.bitDepth ? t(" · {{0}} 位", [stream.bitDepth]) : "",
          ])
        : t("像素格式 {{0}}", [stream.pixelFormat]),
    );
  const color = [
    stream.colorSpace,
    stream.colorTransfer,
    stream.colorPrimaries,
  ].filter(Boolean);
  if (color.length && color.every((value) => value === "bt709"))
    parts.push(t("BT.709 标准色彩"));
  else if (color.length)
    parts.push(
      [...new Set(color)].map((value) => value.toUpperCase()).join(" / "),
    );
  return parts.join(" · ") || t("色彩信息未标注");
}
