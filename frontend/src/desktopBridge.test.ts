import { afterEach, describe, expect, it, vi } from "vitest";
import { API } from "./bridge";

afterEach(() => vi.unstubAllGlobals());

describe("stable desktop runtime contract", () => {
  it("consumes direct v2 event payloads and retains listener cleanup", () => {
    const listeners = new Map<string, (value: unknown) => void>();
    const stop = vi.fn();
    vi.stubGlobal("window", {
      runtime: {
        EventsOnMultiple: (
          name: string,
          callback: (value: unknown) => void,
        ) => {
          listeners.set(name, callback);
          return stop;
        },
      },
    });
    const queue = vi.fn();
    const close = vi.fn();
    const dispose = API.subscribeQueue(queue);
    API.onCloseRequested(close);
    listeners.get("queue:updated")?.({
      paused: true,
      pendingIDs: null,
      revision: 4,
    });
    listeners.get("app:close-requested")?.({
      jobs: 1,
      downloading: false,
      unsaved: true,
    });
    expect(queue).toHaveBeenCalledWith({
      paused: true,
      pendingIDs: [],
      revision: 4,
    });
    expect(close).toHaveBeenCalledWith({
      jobs: 1,
      downloading: false,
      unsaved: true,
    });
    dispose();
    expect(stop).toHaveBeenCalledOnce();
  });
  it("registers native file drops for the whole workspace and detaches on unmount", () => {
    const attach = vi.fn();
    const detach = vi.fn();
    vi.stubGlobal("window", {
      runtime: { OnFileDrop: attach, OnFileDropOff: detach },
    });
    const receive = vi.fn();
    const dispose = API.onDrop(receive);
    expect(attach).toHaveBeenCalledWith(expect.any(Function), false);
    const callback: (x: number, y: number, paths: string[]) => void =
      attach.mock.calls[0][0];
    callback(10, 20, ["C:/测试 & clip.mp4"]);
    expect(receive).toHaveBeenCalledWith(["C:/测试 & clip.mp4"]);
    dispose();
    expect(detach).toHaveBeenCalledOnce();
  });
});
