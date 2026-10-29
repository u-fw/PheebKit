// Reconcile only when an event gap needs it. Requests coalesce while a read is
// in flight; failed reads keep their retry even if no further event arrives.
export function createQueueReconciler(
  reconcile: () => Promise<void>,
  reportFailure: (cause: unknown) => void,
) {
  let disposed = false;
  let pending = false;
  let running = false;
  let retryDelay = 500;
  let timer: ReturnType<typeof setTimeout> | undefined;

  function schedule(delay: number) {
    if (disposed || running || timer !== undefined) return;
    timer = setTimeout(() => void run(), delay);
  }

  async function run() {
    timer = undefined;
    if (disposed) return;
    running = true;
    pending = false;
    let delay = 0;
    try {
      await reconcile();
      retryDelay = 500;
    } catch (cause) {
      pending = true;
      delay = retryDelay;
      retryDelay = Math.min(retryDelay * 2, 30_000);
      if (!disposed) reportFailure(cause);
    } finally {
      running = false;
      if (pending) schedule(delay);
    }
  }

  return {
    request() {
      if (disposed) return;
      pending = true;
      schedule(0);
    },
    dispose() {
      disposed = true;
      if (timer !== undefined) clearTimeout(timer);
    },
  };
}
