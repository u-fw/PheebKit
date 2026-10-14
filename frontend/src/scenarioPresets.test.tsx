import { describe, it, expect, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { buildScenario } from "./scenarioPresets";
import { validateSettings } from "./project";
import { ScenarioPresets } from "./components/ScenarioPresets";
import { EditorPreferences } from "./components/EditorPreferences";
import { ProjectLibrary } from "./components/ProjectLibrary";
import type { Media } from "./bridge";
vi.mock("./bridge", () => ({ API: {} }));
const media = {
  source: { uri: "C:/in.mp4", kind: "local-file", displayName: "in.mp4" },
  streams: [
    {
      index: 0,
      codecType: "video",
      codecName: "h264",
      width: 640,
      height: 360,
      frameRate: 30,
    },
  ],
  durationSeconds: 60,
} as Media;
describe("editable scenario and recovery controls", () => {
  it("does not upscale small media and keeps scene parameters editable", () => {
    for (const id of ["compatible", "share", "quality", "size"]) {
      const settings = buildScenario(id, media);
      expect(settings.resolution).toBe(0);
      settings.quality = 17;
      settings.audioBitrateKbps = 80;
      expect(settings.audioBitrateKbps).toBe(80);
    }
    const view = renderToStaticMarkup(
      <ScenarioPresets media={media} apply={() => {}} />,
    );
    expect(view).toContain("所有参数均可继续修改");
    expect(view).toContain("应用场景");
    expect(
      validateSettings(
        { ...buildScenario("compatible", media), audioBitrateKbps: 999 },
        "C:/out.mp4",
        media.source.uri,
        media,
      ).join(),
    ).toContain("音频码率");
  });
  it("exposes bounded recovery preferences and disables active-session recovery", () => {
    const preferences = {
      autosave: true,
      intervalSeconds: 15,
      draftLimit: 5,
      retentionDays: 14,
      recentLimit: 10,
      undoLimit: 50,
    };
    const view = renderToStaticMarkup(
      <EditorPreferences value={preferences} save={async () => {}} />,
    );
    expect(view).toContain("草稿保存间隔");
    expect(view).toContain("撤销步数");
    const library = renderToStaticMarkup(
      <ProjectLibrary
        library={{
          preferences,
          error: "",
          recent: [],
          drafts: [
            {
              id: "active",
              source: "C:/in.mp4",
              updatedAt: new Date(0).toISOString(),
              active: true,
            },
          ],
        }}
        connected
        refresh={async () => {}}
        openRecent={() => {}}
        restore={() => {}}
      />,
    );
    expect(library).toContain("其他窗口正在编辑");
    expect(library).toMatch(/disabled="">恢复草稿/);
  });
});
