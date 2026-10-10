import { useState } from "react";
import type { FFmpegSetupStatus } from "../bridge";
import { errorMessage, formatBytes } from "../format";
import { MaterialSelect } from "./MaterialSelect";

const stages: Record<string, string> = {
  selecting: "正在选择下载线路…",
  downloading: "正在下载",
  verifying: "正在校验 SHA-256…",
  extracting: "正在解压…",
  configuring: "正在验证并保存配置…",
  retrying: "正在切换备用来源…",
  cancelling: "正在取消并清理…",
  cancelled: "下载已取消，可以重试。",
  completed: "安装完成",
  failed: "下载未完成",
};

export function FFmpegDownload({
  status,
  enabled,
  busy,
  start,
  cancel,
}: {
  status: FFmpegSetupStatus | null;
  enabled: boolean;
  busy: boolean;
  start: (source: string) => Promise<void>;
  cancel: () => Promise<void>;
}) {
  const [source, setSource] = useState("auto");
  const [requestBusy, setRequestBusy] = useState(false);
  const [error, setError] = useState("");
  const active = Boolean(status?.active);
  async function perform(action: () => Promise<void>) {
    setError("");
    setRequestBusy(true);
    try {
      await action();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setRequestBusy(false);
    }
  }
  const progress =
    status?.stage === "downloading" && status.total > 0
      ? Math.min(100, Math.max(0, (status.received / status.total) * 100))
      : undefined;
  return (
    <div className="ffmpeg-download">
      <div>
        <h3>一键补全</h3>
        <p className="field-hint">
          安装到 kompeg 私有目录，无需管理员权限。保留现有系统 PATH。
        </p>
      </div>
      <MaterialSelect
        label="下载来源"
        value={source}
        onChange={setSource}
        disabled={!enabled || busy || active || requestBusy}
        options={[
          { value: "auto", label: "自动 · 稳定版优先，国内备用" },
          { value: "stable", label: "Gyan 9.0.2 · 稳定版完整构建" },
          { value: "cn", label: "国内镜像 6.1.1 · 较旧版本" },
        ]}
      />
      <p className="field-hint">
        稳定版约 246 MiB；国内备用约 56
        MiB，版本较旧，部分编码器可能不可用。自动模式会先尝试稳定版，失败后切换国内备用；最终版本显示在下方。
      </p>
      {status?.stage && (
        <div className="download-status">
          <p role="status" aria-live="polite">
            {stages[status.stage] || status.stage}
            {status.version ? ` · ${status.version}` : ""}
          </p>
          {status.package && <p className="field-hint">{status.package}</p>}
          {active && (
            <progress aria-label="FFmpeg 下载进度" max={100} value={progress} />
          )}
          {status.stage === "downloading" && (
            <p className="field-hint">
              {formatBytes(status.received)} /{" "}
              {status.total > 0 ? formatBytes(status.total) : "未知大小"} ·{" "}
              {formatBytes(status.speed)}/s
            </p>
          )}
          {status.source && (
            <p className="download-source">
              线路：{new URL(status.source).hostname}
            </p>
          )}
          {status.error && (
            <p className="inline-error" role="alert">
              {status.error}
            </p>
          )}
        </div>
      )}
      <div className="settings-actions">
        {active ? (
          <button
            className="button outlined"
            disabled={
              requestBusy ||
              status?.stage === "configuring" ||
              status?.stage === "cancelling"
            }
            onClick={() => void perform(cancel)}
          >
            取消下载
          </button>
        ) : (
          <button
            className="button primary"
            disabled={!enabled || busy || requestBusy}
            onClick={() => void perform(() => start(source))}
          >
            {requestBusy
              ? "正在开始…"
              : status?.stage === "failed" || status?.stage === "cancelled"
                ? "重试下载"
                : "下载并配置"}
          </button>
        )}
      </div>
      {error && (
        <p className="inline-error" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
