import { afterEach, describe, expect, it, vi } from "vitest";
import { createQueueReconciler } from "./queueReconciler";

afterEach(() => vi.useRealTimers());

describe("queue snapshot recovery", () => {
  it("retries a failed final snapshot without waiting for another event", async () => {
    vi.useFakeTimers();
    const failure = new Error("IPC temporarily unavailable");
    const read = vi
      .fn()
      .mockRejectedValueOnce(failure)
      .mockResolvedValue(undefined);
    const report = vi.fn();
    const reconciler = createQueueReconciler(read, report);
    reconciler.request();
    await vi.advanceTimersByTimeAsync(0);
    expect(read).toHaveBeenCalledTimes(1);
    expect(report).toHaveBeenCalledWith(failure);
    await vi.advanceTimersByTimeAsync(499);
    expect(read).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(read).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(60_000);
    expect(read).toHaveBeenCalledTimes(2);
    reconciler.dispose();
  });

  it("coalesces requests and never starts concurrent snapshot reads", async () => {
    vi.useFakeTimers();
    let finish!: () => void;
    const read = vi
      .fn()
      .mockImplementationOnce(
        () =>
          new Promise<void>((resolve) => {
            finish = resolve;
          }),
      )
      .mockResolvedValue(undefined);
    const reconciler = createQueueReconciler(read, vi.fn());
    reconciler.request();
    reconciler.request();
    await vi.advanceTimersByTimeAsync(0);
    expect(read).toHaveBeenCalledTimes(1);
    reconciler.request();
    reconciler.request();
    await vi.advanceTimersByTimeAsync(1000);
    expect(read).toHaveBeenCalledTimes(1);
    finish();
    await vi.advanceTimersByTimeAsync(0);
    expect(read).toHaveBeenCalledTimes(2);
    reconciler.dispose();
  });

  it("backs off up to 30 seconds and resets after a successful recovery", async () => {
    vi.useFakeTimers();
    const read = vi.fn().mockRejectedValue(new Error("offline"));
    const reconciler = createQueueReconciler(read, vi.fn());
    reconciler.request();
    await vi.advanceTimersByTimeAsync(0);
    let calls = 1;
    for (const delay of [500, 1000, 2000, 4000, 8000, 16_000, 30_000, 30_000]) {
      await vi.advanceTimersByTimeAsync(delay - 1);
      expect(read).toHaveBeenCalledTimes(calls);
      await vi.advanceTimersByTimeAsync(1);
      expect(read).toHaveBeenCalledTimes(++calls);
    }
    read.mockResolvedValueOnce(undefined);
    await vi.advanceTimersByTimeAsync(30_000);
    expect(read).toHaveBeenCalledTimes(++calls);
    reconciler.request();
    await vi.advanceTimersByTimeAsync(0);
    expect(read).toHaveBeenCalledTimes(++calls);
    await vi.advanceTimersByTimeAsync(500);
    expect(read).toHaveBeenCalledTimes(calls + 1);
    reconciler.dispose();
  });

  it("cancels pending retries and ignores failures after disposal", async () => {
    vi.useFakeTimers();
    let reject!: (cause: unknown) => void;
    const read = vi.fn().mockImplementation(
      () =>
        new Promise<void>((_, fail) => {
          reject = fail;
        }),
    );
    const report = vi.fn();
    const reconciler = createQueueReconciler(read, report);
    reconciler.request();
    await vi.advanceTimersByTimeAsync(0);
    reconciler.dispose();
    reject(new Error("late error"));
    await vi.advanceTimersByTimeAsync(60_000);
    reconciler.request();
    await vi.advanceTimersByTimeAsync(0);
    expect(read).toHaveBeenCalledTimes(1);
    expect(report).not.toHaveBeenCalled();

    const failed = vi.fn().mockRejectedValue(new Error("offline"));
    const retry = createQueueReconciler(failed, report);
    retry.request();
    await vi.advanceTimersByTimeAsync(0);
    retry.dispose();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(failed).toHaveBeenCalledTimes(1);
  });
});
