import { describe, expect, it } from "vitest";
import type { Capabilities, EnvironmentStatus } from "./bridge";
import { summarizeProcessingReadiness } from "./readiness";

const environment: EnvironmentStatus = {
  platform: "windows",
  ready: true,
  ffmpeg: {
    path: "C:\\tools\\ffmpeg.exe",
    version: "ffmpeg version test",
    source: "environment",
    available: true,
  },
  ffprobe: {
    path: "C:\\tools\\ffprobe.exe",
    version: "ffprobe version test",
    source: "environment",
    available: true,
  },
  message: "",
  configurable: true,
};

const capabilities: Capabilities = {
  version: "ffmpeg version test",
  encoders: ["libx265", "hevc_nvenc"],
  codecs: [],
  gpus: [{ name: "Test GPU", vendor: "NVIDIA" }],
  videoEncoders: [
    {
      name: "hevc_nvenc",
      codec: "hevc",
      kind: "gpu",
      label: "NVIDIA NVENC",
      available: true,
      reason: "real encode test passed",
      pixelFormats: [],
    },
  ],
};

describe("processing readiness", () => {
  it("waits for verified FFmpeg before enabling media import", () => {
    const readiness = summarizeProcessingReadiness({
      connected: true,
      environment: null,
      capabilities: null,
    });
    expect(readiness.canChooseMedia).toBe(false);
    expect(readiness.ffmpeg.tone).toBe("checking");
    expect(readiness.acceleration.tone).toBe("checking");
  });

  it("surfaces validated GPU acceleration without hiding CPU fallback", () => {
    const gpu = summarizeProcessingReadiness({
      connected: true,
      environment,
      capabilities,
    });
    expect(gpu.canChooseMedia).toBe(true);
    expect(gpu.ffmpeg.detail).toBe("test");
    expect(gpu.acceleration).toMatchObject({
      tone: "ready",
      label: "GPU 加速可用",
      detailValues: ["NVIDIA NVENC"],
    });

    const cpu = summarizeProcessingReadiness({
      connected: true,
      environment,
      capabilities: { ...capabilities, videoEncoders: [] },
    });
    expect(cpu.canChooseMedia).toBe(true);
    expect(cpu.acceleration).toMatchObject({
      tone: "fallback",
      label: "CPU 编码可用",
    });
  });

  it("keeps automatic processing available when hardware detection fails", () => {
    const readiness = summarizeProcessingReadiness({
      connected: false,
      environment,
      capabilities: null,
      capabilityError: "GPU probe failed",
    });
    expect(readiness.canChooseMedia).toBe(false);
    expect(readiness.acceleration.tone).toBe("attention");
    expect(readiness.storage.tone).toBe("attention");
  });
});
