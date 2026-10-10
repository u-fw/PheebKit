import { describe, expect, it } from "vitest";
import { streamLabel, videoLabel } from "./mediaLabels";
import type { Stream } from "./bridge";

const stream: Stream = {
  index: 0,
  codecType: "video",
  codecName: "aac",
  width: 0,
  height: 0,
  frameRate: 0,
  pixelFormat: "yuv420p",
  bitDepth: 8,
  colorSpace: "bt709",
  colorTransfer: "bt709",
  colorPrimaries: "bt709",
  hdr: "",
  language: "und",
  title: "",
  attachedPicture: false,
  timedThumbnails: false,
};

describe("human-readable media information", () => {
  it("names pixel format and deduplicates standard color properties", () => {
    expect(videoLabel(stream)).toBe("YUV 4:2:0 · 8 位 · BT.709 标准色彩");
    expect(
      videoLabel({
        ...stream,
        colorPrimaries: "bt2020",
        colorTransfer: "smpte2084",
      }),
    ).toContain("BT2020");
  });
  it("explains unspecified language instead of leaking und", () => {
    expect(streamLabel(stream)).toBe("AAC · 语言未标注");
    expect(streamLabel({ ...stream, language: "eng" })).toBe("AAC · 英语");
  });
});
