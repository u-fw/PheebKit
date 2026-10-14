import { describe, expect, it } from "vitest";
import { jobErrorHelp } from "./jobError";

describe("output verification failure help", () => {
  it("explains that output was not published", () => {
    expect(
      jobErrorHelp("verify output structure: expected hevc video, found h264"),
    ).toContain("结果未发布");
    expect(jobErrorHelp("FFmpeg failed: encoder unavailable")).toContain(
      "编码器",
    );
  });
});
