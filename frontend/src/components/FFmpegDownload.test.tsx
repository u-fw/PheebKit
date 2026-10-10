import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import type { FFmpegSetupStatus } from "../bridge";
import { FFmpegDownload } from "./FFmpegDownload";

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
const action = async () => undefined;
function markup(value: FFmpegSetupStatus | null, enabled = true) {
  return renderToStaticMarkup(
    <FFmpegDownload
      status={value}
      enabled={enabled}
      busy={false}
      start={action}
      cancel={action}
    />,
  );
}
describe("FFmpeg download feedback", () => {
  it("shows version differences and a named source selector", () => {
    const html = markup(null);
    expect(html).toContain("下载来源");
    expect(html).toContain("较旧");
    expect(html).toContain("下载并配置");
  });
  it("shows real transfer progress and offers cancellation", () => {
    const html = markup(status);
    expect(html).toContain('value="50"');
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
