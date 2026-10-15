import { describe, it, expect, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { Media, Settings } from "./bridge";
import { MediaRecovery } from "./components/MediaRecovery";
import {
  applyPresetSettings,
  changeOperation,
  defaultSettings,
  validateSettings,
} from "./project";
import { selectedDuration } from "./trim";
import { makeBatchProjects } from "./batch";
vi.mock("./bridge", () => ({ API: {} }));
const media: Media = {
  source: { uri: "C:/in.mp4", kind: "local-file", displayName: "in.mp4" },
  name: "in.mp4",
  container: "mp4",
  durationSeconds: 16,
  sizeBytes: 1000,
  bitrate: 500,
  sourceFingerprint: "fp",
  streams: [],
};
function settings(): Settings {
  return {
    ...defaultSettings,
    audio: "aac",
    recovery: {
      mode: "delete",
      sourceFingerprint: "fp",
      damagedRanges: [{ startSeconds: 5, endSeconds: 10 }],
    },
  };
}
describe("conservative media recovery", () => {
  it("budgets retained duration and preserves timeline for black fill", () => {
    const s = settings();
    expect(selectedDuration(s, 16)).toBe(11);
    s.recovery!.mode = "black";
    expect(selectedDuration(s, 16)).toBe(16);
  });
  it("preserves recovery across same-operation presets, omits it from other sources and operations", () => {
    expect(
      applyPresetSettings(settings(), defaultSettings, media).recovery,
    ).toEqual(settings().recovery);
    expect(changeOperation(settings(), "remux", media).recovery).toBeNull();
    expect(
      makeBatchProjects([media], settings(), "C:/out", "-recovered")[0].settings
        .recovery,
    ).toBeNull();
  });
  it("rejects copy audio and incompatible edits", () => {
    const errors = validateSettings(
      {
        ...settings(),
        audio: "copy",
        trim: { startSeconds: 0, endSeconds: 1 },
      },
      "C:/out.mp4",
      "C:/in.mp4",
      media,
    ).join(" ");
    expect(errors).toContain("恢复需要重新编码");
    expect(errors).toContain("恢复不能同时剪辑");
  });
  it("exposes conservative precision and source-preserving intent, with offline scan disabled", () => {
    const html = renderToStaticMarkup(
      <MediaRecovery
        media={media}
        settings={settings()}
        connected={false}
        onChange={() => {}}
      />,
    );
    expect(html).toContain("5 秒窗口");
    expect(html).toContain("可能包含正常画面");
    expect(html).toContain("保存前会完整解码复检");
    expect(html).toMatch(/disabled="">扫描当前视频/);
  });
});
