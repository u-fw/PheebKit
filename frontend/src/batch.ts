import type { Media, Project, Settings, TrackSelection } from "./bridge";
import { makeProject, operationOf, outputExtension } from "./project";

export interface BatchTrackOverride {
  audio: string;
  subtitle: string;
}
export function directoryOf(path: string): string {
  return path.replace(/[\\/][^\\/]*$/, "");
}
function selection(value: string): TrackSelection {
  return value === "all" || value === "none"
    ? { mode: value }
    : { mode: "selected", indices: [Number(value)] };
}
export function defaultBatchTracks(
  media: Media,
  settings: Settings,
): BatchTrackOverride {
  const operation = operationOf(settings);
  return {
    audio:
      operation === "extract-subtitle"
        ? "none"
        : operation === "extract-audio"
          ? String(
              media.streams.find((s) => s.codecType === "audio")?.index ??
                "none",
            )
          : settings.audioSelection?.mode === "none" ||
              settings.audio === "none"
            ? "none"
            : "all",
    subtitle:
      operation === "extract-subtitle"
        ? String(
            media.streams.find((s) => s.codecType === "subtitle")?.index ??
              "none",
          )
        : settings.subtitleSelection?.mode === "all"
          ? "all"
          : "none",
  };
}

// Each source receives its own indexes. Media-specific trim and crop are never
// transferred from the currently edited source onto unrelated files.
export function makeBatchProjects(
  files: Media[],
  settings: Settings,
  directory: string,
  suffix: string,
  overrides: Record<string, BatchTrackOverride> = {},
): Project[] {
  if (!directory.trim()) throw new Error("请选择批量输出文件夹。");
  if (
    !suffix ||
    suffix.length > 64 ||
    /[<>:"/\\|?*]/.test(suffix) ||
    [...suffix].some((character) => character.charCodeAt(0) < 32) ||
    /[. ]$/.test(suffix)
  )
    throw new Error("文件名后缀不能为空，且不能包含路径符号或以空格、点结尾。");
  const usedNames = new Set<string>();
  const ext = outputExtension(settings);
  return files.map((media) => {
    const tracks =
      overrides[media.source.uri] ?? defaultBatchTracks(media, settings);
    const current: Settings = {
      ...settings,
      trim: null,
      segments: null,
      recovery: null,
      subtitleBurn: null,
      rawArguments: [...settings.rawArguments],
      audioSelection: selection(tracks.audio),
      subtitleSelection: selection(tracks.subtitle),
      transform: settings.transform
        ? { ...settings.transform, crop: null }
        : null,
    };
    const leaf = media.source.uri.split(/[\\/]/).pop() || "media";
    const stem = leaf.replace(/\.[^.]+$/, "");
    const name = `${stem}-${suffix}`;
    let outputName = `${name}.${ext}`;
    let duplicate = 2;
    while (usedNames.has(outputName.toLowerCase())) {
      outputName = `${name}-${duplicate++}.${ext}`;
    }
    usedNames.add(outputName.toLowerCase());
    const output = `${directory.replace(/[\\/]+$/, "")}/${outputName}`;
    return makeProject(media, current, output, `batch-${crypto.randomUUID()}`);
  });
}
