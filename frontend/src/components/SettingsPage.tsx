import { t, useI18n } from "../i18n";
import { MaterialSelect } from "./MaterialSelect";
import { AppearanceSettings } from "./AppearanceSettings";
import { useState, type ReactNode } from "react";
import type {
  BinaryStatus,
  Capabilities,
  EnvironmentStatus,
  FFmpegSetupStatus,
} from "../bridge";
import { errorMessage } from "../format";
import { Icon } from "./Icon";
import { FFmpegDownload, explicitEnvironmentOverride } from "./FFmpegDownload";
import { DiagnosticsPanel } from "./DiagnosticsPanel";
import { FFmpegExplorer } from "./FFmpegExplorer";

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
        <h3>{t(label)}</h3>
        <span className={`status-pill ${binary?.available ? "connected" : ""}`}>
          {t(binary?.available ? "已就绪" : "未找到或不可用")}
        </span>
      </div>
      <dl>
        <div>
          <dt>{t("实际路径")}</dt>
          <dd>{binary?.path || t("尚未配置")}</dd>
        </div>
        <div>
          <dt>{t("来源")}</dt>
          <dd>
            {binary?.source === "config"
              ? t("菲比工具箱 本机配置")
              : binary?.source === "PATH"
                ? t("PATH 环境变量")
                : binary?.source === "bundled"
                  ? t("内置 FFmpeg 运行时")
                  : binary?.source
                    ? t("环境变量 {{0}}", [binary.source])
                    : t("未发现")}
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
  editorSettings,
  systemAccent,
  systemAccentError,
  systemAccentSupported,
  refreshSystemAccent,
  platform,
  connected,
  capabilities,
  capabilityError,
  environment,
  environmentBusy,
  refreshEnvironment,
  chooseFFmpeg,
  configureFFmpeg,
  ffmpegSetup,
  startFFmpegSetup,
  startCustomFFmpegSetup,
  cancelFFmpegSetup,
}: {
  editorSettings?: ReactNode;
  systemAccent: string;
  systemAccentError: "" | "invalid" | "read";
  systemAccentSupported: boolean;
  refreshSystemAccent: () => Promise<void>;
  platform: string;
  connected: boolean;
  capabilities: Capabilities | null;
  capabilityError: string;
  environment: EnvironmentStatus | null;
  environmentBusy: boolean;
  refreshEnvironment: () => Promise<void>;
  chooseFFmpeg: () => Promise<void>;
  configureFFmpeg: (ffmpegPath: string, ffprobePath: string) => Promise<void>;
  ffmpegSetup: FFmpegSetupStatus | null;
  startFFmpegSetup: (source: string) => Promise<void>;
  startCustomFFmpegSetup: (
    link: string,
    sha256: string,
    trustBuild: boolean,
  ) => Promise<void>;
  cancelFFmpegSetup: () => Promise<void>;
}) {
  const [ffmpegPath, setFFmpegPath] = useState<string | null>(null);
  const { language, setLanguage, persistent } = useI18n();
  const [ffprobePath, setFFprobePath] = useState<string | null>(null);
  const [actionError, setActionError] = useState("");
  const [actionStatus, setActionStatus] = useState("");
  const configurable =
    connected &&
    Boolean(environment?.configurable) &&
    !explicitEnvironmentOverride(environment);

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
          <h1>{t("设置")}</h1>
          <p>{t("外观、FFmpeg 路径与设备编码能力。")}</p>
        </div>
      </header>
      <section className="surface preferences-card" id="settings-appearance">
        <div className="section-title">
          <h2>{t("外观")}</h2>
          <Icon name="sun" />
        </div>
        <MaterialSelect
          label={t("界面语言")}
          value={language}
          options={[
            { value: "zh-CN", label: "简体中文", translate: false },
            { value: "en-US", label: "English", translate: false },
          ]}
          onChange={(value) => {
            if (value === "zh-CN" || value === "en-US") setLanguage(value);
          }}
        />
        {!persistent && (
          <p className="inline-error" role="status">
            {t(
              "当前环境无法保存本机偏好，本次切换仍有效。重启后将使用系统语言。",
            )}
          </p>
        )}
        <p className="field-hint">
          {t(
            "语言立即生效并保存在本机，不会重置当前编辑。FFmpeg 原始日志与用户文件名保持原样。",
          )}
        </p>
        <AppearanceSettings
          systemAccent={systemAccent}
          systemAccentError={systemAccentError}
          systemAccentSupported={systemAccentSupported}
          refreshSystemAccent={refreshSystemAccent}
        />
      </section>
      <div id="settings-editor">{editorSettings}</div>
      <section className="surface preferences-card" id="settings-environment">
        <div className="section-title">
          <h2>{t("FFmpeg 环境")}</h2>
          <span
            className={`status-pill ${environment?.ready ? "connected" : ""}`}
          >
            {t(
              environmentBusy || !environment
                ? "检测中…"
                : environment.ready
                  ? "可以处理媒体"
                  : "需要补全",
            )}
          </span>
        </div>
        <p className="settings-description">
          {t(
            "使用环境变量或已配置的 FFmpeg 和 ffprobe；缺少时可一键下载或选择本机文件补全。",
          )}
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
            <p>{t(environment.message)}</p>
          </div>
        )}
        <div className="settings-actions">
          <button
            className="button primary"
            disabled={!configurable || environmentBusy}
            onClick={() => void perform(chooseFFmpeg, "")}
          >
            <Icon name="folder" />
            {t("选择 FFmpeg…")}
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
            {t(environmentBusy ? "正在检测…" : "重新检测")}
          </button>
        </div>
        <FFmpegDownload
          status={ffmpegSetup}
          environment={environment}
          enabled={configurable && platform === "windows"}
          busy={environmentBusy}
          start={startFFmpegSetup}
          startCustom={startCustomFFmpegSetup}
          cancel={cancelFFmpegSetup}
        />
        <details className="manual-environment">
          <summary>{t("手动填写路径")}</summary>
          <div className="manual-fields">
            <label className="field-label">
              {t("FFmpeg 可执行文件")}
              <input
                value={ffmpegPath ?? environment?.ffmpeg.path ?? ""}
                onChange={(event) => setFFmpegPath(event.target.value)}
                placeholder={t("C:\\工具\\ffmpeg\\bin\\ffmpeg.exe")}
                spellCheck={false}
                disabled={!configurable || environmentBusy}
              />
            </label>
            <label className="field-label">
              {t("ffprobe 可执行文件")}
              <input
                value={ffprobePath ?? environment?.ffprobe.path ?? ""}
                onChange={(event) => setFFprobePath(event.target.value)}
                placeholder={t("C:\\工具\\ffmpeg\\bin\\ffprobe.exe")}
                spellCheck={false}
                disabled={!configurable || environmentBusy}
              />
            </label>
            <p className="field-hint">
              {t(
                "仅保存到 菲比工具箱 本机配置。环境变量仍具有更高优先级。留空 ffprobe 时查找 FFmpeg 同目录下的 ffprobe。",
              )}
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
              {t("验证并保存")}
            </button>
          </div>
        </details>
        <p className="field-hint discovery-hint">
          {t(
            "查找顺序：PHEEBKIT_FFMPEG / PHEEBKIT_FFPROBE → FFMPEG_PATH / FFPROBE_PATH → 本机配置 → 内置运行时 → PATH。",
          )}
        </p>
        {actionError && (
          <p className="inline-error" role="alert">
            {t(actionError)}
          </p>
        )}
        {actionStatus && (
          <p className="settings-status" role="status">
            {t(actionStatus)}
          </p>
        )}
      </section>
      <section className="surface preferences-card">
        <div className="section-title">
          <h2>{t("GPU 与编码器")}</h2>
          <span className="subtle-label">{t("自动模式优先 GPU")}</span>
        </div>
        <p className="settings-description">
          {t(
            "GPU 编码器通过实际编码测试后才会启用。自动模式会在 GPU 不可用时选用 CPU，压缩页显示最终使用方式。",
          )}
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
                      <span>{t(gpu.vendor)}</span>
                    </div>
                  </div>
                ))
              ) : (
                <p className="field-hint">
                  {t("没有检测到 GPU 设备信息；编码器可用性仍以实际测试为准。")}
                </p>
              )}
            </div>
            <details className="encoder-disclosure">
              <summary>
                {t("编码器实测结果 ·")}
                {t(" ")}
                {
                  capabilities.videoEncoders.filter(
                    (encoder) => encoder.kind === "gpu" && encoder.available,
                  ).length
                }
                {t(" ")}
                {t("个 GPU 编码器可用")}
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
                        <strong>{t(encoder.label)}</strong>
                        <code>{encoder.name}</code>
                      </div>
                      <span
                        className={`status-pill ${encoder.available ? "connected" : ""}`}
                      >
                        {t(encoder.available ? "可用" : "不可用")}
                      </span>
                      <p>
                        {t(
                          encoder.reason ||
                            (encoder.available
                              ? "实际编码测试通过"
                              : "当前设备或驱动不支持"),
                        )}
                      </p>
                    </div>
                  ))}
              </div>
            </details>
            <details className="encoder-disclosure">
              <summary>
                {t("全部 FFmpeg 编码器 ·")} {capabilities.encoders.length}
              </summary>
              <div className="encoder-chips">
                {capabilities.encoders.map((encoder) => (
                  <span key={encoder} className="chip">
                    {t(encoder)}
                  </span>
                ))}
              </div>
              <pre className="version-text">
                {capabilities.version || t("未返回版本信息")}
              </pre>
            </details>
          </>
        ) : (
          <div className="notice">
            <Icon name="info" />
            <p>{t(capabilityError || "正在检测设备和可用编码器…")}</p>
          </div>
        )}
      </section>
      {platform === "android" && (
        <div className="notice">
          <Icon name="info" />
          <p>{t("Android 媒体处理仍在开发中，当前版本暂不可执行压缩任务。")}</p>
        </div>
      )}
      <DiagnosticsPanel connected={connected} />
      <FFmpegExplorer
        key={`${environment?.ffmpeg.path}:${environment?.ffmpeg.version}`}
        connected={connected}
      />
      <p className="field-hint">
        {t("菲比工具箱 · PheebKit")} {t(__PHEEBKIT_VERSION__)} · Windows{t(" ")}
        {t(__PHEEBKIT_VERSION__.includes("-") ? "候选版" : "正式版")}{" "}
        {t("· Android 暂缓")}
      </p>
    </div>
  );
}
