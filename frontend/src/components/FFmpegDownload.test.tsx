import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import type { EnvironmentStatus, FFmpegSetupStatus } from "../bridge";
import { FFmpegDownload } from "./FFmpegDownload";
import { legacyBinaryVariables } from "../legacy";

const status: FFmpegSetupStatus = {
  stage: "downloading",
  package: "国内备用 6.1.1",
  source: "https://registry.npmmirror.com/file",
  file: "ffmpeg.exe",
  received: 500,
  total: 1000,
  speed: 100,
  active: true,
  error: "",
  version: "",
};
const missingEnvironment: EnvironmentStatus = {
  platform: "windows",
  ready: false,
  configurable: true,
  message: "",
  ffmpeg: { path: "ffmpeg", source: "PATH", version: "", available: false },
  ffprobe: { path: "ffprobe", source: "PATH", version: "", available: false },
};
const action = async () => undefined;
function markup(
  value: FFmpegSetupStatus | null,
  enabled = true,
  environment: EnvironmentStatus | null = missingEnvironment,
) {
  return renderToStaticMarkup(
    <FFmpegDownload
      status={value}
      environment={environment}
      enabled={enabled}
      busy={false}
      start={action}
      startCustom={action}
      cancel={action}
    />,
  );
}
describe("FFmpeg download feedback", () => {
  it("blocks repeat downloads for a ready PATH or configured environment", () => {
    for (const source of ["PATH", "config"]) {
      const html = markup(null, true, {
        ...missingEnvironment,
        ready: true,
        ffmpeg: { ...missingEnvironment.ffmpeg, available: true, source },
        ffprobe: { ...missingEnvironment.ffprobe, available: true, source },
      });
      expect(html).toContain("无需重复下载");
      expect(html).toMatch(/<button[^>]*disabled=""[^>]*>已就绪，无需下载/);
      expect(html).toContain('aria-describedby="ffmpeg-download-block-reason"');
    }
  });
  it("explains explicit overrides even when the selected program is invalid", () => {
    for (const source of [
      "PHEEBKIT_FFMPEG",
      "FFMPEG_PATH",
      "PHEEBKIT_FFPROBE",
      "FFPROBE_PATH",
      ...legacyBinaryVariables,
    ]) {
      const html = markup(null, true, {
        ...missingEnvironment,
        ffmpeg: { ...missingEnvironment.ffmpeg, source },
      });
      expect(html).toContain(source);
      expect(html).toContain("清除或修正");
      expect(html).toMatch(/<button[^>]*disabled=""[^>]*>下载暂不可用/);
    }
  });
  it("waits for environment detection before enabling download", () => {
    expect(markup(null, true, null)).toContain("正在检测 FFmpeg 环境");
    expect(markup(null, true, null)).toMatch(
      /<button[^>]*disabled=""[^>]*>下载暂不可用/,
    );
  });
  it("shows version differences and a named source selector", () => {
    const html = markup(null);
    expect(html).toContain("下载来源");
    expect(html).toContain("较旧");
    expect(html).toContain("官网推荐 · Gyan");
    expect(html).not.toContain("国内备用</span>");
    expect(html).toContain("下载并配置");
  });
  it("shows real transfer progress and offers cancellation", () => {
    const html = markup(status);
    expect(html).toContain('value="50"');
    expect(html).toContain("50.0%");
    expect(html).toContain('aria-label="FFmpeg 下载进度"');
    expect(html).toContain("取消下载");
    expect(html).toContain("registry.npmmirror.com");
  });
  it("makes extraction indeterminate and configuration uncancellable", () => {
    const html = markup({ ...status, stage: "configuring" });
    expect(html).toContain("正在验证并保存");
    expect(html).not.toContain('value="50"');
    expect(html).toMatch(/<button[^>]*disabled=""[^>]*>取消下载/);
  });
  it("retains failure details and offers retry", () => {
    const html = markup({
      ...status,
      stage: "failed",
      active: false,
      error: "SHA-256 校验失败",
    });
    expect(html).toContain('role="alert"');
    expect(html).toContain("SHA-256 校验失败");
    expect(html).toContain("重试下载");
  });
});
