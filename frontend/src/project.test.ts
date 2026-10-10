import { describe, expect, it } from "vitest";
import { defaultSettings, validateSettings } from "./project";

describe("compression mode validation", () => {
  it.each([99, -1, 1.5, Number.NaN, Number.POSITIVE_INFINITY])(
    "lets bitrate and target-size mode recover from an invalid quality value (%s)",
    (quality) => {
      const settings = { ...defaultSettings, quality, audio: "aac" };
      expect(validateSettings(settings, "out.mp4", "in.mp4")).toEqual([
        "质量参数必须是 0–51 的整数。",
      ]);
      expect(
        validateSettings({ ...settings, mode: "bitrate" }, "out.mp4", "in.mp4"),
      ).toEqual([]);
      expect(
        validateSettings({ ...settings, mode: "size" }, "out.mp4", "in.mp4"),
      ).toEqual([]);
      expect(validateSettings(settings, "out.mp4", "in.mp4")).toContain(
        "质量参数必须是 0–51 的整数。",
      );
    },
  );

  it("checks the parameter for the selected codec and mode", () => {
    expect(
      validateSettings(
        { ...defaultSettings, codec: "av1", quality: 63 },
        "out.mp4",
        "in.mp4",
      ),
    ).toEqual([]);
    expect(
      validateSettings(
        { ...defaultSettings, quality: 63 },
        "out.mp4",
        "in.mp4",
      ),
    ).toEqual(["质量参数必须是 0–51 的整数。"]);
    expect(
      validateSettings(
        { ...defaultSettings, quality: 99, mode: "bitrate", bitrateKbps: 0 },
        "out.mp4",
        "in.mp4",
      ),
    ).toEqual(["目标视频码率必须大于 0 kbps。"]);
    expect(
      validateSettings(
        { ...defaultSettings, quality: 99, mode: "size", targetSizeMB: 0 },
        "out.mp4",
        "in.mp4",
      ),
    ).toEqual(["目标文件大小必须大于 0 MB。"]);
  });
});
