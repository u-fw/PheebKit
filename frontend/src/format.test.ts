import { describe, expect, it } from "vitest";
import {
  formatBytes,
  formatDuration,
  savedPercent,
  suggestedOutput,
} from "./format";
import {
  defaultSettings,
  parseRawArguments,
  validateSettings,
} from "./project";

describe("媒体显示", () => {
  it("以二进制单位显示大小，并保留跨小时的时长", () => {
    expect(formatBytes(1024 ** 3)).toBe("1 GiB");
    expect(formatDuration(6151.99)).toBe("01:42:31");
    expect(formatBytes(-1)).toBe("未知");
  });
  it("保留 Windows Unicode 文件名并选择 Opus 兼容容器", () => {
    expect(suggestedOutput("D:\\媒体\\电影 final.mkv", "hevc", "copy")).toBe(
      "D:\\媒体\\电影 final-compressed.mp4",
    );
    expect(suggestedOutput("content://provider/doc/8", "hevc", "opus")).toBe(
      "compressed-hevc.mkv",
    );
  });
  it("输出变大时明确显示增加，不伪造节省", () => {
    expect(savedPercent(100, 125)).toBe("增加 25.0%");
    expect(savedPercent(0, 125)).toBe("未知");
  });
});

describe("表单边界", () => {
  it("拒绝覆盖源文件、非偶数分辨率和无效码率", () => {
    expect(
      validateSettings(
        {
          ...defaultSettings,
          resolution: 721,
          mode: "bitrate",
          bitrateKbps: 0,
        },
        "source.mp4",
        "source.mp4",
      ),
    ).toHaveLength(3);
    expect(
      validateSettings(defaultSettings, "output.mp4", "source.mp4"),
    ).toEqual([]);
  });
  it("CRF 上限匹配编码族，拒绝非有限数值", () => {
    expect(
      validateSettings(
        { ...defaultSettings, quality: 60 },
        "out.mkv",
        "in.mkv",
      ),
    ).toHaveLength(1);
    expect(
      validateSettings(
        { ...defaultSettings, codec: "av1", quality: 60 },
        "out.mkv",
        "in.mkv",
      ),
    ).toEqual([]);
    expect(
      validateSettings(
        { ...defaultSettings, mode: "size", targetSizeMB: Number.NaN },
        "out.mkv",
        "in.mkv",
      ),
    ).toHaveLength(1);
  });
  it("Raw arguments 每行保留为一个 argv，不解释 shell 字符", () => {
    expect(
      parseRawArguments("-metadata\r\ntitle=电影 $(echo bad)\n\n-preset\nslow"),
    ).toEqual(["-metadata", "title=电影 $(echo bad)", "-preset", "slow"]);
  });
});
