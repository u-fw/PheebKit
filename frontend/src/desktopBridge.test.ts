import { afterEach, describe, expect, it, vi } from "vitest";
import { API } from "./bridge";
import { setLanguage } from "./i18n";

afterEach(() => vi.unstubAllGlobals());

describe("stable desktop runtime contract", () => {
  it("awaits the current locale before opening each native dialog", async () => {
    const calls: string[] = [];
    let complete: (() => void) | undefined;
    vi.stubGlobal("localStorage", { setItem: vi.fn() });
    vi.stubGlobal("window", {
      go: {
        app: {
          Desktop: {
            SetLocale: (language: string) => {
              calls.push(language);
              return new Promise<void>((resolve) => {
                complete = resolve;
              });
            },
            ChooseSubtitleFile: async () => {
              calls.push("dialog");
              return "C:/设置.srt";
            },
          },
        },
      },
    });
    try {
      for (const language of ["en-US", "zh-CN"] as const) {
        setLanguage(language);
        const pending = API.chooseSubtitleFile();
        expect(calls.at(-1)).toBe(language);
        expect(calls.filter((call) => call === "dialog")).toHaveLength(
          language === "en-US" ? 0 : 1,
        );
        complete?.();
        expect(await pending).toBe("C:/设置.srt");
        expect(calls.at(-1)).toBe("dialog");
      }
    } finally {
      setLanguage("zh-CN");
    }
  });
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
    const activated = vi.fn();
    const dispose = API.subscribeQueue(queue);
    API.onCloseRequested(close);
    const disposeActivated = API.subscribeJobActivated(activated);
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
    listeners.get("job:activated")?.("job-42");
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
    expect(activated).toHaveBeenCalledWith("job-42");
    dispose();
    disposeActivated();
    expect(stop).toHaveBeenCalledTimes(2);
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
