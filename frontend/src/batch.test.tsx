import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { makeBatchProjects } from "./batch";
import { defaultSettings } from "./project";
import { BatchPanel } from "./components/BatchPanel";
import { ProjectActions } from "./components/ProjectActions";
import type { Media } from "./bridge";
vi.mock("./bridge", () => ({ API: {} }));
const media = (name: string, directory: string): Media => ({
  source: {
    uri: `C:/${directory}/${name}.mp4`,
    displayName: `${name}.mp4`,
    kind: "local-file",
  },
  name: `${name}.mp4`,
  container: "mp4",
  sizeBytes: 1000,
  durationSeconds: 10,
  bitrate: 800,
  streams: [
    {
      index: 2,
      codecType: "audio",
      codecName: "aac",
      width: 0,
      height: 0,
      frameRate: 0,
      pixelFormat: "",
      bitDepth: 0,
      colorSpace: "",
      colorTransfer: "",
      colorPrimaries: "",
      hdr: "",
      language: "zho",
      title: "",
      attachedPicture: false,
      timedThumbnails: false,
    },
  ],
});
describe("batch and project workflows", () => {
  it("allocates unique names even when a numbered duplicate matches a natural source name", () => {
    const projects = makeBatchProjects(
      [media("a", "one"), media("a", "two"), media("a-2", "three")],
      defaultSettings,
      "C:/out",
      "2",
    );
    expect(new Set(projects.map((p) => p.output.uri.toLowerCase())).size).toBe(
      3,
    );
  });
  it("keeps media-specific trim and crop off unrelated files and uses each audio index", () => {
    const projects = makeBatchProjects(
      [media("a", "one"), media("b", "two")],
      {
        ...defaultSettings,
        operation: "extract-audio",
        trim: { startSeconds: 2, endSeconds: 4 },
        audioSelection: { mode: "selected", indices: [99] },
        transform: {
          rotation: 0,
          flipHorizontal: false,
          flipVertical: false,
          crop: { left: 2, right: 2, top: 0, bottom: 0 },
        },
      },
      "C:/out",
      "audio",
    );
    expect(projects[1].settings.trim).toBeNull();
    expect(projects[1].settings.transform?.crop).toBeNull();
    expect(projects[1].settings.audioSelection?.indices).toEqual([2]);
    expect(projects[1].settings.subtitleSelection?.mode).toBe("none");
  });
  it("rejects path fragments and makes batch full-duration and sample audio behavior explicit", () => {
    expect(() =>
      makeBatchProjects(
        [media("a", "one")],
        defaultSettings,
        "C:/out",
        "../bad",
      ),
    ).toThrow();
    const html = renderToStaticMarkup(
      <BatchPanel
        files={[media("a", "one"), media("b", "two")]}
        settings={defaultSettings}
        connected
        onJobs={() => {}}
        showQueue={() => {}}
      />,
    );
    expect(html).toContain("完整时长");
    expect(html).toContain("字幕");
    expect(html).toContain("将 2 个文件加入队列");
    const project = makeBatchProjects(
      [media("a", "one")],
      defaultSettings,
      "C:/out",
      "done",
    )[0];
    const actions = renderToStaticMarkup(
      <ProjectActions
        project={project}
        connected
        load={async () => {}}
        onJob={() => {}}
        showQueue={() => {}}
      />,
    );
    expect(actions).toContain("Copy 音频会在样片中改为 AAC");
    expect(actions).toContain("保存项目");
  });
});
