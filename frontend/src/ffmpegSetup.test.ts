import { describe, expect, it } from "vitest";
import type { FFmpegSetupStatus } from "./bridge";
import { latestFFmpegSetup } from "./ffmpegSetup";

function status(sequence: number): FFmpegSetupStatus {
  return {
    stage: sequence === 2 ? "completed" : "downloading",
    package: "Gyan",
    source: "https://example.test/ffmpeg.zip",
    file: "ffmpeg.exe",
    received: sequence,
    total: 2,
    speed: 1,
    active: sequence !== 2,
    error: "",
    version: sequence === 2 ? "9.0.2" : "",
    sequence,
  };
}

describe("FFmpeg setup event ordering", () => {
  it("ignores late snapshots while retaining an equal-sequence status refresh", () => {
    const newest = status(2);
    expect(latestFFmpegSetup(newest, status(1))).toBe(newest);
    const refreshed = status(2);
    expect(latestFFmpegSetup(newest, refreshed)).toBe(refreshed);
    expect(latestFFmpegSetup(null, newest)).toBe(newest);
  });
});
