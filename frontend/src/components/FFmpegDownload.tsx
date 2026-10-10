import { useState } from "react";
import type { FFmpegSetupStatus } from "../bridge";
import { errorMessage, formatBytes } from "../format";
import { MaterialSelect } from "./MaterialSelect";
import {
  CustomDownloadFields,
  customDownloadError,
} from "./CustomDownloadFields";

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
  startCustom,
  cancel,
}: {
  status: FFmpegSetupStatus | null;
  enabled: boolean;
  busy: boolean;
  start: (source: string) => Promise<void>;
  startCustom: (
    link: string,
    sha256: string,
    trustBuild: boolean,
  ) => Promise<void>;
  cancel: () => Promise<void>;
}) {
  const [source, setSource] = useState("stable");
  const [requestBusy, setRequestBusy] = useState(false);
  const [error, setError] = useState("");
  const [customLink, setCustomLink] = useState("");
  const [customHash, setCustomHash] = useState("");
  const [otherBuild, setOtherBuild] = useState(false);
  const [trusted, setTrusted] = useState(false);
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
  async function startDownload() {
    if (source !== "custom") {
      await start(source);
      return;
    }
    const validation = customDownloadError(
      customLink,
      customHash,
      otherBuild,
      trusted,
    );
    if (validation) throw new Error(validation);
    await startCustom(
      customLink.trim(),
      otherBuild ? customHash.trim() : "",
      otherBuild && trusted,
    );
  }
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
          { value: "stable", label: "官网推荐 · Gyan 9.0.2 完整构建" },
          { value: "cn", label: "国内镜像 6.1.1 · 较旧版本" },
          { value: "custom", label: "自定义下载链接 · 多种压缩格式" },
        ]}
      />
      {source === "custom" ? (
        <CustomDownloadFields
          link={customLink}
          sha256={customHash}
          otherBuild={otherBuild}
          trusted={trusted}
          disabled={!enabled || busy || active || requestBusy}
          onLink={(value) => {
            setCustomLink(value);
            setTrusted(false);
          }}
          onSHA256={(value) => {
            setCustomHash(value);
            setTrusted(false);
          }}
          onOtherBuild={(value) => {
            setOtherBuild(value);
            setTrusted(false);
          }}
          onTrusted={setTrusted}
        />
      ) : (
        <p className="field-hint">
          稳定版约 246 MiB；国内备用约 56
          MiB，版本较旧，部分编码器可能不可用。默认先从官网推荐的 Gyan
          下载，再尝试同版本 GitHub
          备用线路；失败后可手动选择国内镜像或自定义链接。
        </p>
      )}
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
              {progress !== undefined ? `${progress.toFixed(1)}% · ` : ""}
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
            onClick={() => void perform(startDownload)}
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
