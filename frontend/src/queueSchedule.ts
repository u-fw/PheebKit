import type { Job } from "./bridge";

export interface QueueScheduleStatus {
  paused: boolean;
  pendingIDs: string[];
  revision: number;
}

export function latestQueueStatus(
  current: QueueScheduleStatus | null,
  incoming: QueueScheduleStatus,
): QueueScheduleStatus {
  return !current || incoming.revision >= current.revision ? incoming : current;
}

export function estimatedRemainingSeconds(job: Job): number | null {
  if (
    job.state.toLowerCase() !== "running" ||
    ![job.currentTime, job.totalTime, job.speed].every(Number.isFinite) ||
    job.totalTime <= 0 ||
    job.currentTime < 0 ||
    job.currentTime > job.totalTime ||
    job.speed <= 0
  ) {
    return null;
  }
  return Math.ceil((job.totalTime - job.currentTime) / job.speed);
}

// Execution order stays visible even while the history panel shows newest first.
export function orderQueueJobs(jobs: Job[], pendingIDs?: string[]): Job[] {
  const waiting = new Map((pendingIDs ?? []).map((id, index) => [id, index]));
  const category = (job: Job): number => {
    if (
      ["preparing", "running", "cancelling"].includes(job.state.toLowerCase())
    )
      return 0;
    return job.state.toLowerCase() === "queued" ? 1 : 2;
  };
  const original = new Map(jobs.map((job, index) => [job.id, index]));
  return [...jobs].sort((a, b) => {
    const delta = category(a) - category(b);
    if (delta) return delta;
    if (category(a) === 1)
      return (
        (waiting.get(a.id) ?? original.get(a.id) ?? 0) -
        (waiting.get(b.id) ?? original.get(b.id) ?? 0)
      );
    return (original.get(b.id) ?? 0) - (original.get(a.id) ?? 0);
  });
}
