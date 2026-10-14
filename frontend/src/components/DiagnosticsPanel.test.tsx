import { describe, it, expect, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { DiagnosticsPanel } from "./DiagnosticsPanel";
vi.mock("../bridge", () => ({ API: {} }));
describe("diagnostic export controls", () => {
  it("requires a preview before saving and explains the privacy boundary", () => {
    const markup = renderToStaticMarkup(<DiagnosticsPanel connected />);
    expect(markup).toContain("生成诊断预览");
    expect(markup).toMatch(/disabled="">保存报告/);
    expect(markup).toContain("不包含路径");
    expect(markup).toContain("不会自动上传");
    expect(markup).toContain("结构化错误摘要");
  });
});
