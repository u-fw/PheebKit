import { describe, it, expect, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { JobHistory } from "./jobHistory";
import { jobErrorHelp } from "./jobError";
import { Queue } from "./components/Queue";
import { ClipPreview } from "./components/ClipPreview";
import { defaultSettings, makeProject } from "./project";
import type { Job, Media } from "./bridge";
vi.mock("./bridge", () => ({ API: {} }));
const media: Media = {
  source: { uri: "in.mp4", displayName: "in.mp4", kind: "local-file" },
  name: "in.mp4",
  container: "mp4",
  durationSeconds: 10,
  sizeBytes: 1000,
  bitrate: 1000,
  streams: [],
};
const job: Job = {
  id: "job",
  projectID: "p",
  state: "interrupted",
  input: media.source,
  output: { uri: "out.mp4", displayName: "out.mp4", kind: "local-file" },
  queuedAt: "2026-10-10T00:00:00Z",
  startedAt: "",
  completedAt: "2026-10-10T00:00:10Z",
  duration: 0,
  progress: 0.5,
  fps: 0,
  speed: 0,
  currentTime: 5,
  totalTime: 10,
  inputSize: 1000,
  outputSize: 500,
  error: "previous session interrupted",
  logs: [],
  project: makeProject(media, defaultSettings, "out.mp4", "p"),
};
describe("task workflows", () => {
  it("does not resurrect interrupted jobs from stale snapshots and prunes them", () => {
    const history = new JobHistory();
    history.update([job]);
    history.update([{ ...job, state: "running" }]);
    expect(history.list()[0].state).toBe("interrupted");
    history.update(
      Array.from({ length: 205 }, (_, i) => ({
        ...job,
        id: `new-${i}`,
        queuedAt: `2026-10-10T00:01:${String(i % 60).padStart(2, "0")}Z`,
      })),
    );
    expect(history.list()).toHaveLength(200);
    expect(history.list().find((item) => item.id === "job")).toBeUndefined();
  });
  it("offers result actions only for finished outputs and preserves diagnostic detail", () => {
    const interrupted = renderToStaticMarkup(
      <Queue
        jobs={[job]}
        cancel={() => {}}
        refresh={() => {}}
        compress={() => {}}
        reuse={async () => {}}
      />,
    );
    expect(interrupted).toContain("沿用设置再处理");
    expect(interrupted).not.toContain("打开文件");
    expect(interrupted).toContain("previous session interrupted");
    const complete = renderToStaticMarkup(
      <Queue
        jobs={[{ ...job, state: "completed", error: "" }]}
        cancel={() => {}}
        refresh={() => {}}
        compress={() => {}}
        reuse={async () => {}}
      />,
    );
    expect(complete).toContain("打开所在文件夹");
  });
  it("explains an output collision without hiding the original reason", () => {
    expect(jobErrorHelp("output already exists")).toContain("新的文件名");
    expect(jobErrorHelp("Permission denied")).toContain("权限");
  });
  it("makes preview location and boundary actions accessible", () => {
    const html = renderToStaticMarkup(
      <ClipPreview
        media={media}
        start={1}
        end={8}
        visible
        onBoundary={() => {}}
      />,
    );
    expect(html).toContain("预览位置");
    expect(html).toContain("当前位置设为起点");
    expect(html).toContain("播放所选片段");
  });
});
