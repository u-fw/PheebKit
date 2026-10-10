import { describe, expect, it, vi } from "vitest";
import type { Capabilities, EnvironmentStatus } from "./bridge";
import { refreshEnvironmentSnapshot } from "./environmentRefresh";

function environment(path: string, ready = true): EnvironmentStatus {
  return {
    platform: "windows",
    ready,
    configurable: true,
    message: ready ? "" : "FFmpeg 缺失，请补全路径。",
    ffmpeg: {
      path,
      version: "ffmpeg test",
      available: ready,
      source: "config",
    },
    ffprobe: {
      path: `${path}-probe`,
      version: "ffprobe test",
      available: ready,
      source: "config",
    },
  };
}

function capabilities(version: string): Capabilities {
  return { version, encoders: [], codecs: [], videoEncoders: [], gpus: [] };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

describe("environment rescan and shared configuration", () => {
  it("reloads before reading paths and can recover a window whose previous environment was missing", async () => {
    let activeEnvironment = environment("", false);
    const nextEnvironment = environment("C:/new/ffmpeg.exe");
    const nextCapabilities = capabilities("new ffmpeg");
    const calls: string[] = [];
    const publish = vi.fn();
    await refreshEnvironmentSnapshot(
      {
        refreshCapabilities: async () => {
          calls.push("reload-and-scan");
          activeEnvironment = nextEnvironment;
          return nextCapabilities;
        },
        getEnvironment: async () => {
          calls.push("read-actual-paths");
          return activeEnvironment;
        },
      },
      () => true,
      publish,
    );
    expect(calls).toEqual(["reload-and-scan", "read-actual-paths"]);
    expect(publish).toHaveBeenCalledExactlyOnceWith({
      environment: nextEnvironment,
      capabilities: nextCapabilities,
    });
  });

  it("shows the actual old runner paths after invalid external configuration and rejects the action", async () => {
    const scanFailure = new Error("无法同步新配置，本窗口继续使用原有路径。");
    const actualEnvironment = {
      ...environment("C:/old/ffmpeg.exe"),
      message: scanFailure.message,
    };
    const getEnvironment = vi.fn().mockResolvedValue(actualEnvironment);
    const publish = vi.fn();
    await expect(
      refreshEnvironmentSnapshot(
        {
          refreshCapabilities: vi.fn().mockRejectedValue(scanFailure),
          getEnvironment,
        },
        () => true,
        publish,
      ),
    ).rejects.toBe(scanFailure);
    expect(getEnvironment).toHaveBeenCalledOnce();
    expect(publish).toHaveBeenCalledExactlyOnceWith({
      environment: actualEnvironment,
      capabilities: null,
    });
  });

  it("does not report success or usable capabilities while the actual environment is still missing", async () => {
    const missing = environment("", false);
    const publish = vi.fn();
    await expect(
      refreshEnvironmentSnapshot(
        {
          refreshCapabilities: async () => capabilities("ffmpeg only"),
          getEnvironment: async () => missing,
        },
        () => true,
        publish,
      ),
    ).rejects.toThrow(missing.message);
    expect(publish).toHaveBeenCalledExactlyOnceWith({
      environment: missing,
      capabilities: null,
    });
  });

  it("keeps both the new paths and new capabilities when older rescans finish late", async () => {
    let generation = 1;
    const olderScan = deferred<Capabilities>();
    const oldEnvironment = environment("C:/old/ffmpeg.exe");
    const publish = vi.fn();
    const olderRequest = refreshEnvironmentSnapshot(
      {
        refreshCapabilities: () => olderScan.promise,
        getEnvironment: async () => oldEnvironment,
      },
      () => generation === 1,
      publish,
    );
    generation = 2;
    const newestEnvironment = environment("C:/new/ffmpeg.exe");
    const newestCapabilities = capabilities("new ffmpeg");
    await refreshEnvironmentSnapshot(
      {
        refreshCapabilities: async () => newestCapabilities,
        getEnvironment: async () => newestEnvironment,
      },
      () => generation === 2,
      publish,
    );
    olderScan.resolve(capabilities("old ffmpeg"));
    await olderRequest;
    expect(publish).toHaveBeenCalledExactlyOnceWith({
      environment: newestEnvironment,
      capabilities: newestCapabilities,
    });
  });

  it("does not overwrite current environment paths when reading the actual runner status fails", async () => {
    const publish = vi.fn();
    const statusFailure = new Error("原生状态查询失败");
    await expect(
      refreshEnvironmentSnapshot(
        {
          refreshCapabilities: async () => capabilities("new ffmpeg"),
          getEnvironment: vi.fn().mockRejectedValue(statusFailure),
        },
        () => true,
        publish,
      ),
    ).rejects.toBe(statusFailure);
    expect(publish).not.toHaveBeenCalled();
  });
});
