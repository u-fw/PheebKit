import type { Settings } from "./bridge";

// Durations are not wall-clock times: allow hours above 23 and millisecond input.
export function parseTimecode(text: string): number {
  const value = text.trim();
  if (!/^\d+(?::\d{1,2}){0,2}(?:\.\d{1,3})?$/.test(value)) return -1;
  const parts = value.split(":").map(Number);
  if (parts.length > 1 && parts.slice(1).some((part) => part >= 60)) return -1;
  const seconds = parts.reduce((sum, part) => sum * 60 + part, 0);
  return Number.isFinite(seconds) ? seconds : -1;
}

export function formatTimecode(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "";
  const milliseconds = Math.floor(seconds * 1000 + 1e-7);
  const hours = Math.floor(milliseconds / 3_600_000);
  const minutes = Math.floor((milliseconds % 3_600_000) / 60_000);
  const rest = Math.floor((milliseconds % 60_000) / 1000);
  return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(rest).padStart(2, "0")}.${String(milliseconds % 1000).padStart(3, "0")}`;
}

export function trimError(
  settings: Settings,
  total: number,
  frameRate = 0,
): string {
  const trim = settings.trim;
  if (!trim) return "";
  if (!Number.isFinite(total) || total <= 0)
    return "视频时长未知，无法选择剪辑范围。";
  if (
    !Number.isFinite(trim.startSeconds) ||
    !Number.isFinite(trim.endSeconds) ||
    trim.startSeconds < 0 ||
    trim.endSeconds < 0
  )
    return "请输入有效时间，如 00:01:30.500，或直接输入秒数。";
  if (trim.endSeconds <= trim.startSeconds) return "结束时间必须晚于开始时间。";
  if (trim.endSeconds > total) return "剪辑范围不能超过源视频总时长。";
  if (
    Number.isFinite(frameRate) &&
    frameRate > 0 &&
    trim.endSeconds - trim.startSeconds < 1 / frameRate
  )
    return `剪辑片段至少需要一帧视频（约 ${(1 / frameRate).toFixed(3)} 秒）。`;
  return "";
}

export function selectedDuration(settings: Settings, total: number): number {
  return settings.trim
    ? trimError(settings, total)
      ? 0
      : settings.trim.endSeconds - settings.trim.startSeconds
    : total;
}
