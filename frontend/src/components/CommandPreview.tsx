import { t } from "../i18n";
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
    <section className="command-card surface" aria-label={t("执行参数预览")}>
      <div className="section-title">
        <h3>
          <Icon name="terminal" />
          {t("执行参数")}
        </h3>
        <button
          className="button text compact-button"
          disabled={!plan || loading}
          onClick={() => void copy()}
        >
          <Icon name="copy" />
          {t("复制")}
        </button>
      </div>
      <p>{t("根据当前设置生成，用于核对实际处理方式。")}</p>
      {plan?.subtitleAsset && (
        <p className="field-hint">
          {t(
            "执行前会在临时目录将所选文字字幕转换为 ASS；下方命令使用该临时目录中的固定字幕文件，不能单独复制运行。",
          )}
        </p>
      )}
      <pre aria-live="polite">
        {t(
          loading
            ? "正在生成执行计划…"
            : plan?.preview ||
                error ||
                "导入媒体并选择输出文件后，执行参数会显示在这里。",
        )}
      </pre>
      {plan && (
        <span className="subtle-label">
          {plan.args.length} {t("个独立参数 · 本地执行")}
        </span>
      )}
      {copyStatus && (
        <span className="copy-status" role="status">
          {t(copyStatus)}
        </span>
      )}
    </section>
  );
}
