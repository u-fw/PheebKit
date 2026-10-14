import { useState } from "react";
import { API } from "../bridge";
import { errorMessage } from "../format";

export function DiagnosticsPanel({ connected }: { connected: boolean }) {
  const [includeJobs, setIncludeJobs] = useState(false);
  const [jobLimit, setJobLimit] = useState(20);
  const [preview, setPreview] = useState<{
    token: string;
    json: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const invalid =
    includeJobs &&
    (!Number.isInteger(jobLimit) || jobLimit < 1 || jobLimit > 50);
  async function generate() {
    setBusy(true);
    setMessage("");
    setPreview(null);
    try {
      setPreview(
        await API.getDiagnosticPreview(
          includeJobs,
          includeJobs ? jobLimit : 20,
        ),
      );
    } catch (cause) {
      setMessage(`生成失败：${errorMessage(cause)}`);
    } finally {
      setBusy(false);
    }
  }
  async function save() {
    if (!preview) return;
    setBusy(true);
    setMessage("");
    try {
      const path = await API.exportDiagnosticReport(preview.token);
      if (path)
        setMessage(`报告已保存：${path}。确认内容后，可自行附到问题反馈中。`);
    } catch (cause) {
      setMessage(`保存失败：${errorMessage(cause)}`);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="surface preferences-card diagnostics-card">
      <div className="section-title">
        <h2>故障诊断</h2>
        <span className="subtle-label">本地预览与导出</span>
      </div>
      <p className="field-hint">
        包含软件/系统版本、FFmpeg
        版本、编码器可用性及任务状态统计。使用当前检测结果，需要更新时先重新检测。
      </p>
      <p className="field-hint">
        不包含路径、文件名、用户名、命令参数、原始日志、错误原文、下载链接和媒体内容。报告不会自动上传。
      </p>
      <label className="track-choice">
        <input
          type="checkbox"
          checked={includeJobs}
          disabled={busy}
          onChange={(event) => {
            setIncludeJobs(event.target.checked);
            setPreview(null);
            setMessage("");
          }}
        />
        <span>包含最近任务的结构化错误摘要</span>
      </label>
      <label className="field-label">
        摘要数量（1–50）
        <input
          type="number"
          min="1"
          max="50"
          step="1"
          value={jobLimit}
          disabled={!includeJobs || busy}
          onChange={(event) => {
            setJobLimit(Number(event.target.value));
            setPreview(null);
            setMessage("");
          }}
        />
      </label>
      <div className="action-row">
        <button
          className="button outlined"
          disabled={!connected || busy || invalid}
          onClick={() => void generate()}
        >
          {busy ? "处理中…" : "生成诊断预览"}
        </button>
        <button
          className="button primary"
          disabled={!connected || busy || !preview || invalid}
          onClick={() => void save()}
        >
          保存报告…
        </button>
      </div>
      {preview && (
        <>
          <p className="field-hint">
            保存内容与下方预览一致；预览有效期 10 分钟，过期后请重新生成。
          </p>
          <pre
            className="diagnostic-preview"
            tabIndex={0}
            aria-label="脱敏诊断内容"
          >
            {preview.json}
          </pre>
        </>
      )}
      {message && (
        <p className="field-hint" role="status">
          {message}
        </p>
      )}
    </section>
  );
}
