import type { Stream } from "./bridge";

export function streamLabel(stream: Stream): string {
  const languages: Record<string, string> = {
    und: "语言未标注",
    zho: "中文",
    chi: "中文",
    eng: "英语",
    jpn: "日语",
    kor: "韩语",
  };
  const language = stream.language
    ? languages[stream.language] || stream.language
    : "语言未标注";
  return [stream.codecName.toUpperCase(), language, stream.title]
    .filter(Boolean)
    .join(" · ");
}

export function videoLabel(stream: Stream): string {
  const parts = [];
  if (stream.pixelFormat)
    parts.push(
      stream.pixelFormat.startsWith("yuv420p")
        ? `YUV 4:2:0${stream.bitDepth ? ` · ${stream.bitDepth} 位` : ""}`
        : `像素格式 ${stream.pixelFormat}`,
    );
  const color = [
    stream.colorSpace,
    stream.colorTransfer,
    stream.colorPrimaries,
  ].filter(Boolean);
  if (color.length && color.every((value) => value === "bt709"))
    parts.push("BT.709 标准色彩");
  else if (color.length)
    parts.push(
      [...new Set(color)].map((value) => value.toUpperCase()).join(" / "),
    );
  return parts.join(" · ") || "色彩信息未标注";
}
