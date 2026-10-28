import type { Job } from "./bridge";

const historyLimit = 200;
const terminalStates = new Set([
  "completed",
  "failed",
  "cancelled",
  "interrupted",
]);
const stateOrder: Record<string, number> = {
  queued: 0,
  preparing: 1,
  running: 2,
  cancelling: 3,
  completed: 4,
  failed: 4,
  cancelled: 4,
  interrupted: 4,
};

function compareTimestamp(left = "", right = ""): number {
  const leftTime = Date.parse(left);
  const rightTime = Date.parse(right);
  if (!Number.isFinite(leftTime) || !Number.isFinite(rightTime)) return 0;
  if (leftTime !== rightTime) return leftTime - rightTime;
  // Go timestamps retain nanoseconds; Date.parse alone loses ordering within a millisecond.
  const fraction = (value: string) =>
    (value.match(/\.(\d+)/)?.[1] ?? "").padEnd(9, "0");
  return fraction(left).localeCompare(fraction(right));
}

function newestJob(current: Job | undefined, incoming: Job): Job {
  if (!current) return incoming;
  if (current.eventSequence || incoming.eventSequence) {
    if (
      current.eventSequence &&
      incoming.eventSequence &&
      incoming.eventSequence < current.eventSequence
    )
      return current;
  }
  if ((stateOrder[incoming.state] ?? 0) < (stateOrder[current.state] ?? 0))
    return current;
  if (terminalStates.has(current.state)) {
    return compareTimestamp(incoming.completedAt, current.completedAt) > 0
      ? incoming
      : current;
  }
  if (
    incoming.state === current.state &&
    (incoming.progress < current.progress ||
      incoming.currentTime < current.currentTime)
  )
    return current;
  return incoming;
}

interface Entry {
  job: Job;
  revision: number;
}

// Events are batched for rendering, but enter this store immediately. A Jobs()
// reply can therefore never erase an event received after its request began.
export class JobHistory {
  private entries = new Map<string, Entry>();
  private currentRevision = 0;
  private readEventSequence = 0;

  get revision(): number {
    return this.currentRevision;
  }

  update(jobs: Job[]): Job[] {
    return this.updateWithSequence(jobs).jobs;
  }

  updateWithSequence(jobs: Job[]): { jobs: Job[]; sequenceGap: boolean } {
    const revision = ++this.currentRevision;
    let sequenceGap = false;
    for (const job of jobs) {
      if (
        job.eventSequence &&
        job.eventSequence > this.readEventSequence &&
        job.eventSequence !== this.readEventSequence + 1
      ) {
        sequenceGap = true;
      }
      if (job.eventSequence && job.eventSequence > this.readEventSequence)
        this.readEventSequence = job.eventSequence;
      const current = this.entries.get(job.id)?.job;
      this.entries.set(job.id, { job: newestJob(current, job), revision });
    }
    return { jobs: this.trim(), sequenceGap };
  }

  reconcile(jobs: Job[], requestedAt: number): Job[] {
    const revision = ++this.currentRevision;
    const next = new Map<string, Entry>();
    for (const job of jobs) {
      if (job.eventSequence && job.eventSequence > this.readEventSequence)
        this.readEventSequence = job.eventSequence;
      const current = this.entries.get(job.id);
      next.set(
        job.id,
        current && current.revision > requestedAt
          ? current
          : { job: newestJob(current?.job, job), revision },
      );
    }
    for (const [id, entry] of this.entries) {
      if (
        !next.has(id) &&
        (entry.revision > requestedAt || !terminalStates.has(entry.job.state))
      )
        next.set(id, entry);
    }
    this.entries = next;
    return this.trim();
  }

  list(): Job[] {
    return [...this.entries.values()]
      .map(({ job }) => job)
      .sort((left, right) => compareTimestamp(left.queuedAt, right.queuedAt));
  }

  private trim(): Job[] {
    const ordered = this.list();
    let excess = ordered.length - historyLimit;
    for (const job of ordered) {
      if (excess <= 0) break;
      if (terminalStates.has(job.state)) {
        this.entries.delete(job.id);
        excess--;
      }
    }
    return ordered.filter((job) => this.entries.has(job.id));
  }
}
