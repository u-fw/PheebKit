import { useState } from "react";
import type { ExecutionPlan } from "../bridge";
import { errorMessage } from "../format";
import { Icon } from "./Icon";

export function CommandPreview({
  plan,
  loading,
  error,
}: {
  plan: ExecutionPlan | null;
  loading: boolean;
  error: string;
}) {
  const [copyStatus, setCopyStatus] = useState("");
  async function copy() {
    if (!plan) return;
    try {
      await navigator.clipboard.writeText(plan.preview);
      setCopyStatus("已复制");
    } catch (cause) {
      setCopyStatus(`复制失败：${errorMessage(cause)}`);
    }
  }
  return (
    <section className="command-card surface" aria-label="执行参数预览">
      <div className="section-title">
        <h3>
          <Icon name="terminal" />
          Command Preview
        </h3>
        <button
          className="button text compact-button"
          disabled={!plan || loading}
          onClick={() => void copy()}
        >
          <Icon name="copy" />
          复制
        </button>
      </div>
      <p>由 Go Core 根据当前设置生成。</p>
      <pre aria-live="polite">
        {loading
          ? "正在生成执行计划…"
          : plan?.preview ||
            error ||
            "导入媒体并选择输出文件后，执行参数会显示在这里。"}
      </pre>
      {plan && (
        <span className="subtle-label">
          {plan.args.length} 个独立参数 · 平台 Runner 执行
        </span>
      )}
      {copyStatus && (
        <span className="copy-status" role="status">
          {copyStatus}
        </span>
      )}
    </section>
  );
}
