// @vitest-environment happy-dom

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { Job } from "./bridge";
import { Queue } from "./components/Queue";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { setLanguage } from "./i18n";
import {
  estimatedRemainingSeconds,
  latestQueueStatus,
  orderQueueJobs,
} from "./queueSchedule";

vi.mock("./bridge", () => ({
  API: { getWorkspace: async () => ({ error: "" }) },
}));

beforeEach(() => setLanguage("zh-CN"));
afterEach(cleanup);

const job: Job = {
  id: "first",
  projectID: "p",
  state: "running",
  input: { uri: "first.mp4", displayName: "first.mp4", kind: "local-file" },
  output: { uri: "out.mp4", displayName: "out.mp4", kind: "local-file" },
  startedAt: "",
  completedAt: "",
  duration: 0,
  progress: 0.3,
  fps: 30,
  speed: 2,
  currentTime: 3,
  totalTime: 10,
  inputSize: 1000,
  outputSize: 0,
  error: "",
  logs: [],
};

describe("queue scheduling", () => {
  it("keeps server order for waiting tasks and newer results first", () => {
    const list = [
      { ...job, id: "old", state: "completed" },
      { ...job, id: "a", state: "queued" },
      { ...job, id: "b", state: "queued" },
      { ...job, id: "latest", state: "failed" },
      { ...job, id: "current", state: "running" },
    ];
    expect(orderQueueJobs(list, ["b", "a"]).map((j) => j.id)).toEqual([
      "current",
      "b",
      "a",
      "latest",
      "old",
    ]);
    expect(list[0].id).toBe("old");
  });
  it("does not allow an older reply to overwrite a new scheduler event", () => {
    const newer = { paused: true, pendingIDs: ["b", "a"], revision: 4 };
    expect(
      latestQueueStatus(newer, {
        paused: false,
        pendingIDs: ["a", "b"],
        revision: 3,
      }),
    ).toBe(newer);
    expect(latestQueueStatus(null, newer)).toBe(newer);
  });
  it("shows remaining time only for measurable running work", () => {
    expect(estimatedRemainingSeconds(job)).toBe(4);
    for (const changes of [
      { speed: 0 },
      { speed: Infinity },
      { currentTime: NaN },
      { currentTime: -1 },
      { currentTime: 11 },
      { totalTime: 0 },
      { state: "queued" },
      { state: "completed" },
    ])
      expect(estimatedRemainingSeconds({ ...job, ...changes })).toBeNull();
  });
  it("explains pause behavior and exposes named keyboard reorder controls", () => {
    const html = renderToStaticMarkup(
      <Queue
        jobs={[{ ...job, state: "queued" }]}
        cancel={() => {}}
        refresh={() => {}}
        compress={() => {}}
        reuse={async () => {}}
        queueStatus={{ paused: true, pendingIDs: [job.id], revision: 1 }}
        onPauseQueue={async () => {}}
        onMoveQueued={async () => {}}
        notify={() => {}}
      />,
    );
    expect(html).toContain("继续队列");
    expect(html).toContain("当前正在处理的任务会继续完成");
    expect(html).toContain('aria-label="将 first.mp4 上移一位" disabled=""');
    expect(html).toContain('aria-label="将 first.mp4 下移一位" disabled=""');
    expect(html).toContain("第 1 项");
  });
  it("offers comparison only for completed samples and avoids full-source savings claims", () => {
    const sample = {
      ...job,
      projectID: "sample-test",
      state: "completed",
      outputSize: 10,
    };
    const html = renderToStaticMarkup(
      <Queue
        jobs={[sample]}
        cancel={() => {}}
        refresh={() => {}}
        compress={() => {}}
        reuse={async () => {}}
        compareSample={() => {}}
      />,
    );
    expect(html).toContain("样片试压");
    expect(html).toContain("比较样片");
    expect(html).not.toContain("节省 99.0%");
    const queued = renderToStaticMarkup(
      <Queue
        jobs={[{ ...sample, state: "queued" }]}
        cancel={() => {}}
        refresh={() => {}}
        compress={() => {}}
        reuse={async () => {}}
        compareSample={() => {}}
      />,
    );
    expect(queued).not.toContain("比较样片");
  });
  it("renders a bounded history page and exposes the remaining count", () => {
    const history = Array.from({ length: 25 }, (_, index) => ({
      ...job,
      id: `history-${index}`,
      state: "completed",
      queuedAt: new Date(Date.UTC(2026, 9, 10, 0, 0, index)).toISOString(),
    }));
    const html = renderToStaticMarkup(
      <Queue
        jobs={history}
        cancel={() => {}}
        refresh={() => {}}
        compress={() => {}}
        reuse={async () => {}}
      />,
    );
    expect(html.match(/<article/g)).toHaveLength(20);
    expect(html).toContain("加载更多任务");
    expect(html).toContain("5");
  });
  it("cancels an active job through its visible action", async () => {
    const cancel = vi.fn();
    render(
      <Queue
        jobs={[job]}
        cancel={cancel}
        refresh={() => {}}
        compress={() => {}}
        reuse={async () => {}}
      />,
    );
    fireEvent.click(
      screen.getByRole("button", {
        name: "取消任务",
      }),
    );
    await waitFor(() => expect(cancel).toHaveBeenCalledWith(job.id));
  });
});
