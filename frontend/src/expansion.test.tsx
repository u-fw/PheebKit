import { describe, it, expect, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { Media, Settings } from "./bridge";
import {
  defaultSettings,
  applyPresetSettings,
  changeOperation,
  validateSettings,
} from "./project";
import { selectedDuration, trimError } from "./trim";
import { moveSegment, segmentProjects } from "./segments";
import { defaultSubtitle, subtitleError } from "./subtitles";
import { makeBatchProjects } from "./batch";
import { SubtitleBurnSettings } from "./components/SubtitleBurnSettings";
import { FFmpegExplorer, catalogCategories } from "./components/FFmpegExplorer";
vi.mock("./bridge", () => ({ API: {} }));
const media: Media = {
  source: { uri: "C:/media/in.mkv", kind: "local-file", displayName: "in.mkv" },
  name: "in.mkv",
  container: "mkv",
  durationSeconds: 10,
  sizeBytes: 1000,
  bitrate: 800,
  streams: [],
};
const segments = [
  { startSeconds: 5, endSeconds: 8 },
  { startSeconds: 1, endSeconds: 2 },
];
function settings(): Settings {
  return {
    ...defaultSettings,
    audio: "aac",
    segments: structuredClone(segments),
    subtitleBurn: {
      ...defaultSubtitle(),
      source: "external",
      path: "C:/字幕.srt",
    },
  };
}
describe("editable FFmpeg expansion", () => {
  it("preserves explicit order/repeats and rejects invalid selected ranges", () => {
    expect(selectedDuration(settings(), 10)).toBe(4);
    expect(moveSegment(segments, 1, -1)).toEqual([segments[1], segments[0]]);
    expect(segments[0].startSeconds).toBe(5);
    expect(trimError({ ...settings(), trim: segments[0] }, 10)).toContain(
      "不能同时",
    );
    expect(
      trimError(
        { ...settings(), segments: [{ startSeconds: 9, endSeconds: 11 }] },
        10,
      ),
    ).toContain("片段 1");
    expect(
      trimError(
        { ...settings(), segments: [{ startSeconds: 1, endSeconds: 1.01 }] },
        10,
        10,
      ),
    ).toContain("一帧");
  });
  it("creates independent outputs and proportionally shares target-size budget", () => {
    const original = { ...settings(), mode: "size", targetSizeMB: 100 };
    const projects = segmentProjects(media, original, "C:/result.mp4");
    expect(projects.map((p) => p.output.uri)).toEqual([
      "C:/result-part-01.mp4",
      "C:/result-part-02.mp4",
    ]);
    expect(projects.map((p) => p.settings.targetSizeMB)).toEqual([75, 25]);
    expect(projects[0].settings.trim).toEqual(segments[0]);
    expect(projects[0].settings.segments).toBeNull();
    projects[0].settings.subtitleBurn!.font = "Arial";
    expect(original.subtitleBurn!.font).toBe("Microsoft YaHei");
  });
  it("retains media-specific edits across presets but clears them on operation/batch changes", () => {
    const s = settings();
    const applied = applyPresetSettings(
      s,
      { ...defaultSettings, audio: "aac" },
      media,
    );
    expect(applied.segments).toEqual(segments);
    expect(applied.subtitleBurn).toEqual(s.subtitleBurn);
    expect(changeOperation(s, "remux", media).segments).toBeNull();
    expect(changeOperation(s, "remux", media).subtitleBurn).toBeNull();
    const batch = makeBatchProjects([media], s, "C:/out", "batch");
    expect(batch[0].settings.segments).toBeNull();
    expect(batch[0].settings.subtitleBurn).toBeNull();
  });
  it.each(["Arial,Fontsize=999", "Arial';movie=file", "https://font"])(
    "rejects filter/style injection through font %s",
    (font) => {
      expect(
        subtitleError(
          {
            ...settings(),
            subtitleBurn: { ...settings().subtitleBurn!, font },
          },
          media,
        ),
      ).toContain("字体名");
    },
  );
  it("validates colors, offsets and unavailable embedded text tracks", () => {
    expect(subtitleError(settings(), media)).toBe("");
    expect(
      subtitleError(
        {
          ...settings(),
          subtitleBurn: { ...defaultSubtitle(), offsetSeconds: Infinity },
        },
        media,
      ),
    ).not.toBe("");
    expect(
      subtitleError(
        {
          ...settings(),
          subtitleBurn: {
            ...settings().subtitleBurn!,
            color: "white:movie=foo",
          },
        },
        media,
      ),
    ).toContain("#RRGGBB");
    expect(
      subtitleError(
        { ...settings(), subtitleBurn: defaultSubtitle(22) },
        media,
      ),
    ).toContain("文字字幕轨道");
    expect(
      validateSettings(
        { ...settings(), operation: "remux" },
        "C:/out.mkv",
        media.source.uri,
        media,
      ),
    ).toContain("多片段和字幕烧录仅适用于压缩任务。");
  });
  it("labels subtitle permanence, offset, font fallback and actual-capability boundaries", () => {
    const html = renderToStaticMarkup(
      <SubtitleBurnSettings
        media={media}
        settings={settings()}
        onChange={() => undefined}
        capabilities={null}
        connected
      />,
    );
    expect(html).toContain("烧录后无法关闭字幕");
    expect(html).toContain("正偏移让字幕延后");
    expect(html).toContain("会回退");
    expect(html).toContain("字幕位置");
    const explorer = renderToStaticMarkup(<FFmpegExplorer connected />);
    expect(explorer).toContain("不代表界面已提供该功能的执行入口");
    expect(explorer).toContain("完整参数帮助");
    expect(catalogCategories).toHaveLength(13);
  });
});
