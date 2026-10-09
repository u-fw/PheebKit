import { useState } from "react";
import type { BinaryStatus, Capabilities, EnvironmentStatus } from "../bridge";
import { errorMessage } from "../format";
import { Icon } from "./Icon";

function BinaryDetails({
  label,
  binary,
}: {
  label: string;
  binary: BinaryStatus | undefined;
}) {
  return (
    <div className="binary-details">
      <div className="binary-heading">
        <h3>{label}</h3>
        <span className={`status-pill ${binary?.available ? "connected" : ""}`}>
          {binary?.available ? "已就绪" : "未找到或不可用"}
        </span>
      </div>
      <dl>
        <div>
          <dt>实际路径</dt>
          <dd>{binary?.path || "尚未配置"}</dd>
        </div>
        <div>
          <dt>来源</dt>
          <dd>
            {binary?.source === "config"
              ? "kompeg 本机配置"
              : binary?.source === "PATH"
                ? "PATH 环境变量"
                : binary?.source
                  ? `环境变量 ${binary.source}`
                  : "未发现"}
          </dd>
        </div>
      </dl>
      {binary?.version && (
        <p className="binary-version" title={binary.version}>
          {binary.version.split("\n")[0]}
        </p>
      )}
    </div>
  );
}

export function SettingsPage({
  platform,
  connected,
  capabilities,
  capabilityError,
  environment,
  environmentBusy,
  refreshEnvironment,
  chooseFFmpeg,
  configureFFmpeg,
  theme,
  toggleTheme,
}: {
  platform: string;
  connected: boolean;
  capabilities: Capabilities | null;
  capabilityError: string;
  environment: EnvironmentStatus | null;
  environmentBusy: boolean;
  refreshEnvironment: () => Promise<void>;
  chooseFFmpeg: () => Promise<void>;
  configureFFmpeg: (ffmpegPath: string, ffprobePath: string) => Promise<void>;
  theme: "light" | "dark";
  toggleTheme: () => void;
}) {
  const [ffmpegPath, setFFmpegPath] = useState<string | null>(null);
  const [ffprobePath, setFFprobePath] = useState<string | null>(null);
  const [actionError, setActionError] = useState("");
  const [actionStatus, setActionStatus] = useState("");
  const configurable = connected && Boolean(environment?.configurable);

  async function perform(action: () => Promise<void>, success: string) {
    setActionError("");
    setActionStatus("");
    try {
      await action();
      setActionStatus(success);
    } catch (cause) {
      setActionError(errorMessage(cause));
    }
  }

  return (
    <div className="page settings-page">
      <header className="page-heading">
        <div>
          <div className="section-eyebrow">本机偏好</div>
          <h1>设置</h1>
          <p>外观、FFmpeg 路径与设备编码能力。</p>
        </div>
      </header>
      <section className="surface preferences-card">
        <div className="section-title">
          <h2>FFmpeg 环境</h2>
          <span
            className={`status-pill ${environment?.ready ? "connected" : ""}`}
          >
            {environmentBusy || !environment
              ? "检测中…"
              : environment.ready
                ? "可以处理媒体"
                : "需要补全"}
          </span>
        </div>
        <p className="settings-description">
          优先使用环境变量中的 FFmpeg 和 ffprobe；缺少时可选择本机文件补全。
        </p>
        <div className="binary-grid">
          <BinaryDetails label="FFmpeg" binary={environment?.ffmpeg} />
          <BinaryDetails label="ffprobe" binary={environment?.ffprobe} />
        </div>
        {environment?.message && (
          <div
            className={`notice ${environment.ready ? "" : "error-notice"}`}
            role={environment.ready ? undefined : "alert"}
          >
            <Icon name="info" />
            <p>{environment.message}</p>
          </div>
        )}
        <div className="settings-actions">
          <button
            className="button primary"
            disabled={!configurable || environmentBusy}
            onClick={() => void perform(chooseFFmpeg, "")}
          >
            <Icon name="folder" />
            选择 FFmpeg…
          </button>
          <button
            className="button outlined"
            disabled={!connected || environmentBusy}
            onClick={() =>
              void perform(
                refreshEnvironment,
                "检测完成。请查看当前路径与设备状态。",
              )
            }
          >
            {environmentBusy ? "正在检测…" : "重新检测"}
          </button>
        </div>
        <details className="manual-environment">
          <summary>手动填写路径</summary>
          <div className="manual-fields">
            <label className="field-label">
              FFmpeg 可执行文件
              <input
                value={ffmpegPath ?? environment?.ffmpeg.path ?? ""}
                onChange={(event) => setFFmpegPath(event.target.value)}
                placeholder="C:\工具\ffmpeg\bin\ffmpeg.exe"
                spellCheck={false}
                disabled={!configurable || environmentBusy}
              />
            </label>
            <label className="field-label">
              ffprobe 可执行文件
              <input
                value={ffprobePath ?? environment?.ffprobe.path ?? ""}
                onChange={(event) => setFFprobePath(event.target.value)}
                placeholder="C:\工具\ffmpeg\bin\ffprobe.exe"
                spellCheck={false}
                disabled={!configurable || environmentBusy}
              />
            </label>
            <p className="field-hint">
              仅保存到 kompeg 本机配置。环境变量仍具有更高优先级。留空 ffprobe
              时查找 FFmpeg 同目录下的 ffprobe。
            </p>
            <button
              className="button outlined"
              disabled={
                !configurable ||
                environmentBusy ||
                !(ffmpegPath ?? environment?.ffmpeg.path ?? "").trim()
              }
              onClick={() =>
                void perform(
                  () =>
                    configureFFmpeg(
                      (ffmpegPath ?? environment?.ffmpeg.path ?? "").trim(),
                      (ffprobePath ?? environment?.ffprobe.path ?? "").trim(),
                    ),
                  "配置已验证。请查看上方实际使用路径。",
                )
              }
            >
              验证并保存
            </button>
          </div>
        </details>
        <p className="field-hint discovery-hint">
          查找顺序：KOMPEG_FFMPEG / KOMPEG_FFPROBE → FFMPEG_PATH / FFPROBE_PATH
          → 本机配置 → PATH。
        </p>
        {actionError && (
          <p className="inline-error" role="alert">
            {actionError}
          </p>
        )}
        {actionStatus && (
          <p className="settings-status" role="status">
            {actionStatus}
          </p>
        )}
      </section>
      <section className="surface preferences-card">
        <div className="section-title">
          <h2>GPU 与编码器</h2>
          <span className="subtle-label">自动模式优先 GPU</span>
        </div>
        <p className="settings-description">
          GPU 编码器通过实际编码测试后才会启用。自动模式会在 GPU 不可用时选用
          CPU，压缩页显示最终使用方式。
        </p>
        {capabilities ? (
          <>
            <div className="gpu-devices">
              {capabilities.gpus.length ? (
                capabilities.gpus.map((gpu) => (
                  <div className="gpu-device" key={`${gpu.vendor}-${gpu.name}`}>
                    <Icon name="video" />
                    <div>
                      <strong>{gpu.name}</strong>
                      <span>{gpu.vendor}</span>
                    </div>
                  </div>
                ))
              ) : (
                <p className="field-hint">
                  没有检测到 GPU 设备信息；编码器可用性仍以实际测试为准。
                </p>
              )}
            </div>
            <details className="encoder-disclosure">
              <summary>
                编码器实测结果 ·{" "}
                {
                  capabilities.videoEncoders.filter(
                    (encoder) => encoder.kind === "gpu" && encoder.available,
                  ).length
                }{" "}
                个 GPU 编码器可用
              </summary>
              <div className="gpu-encoders">
                {capabilities.videoEncoders
                  .filter((encoder) => encoder.kind === "gpu")
                  .map((encoder) => (
                    <div
                      className={`gpu-encoder ${encoder.available ? "available" : ""}`}
                      key={encoder.name}
                    >
                      <div>
                        <strong>{encoder.label}</strong>
                        <code>{encoder.name}</code>
                      </div>
                      <span
                        className={`status-pill ${encoder.available ? "connected" : ""}`}
                      >
                        {encoder.available ? "可用" : "不可用"}
                      </span>
                      <p>
                        {encoder.reason ||
                          (encoder.available
                            ? "实际编码测试通过"
                            : "当前设备或驱动不支持")}
                      </p>
                    </div>
                  ))}
              </div>
            </details>
            <details className="encoder-disclosure">
              <summary>
                全部 FFmpeg 编码器 · {capabilities.encoders.length}
              </summary>
              <div className="encoder-chips">
                {capabilities.encoders.map((encoder) => (
                  <span key={encoder} className="chip">
                    {encoder}
                  </span>
                ))}
              </div>
              <pre className="version-text">
                {capabilities.version || "未返回版本信息"}
              </pre>
            </details>
          </>
        ) : (
          <div className="notice">
            <Icon name="info" />
            <p>{capabilityError || "正在检测设备和可用编码器…"}</p>
          </div>
        )}
      </section>
      <section className="surface preferences-card">
        <div className="section-title">
          <h2>外观</h2>
          <Icon name="sun" />
        </div>
        <div className="preference-row">
          <div>
            <h3>主题</h3>
            <p>使用明亮或深色工作台。</p>
          </div>
          <button className="button outlined" onClick={toggleTheme}>
            <Icon name={theme === "light" ? "moon" : "sun"} />
            {theme === "light" ? "切换深色" : "切换浅色"}
          </button>
        </div>
      </section>
      {platform === "android" && (
        <div className="notice">
          <Icon name="info" />
          <p>Android 媒体处理仍在开发中，当前版本暂不可执行压缩任务。</p>
        </div>
      )}
    </div>
  );
}
