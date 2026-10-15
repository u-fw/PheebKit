import type { Media, Project, Settings, TimeRange } from "./bridge";
import { makeProject } from "./project";

export function moveSegment(
  ranges: readonly TimeRange[],
  index: number,
  direction: number,
): TimeRange[] {
  const result = ranges.map((range) => ({ ...range }));
  const target = index + direction;
  if (
    index < 0 ||
    target < 0 ||
    index >= result.length ||
    target >= result.length
  )
    return result;
  [result[index], result[target]] = [result[target], result[index]];
  return result;
}

export function segmentProjects(
  media: Media,
  settings: Settings,
  output: string,
): Project[] {
  const ranges = settings.segments ?? [];
  if (!ranges.length || ranges.length > 16 || !output)
    throw new Error("请设置 1–16 个片段及输出文件。");
  const total = ranges.reduce(
    (sum, range) => sum + range.endSeconds - range.startSeconds,
    0,
  );
  if (!Number.isFinite(total) || total <= 0) throw new Error("片段范围无效。");
  return ranges.map((range, index) => {
    const current: Settings = {
      ...structuredClone(settings),
      segments: null,
      trim: { ...range },
      targetSizeMB:
        settings.mode === "size"
          ? (settings.targetSizeMB * (range.endSeconds - range.startSeconds)) /
            total
          : settings.targetSizeMB,
    };
    const path = output.replace(
      /(\.[^./\\]+)$/,
      `-part-${String(index + 1).padStart(2, "0")}$1`,
    );
    if (path === output) throw new Error("输出文件需要有效扩展名。");
    return makeProject(media, current, path, `segment-${crypto.randomUUID()}`);
  });
}
