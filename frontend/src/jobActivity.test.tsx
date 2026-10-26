import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { Job } from "./bridge";
import { JobActivityIndicator } from "./components/JobActivityIndicator";
import { summarizeJobActivity } from "./jobActivity";

function job(id: string, state: string, progress = 0): Job {
  return {
    id,
    projectID: "project",
    state,
    input: { uri: `/media/${id}.mp4`, displayName: `${id}.mp4`, kind: "file" },
    output: {
      uri: `/media/${id}-out.mp4`,
      displayName: `${id}-out.mp4`,
      kind: "file",
    },
    startedAt: "",
    completedAt: "",
    duration: 0,
    progress,
    fps: 0,
    speed: 0,
    currentTime: 0,
    totalTime: 0,
    inputSize: 0,
    outputSize: 0,
    error: "",
    logs: [],
  };
}

describe("job activity", () => {
  it("summarizes the current queue work without past history", () => {
    const summary = summarizeJobActivity([
      job("old", "completed", 1),
      job("queued", "queued"),
      job("running", "running", 0.42),
    ]);

    expect(summary.activeCount).toBe(2);
    expect(summary.current?.id).toBe("running");
    expect(summary.progress).toBeCloseTo(0.42);
    expect(summary.paused).toBe(false);
  });

  it("reports a paused queue when no task has started", () => {
    const summary = summarizeJobActivity([job("queued", "queued")], true);
    expect(summary.paused).toBe(true);
    expect(summary.progress).toBe(0);
  });

  it("renders a keyboard action and bounded progress", () => {
    const summary = summarizeJobActivity([job("running", "running", 1.4)]);
    const html = renderToStaticMarkup(
      <JobActivityIndicator summary={summary} onOpen={() => {}} />,
    );

    expect(html).toContain('<button class="job-activity-chip"');
    expect(html).toContain(
      'aria-label="打开任务队列：正在处理，1 个任务，100%"',
    );
    expect(html).toContain("100%");
    expect(html).toContain("scaleX(1)");
    expect(summary.progress).toBe(1);
  });
});
