import type { Job } from "./bridge";

const activeStates = new Set(["queued", "preparing", "running", "cancelling"]);

export type JobActivitySummary = {
  activeCount: number;
  runningCount: number;
  queuedCount: number;
  progress: number;
  paused: boolean;
  current: Job | null;
};

export function summarizeJobActivity(
  jobs: Job[],
  queuePaused = false,
): JobActivitySummary {
  const active = jobs.filter((job) =>
    activeStates.has(job.state.toLowerCase()),
  );
  const running = active.filter((job) =>
    ["preparing", "running"].includes(job.state.toLowerCase()),
  );
  const current =
    running.find((job) => job.state.toLowerCase() === "running") ??
    running[0] ??
    active.find((job) => job.state.toLowerCase() === "queued") ??
    null;
  const progress = Math.min(
    1,
    Math.max(
      0,
      current?.state.toLowerCase() === "running" ? current.progress || 0 : 0,
    ),
  );
  return {
    activeCount: active.length,
    runningCount: running.length,
    queuedCount: active.filter((job) => job.state.toLowerCase() === "queued")
      .length,
    progress,
    paused: queuePaused && running.length === 0,
    current,
  };
}
