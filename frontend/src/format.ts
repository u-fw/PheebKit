import { t } from "./i18n";
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return t("未知");
  if (bytes === 0) return "0 B";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  const exponent = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  return `${(bytes / 1024 ** exponent).toLocaleString("en-US", { maximumFractionDigits: exponent ? 2 : 0 })} ${units[exponent]}`;
}

export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return t("未知");
  const total = Math.floor(seconds);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const remainder = total % 60;
  return [hours, minutes, remainder]
    .map((value) => String(value).padStart(2, "0"))
    .join(":");
}

export function formatBitrate(bitsPerSecond: number): string {
  if (!Number.isFinite(bitsPerSecond) || bitsPerSecond <= 0) return t("未知");
  return bitsPerSecond >= 1_000_000
    ? `${(bitsPerSecond / 1_000_000).toFixed(2)} Mbps`
    : `${Math.round(bitsPerSecond / 1000)} kbps`;
}

export function errorMessage(error: unknown): string {
  const message = error instanceof Error ? error.message : String(error ?? "");
  return message.trim() || "原生服务未返回可用响应";
}

export function suggestedOutput(
  uri: string,
  codec: string,
  audio: string,
): string {
  if (uri.startsWith("content://"))
    return `compressed-${codec}.${audio === "opus" ? "mkv" : "mp4"}`;
  const extension = audio === "opus" ? "mkv" : "mp4";
  return `${uri.replace(/\.[^./\\]+$/, "")}-compressed.${extension}`;
}

export function savedPercent(input: number, output: number): string {
  if (input <= 0 || output < 0) return t("未知");
  const saved = (1 - output / input) * 100;
  return t("{{0}} {{1}}%", [
    t(saved >= 0 ? "节省" : "增加"),
    Math.abs(saved).toFixed(1),
  ]);
}
