import { describe, expect, it } from "vitest";
import type { Job } from "./bridge";
import { JobHistory } from "./jobHistory";

function job(id = "job", changes: Partial<Job> = {}): Job {
  return {
    id,
    projectID: "project",
    state: "queued",
    input: { uri: "in.mp4", displayName: "in.mp4", kind: "local-file" },
    output: { uri: "out.mp4", displayName: "out.mp4", kind: "local-file" },
    queuedAt: "2026-10-10T00:00:00.000Z",
    startedAt: "",
    completedAt: "",
    duration: 0,
    progress: 0,
    fps: 0,
    speed: 0,
    currentTime: 0,
    totalTime: 10,
    inputSize: 0,
    outputSize: 0,
    error: "",
    logs: [],
    ...changes,
  };
}

const running = job("job", {
  state: "running",
  startedAt: "2026-10-10T00:00:01Z",
  progress: 0.4,
  currentTime: 4,
  outputSize: 400,
  logs: ["encoding"],
});
const completed = job("job", {
  ...running,
  state: "completed",
  completedAt: "2026-10-10T00:00:02Z",
  duration: 1,
  progress: 1,
  currentTime: 10,
  inputSize: 2000,
  outputSize: 1000,
  savedPercent: 50,
  logs: ["encoding", "published output"],
});

describe("job snapshots and event ordering", () => {
  it("detects dropped native events and accepts contiguous events", () => {
    const history = new JobHistory();
    history.reconcile([{ ...running, eventSequence: 4 }], 0);
    expect(
      history.updateWithSequence([{ ...running, eventSequence: 5 }])
        .sequenceGap,
    ).toBe(false);
    const skipped = history.updateWithSequence([
      { ...running, eventSequence: 7 },
    ]);
    expect(skipped.sequenceGap).toBe(true);
    expect(skipped.jobs).toEqual([{ ...running, eventSequence: 7 }]);
  });

  it("never lets a lower sequence replace a newer progress snapshot", () => {
    const history = new JobHistory();
    const newest = {
      ...running,
      eventSequence: 12,
      progress: 0.6,
      currentTime: 6,
    };
    history.update([newest]);
    expect(
      history.update([
        {
          ...newest,
          eventSequence: 11,
          progress: 0.8,
          currentTime: 8,
        },
      ]),
    ).toEqual([newest]);
  });

  it("keeps an early completion event while the initial queue request is pending", () => {
    const history = new JobHistory();
    const initialRequest = history.revision;
    history.update([completed]);
    expect(history.reconcile([running], initialRequest)).toEqual([completed]);
  });

  it("does not let a refresh erase progress or a completion received during the request", () => {
    const history = new JobHistory();
    history.update([running]);
    const refreshRequest = history.revision;
    const advanced = {
      ...running,
      progress: 0.8,
      currentTime: 8,
      outputSize: 800,
    };
    history.update([advanced]);
    expect(history.reconcile([running], refreshRequest)).toEqual([advanced]);

    const secondRefresh = history.revision;
    history.update([completed]);
    expect(history.reconcile([advanced], secondRefresh)).toEqual([completed]);
  });

  it("protects new logs even when the event has the same progress percentage", () => {
    const history = new JobHistory();
    history.update([running]);
    const request = history.revision;
    const withNewLog = {
      ...running,
      logs: [...running.logs, "new encoder log"],
    };
    history.update([withNewLog]);
    expect(history.reconcile([running], request)).toEqual([withNewLog]);
  });

  it("keeps the newer refresh when concurrent replies arrive out of order", () => {
    const history = new JobHistory();
    history.update([running]);
    const olderRequest = history.revision;
    const newerRequest = history.revision;
    history.reconcile([completed], newerRequest);
    expect(history.reconcile([running], olderRequest)).toEqual([completed]);
  });

  it("does not regress when the enqueue reply or an older progress event arrives late", () => {
    const history = new JobHistory();
    history.update([running]);
    expect(history.update([job()])).toEqual([running]);
    const advanced = {
      ...running,
      progress: 0.8,
      currentTime: 8,
      outputSize: 800,
    };
    history.update([advanced]);
    expect(history.update([running])).toEqual([advanced]);
    history.update([completed]);
    expect(history.update([running])).toEqual([completed]);
  });

  it("allows successful publication after cancellation, but never returns to a running state", () => {
    const history = new JobHistory();
    history.update([running]);
    const cancelling = { ...running, state: "cancelling" };
    history.update([cancelling]);
    expect(
      history.update([{ ...running, progress: 0.9, currentTime: 9 }]),
    ).toEqual([cancelling]);
    expect(history.update([completed])).toEqual([completed]);
  });

  it("preserves failed and cancelled results against an older queued snapshot", () => {
    for (const state of ["failed", "cancelled"]) {
      const history = new JobHistory();
      const request = history.revision;
      const terminal = {
        ...completed,
        state,
        error: state === "failed" ? "disk full" : "",
      };
      history.update([terminal]);
      expect(history.reconcile([job()], request)).toEqual([terminal]);
    }
  });

  it("accepts the later committed terminal result within the same millisecond", () => {
    const history = new JobHistory();
    history.update([
      {
        ...completed,
        state: "cancelled",
        completedAt: "2026-10-10T00:00:02.1231Z",
      },
    ]);
    const published = {
      ...completed,
      completedAt: "2026-10-10T00:00:02.1232Z",
    };
    expect(history.update([published])).toEqual([published]);
  });
});

describe("bounded queue history", () => {
  const oldJobs = Array.from({ length: 200 }, (_, index) =>
    job(`old-${index}`, {
      ...completed,
      id: `old-${index}`,
      queuedAt: new Date(Date.UTC(2026, 9, 10, 0, 0, index)).toISOString(),
    }),
  );

  it("keeps exactly 200 records after repeated events while retaining old active jobs", () => {
    const history = new JobHistory();
    const active = ["queued", "preparing", "running", "cancelling"].map(
      (state, index) =>
        job(`active-${index}`, { state, queuedAt: "2026-10-09T23:59:00Z" }),
    );
    history.update(active);
    for (const entry of oldJobs) history.update([entry]);
    const retained = history.list();
    expect(retained).toHaveLength(200);
    expect(retained.slice(0, 4)).toEqual(active);
    expect(retained.some((entry) => entry.id === "old-0")).toBe(false);
    expect(retained.at(-1)?.id).toBe("old-199");
  });

  it("orders a new startup event after older snapshot rows before pruning the oldest terminal job", () => {
    const history = new JobHistory();
    const request = history.revision;
    const latest = job("latest", {
      ...completed,
      id: "latest",
      queuedAt: "2026-10-10T00:04:00Z",
    });
    history.update([latest]);
    const retained = history.reconcile(oldJobs, request);
    expect(retained).toHaveLength(200);
    expect(retained[0].id).toBe("old-1");
    expect(retained.at(-1)).toEqual(latest);
  });

  it("drops history already pruned by the backend and keeps an active job absent from a snapshot", () => {
    const history = new JobHistory();
    const active = job("active", { ...running, id: "active" });
    history.update([oldJobs[0], active]);
    expect(history.reconcile([oldJobs[1]], history.revision)).toEqual([
      active,
      oldJobs[1],
    ]);
  });

  it("never evicts an active record even if the incoming set exceeds the history limit", () => {
    const history = new JobHistory();
    const active = Array.from({ length: 201 }, (_, index) =>
      job(`active-${index}`, { state: "running" }),
    );
    expect(history.update(active)).toEqual(active);
  });
});
