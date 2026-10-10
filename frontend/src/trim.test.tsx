import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import {
  formatTimecode,
  parseTimecode,
  selectedDuration,
  trimError,
} from "./trim";
import { defaultSettings, validateSettings } from "./project";
import { TrimSettings } from "./components/TrimSettings";
import type { Media } from "./bridge";

const media: Media = {
  source: { uri: "in.mp4", displayName: "in.mp4", kind: "local-file" },
  name: "in.mp4",
  container: "mp4",
  durationSeconds: 143.125,
  sizeBytes: 1000,
  bitrate: 1000,
  streams: [],
};

describe("optional trim range", () => {
  it("accepts duration timecodes, including hours beyond a wall-clock day", () => {
    expect(parseTimecode("01:02:03.125")).toBe(3723.125);
    expect(parseTimecode("02:03.125")).toBe(123.125);
    expect(parseTimecode("123.125")).toBe(123.125);
    expect(parseTimecode("25:00:00")).toBe(90000);
    expect(formatTimecode(143.125)).toBe("00:02:23.125");
    expect(formatTimecode(1.9999)).toBe("00:00:01.999");
  });
  it.each([
    "",
    "-1",
    "1:60",
    "1:60:00",
    "00:00:60",
    "NaN",
    "Infinity",
    "1.1234",
    "1e2",
    "1:2:3:4",
  ])("rejects malformed time %s", (text) => {
    expect(parseTimecode(text)).toBe(-1);
  });
  it("validates boundaries against the source and preserves the whole-file default", () => {
    expect(selectedDuration(defaultSettings, 143.125)).toBe(143.125);
    const settings = {
      ...defaultSettings,
      trim: { startSeconds: 2.5, endSeconds: 3.75 },
    };
    expect(
      validateSettings(settings, "out.mp4", media.source.uri, media),
    ).toEqual([]);
    expect(selectedDuration(settings, media.durationSeconds)).toBe(1.25);
    expect(
      trimError({ ...settings, trim: { startSeconds: 5, endSeconds: 5 } }, 10),
    ).toContain("结束时间");
    expect(
      trimError(
        { ...settings, trim: { startSeconds: 0, endSeconds: 10.1 } },
        10,
      ),
    ).toContain("总时长");
    expect(trimError(settings, 0)).toContain("未知");
    expect(
      trimError(
        { ...settings, trim: { startSeconds: 2, endSeconds: 2.01 } },
        10,
        30,
      ),
    ).toContain("至少需要一帧");
    expect(
      trimError(
        { ...settings, trim: { startSeconds: Number.NaN, endSeconds: 5 } },
        10,
      ),
    ).toContain("有效时间");
  });
  it("provides labeled fields, keyboard sliders and inline invalid-range feedback", () => {
    const markup = renderToStaticMarkup(
      <TrimSettings
        media={media}
        settings={{
          ...defaultSettings,
          trim: { startSeconds: 5, endSeconds: 2 },
        }}
        onChange={() => undefined}
      />,
    );
    expect(markup).toContain('for="trim-startSeconds"');
    expect(markup).toContain('for="trim-endSeconds"');
    expect(markup).toContain('aria-label="开始时间滑块"');
    expect(markup).toContain('aria-invalid="true"');
    expect(markup).toContain('id="trim-error"');
    expect(markup).toContain("结束时间必须晚于开始时间");
    const unknown = renderToStaticMarkup(
      <TrimSettings
        media={{ ...media, durationSeconds: 0 }}
        settings={defaultSettings}
        onChange={() => undefined}
      />,
    );
    expect(unknown).toContain('disabled=""');
    expect(unknown).not.toContain('id="trim-startSeconds"');
  });
});
