import { t } from "../i18n";
import type { Capabilities, EnvironmentStatus, Media } from "../bridge";
import { summarizeProcessingReadiness } from "../readiness";
import { Icon } from "./Icon";
import { MediaCard } from "./MediaCard";
import { ProcessingReadiness } from "./ProcessingReadiness";

export function Home({
  media,
  busy,
  connected,
  environment,
  capabilities,
  capabilityError,
  chooseMedia,
  configureEnvironment,
  openProject,
  compress,
}: {
  media: Media | null;
  busy: boolean;
  connected: boolean;
  environment: EnvironmentStatus | null;
  capabilities: Capabilities | null;
  capabilityError: string;
  chooseMedia: () => void;
  configureEnvironment: () => void;
  openProject: () => void;
  compress: () => void;
}) {
  const readiness = summarizeProcessingReadiness({
    connected,
    environment,
    capabilities,
    capabilityError,
  });
  const checkingEnvironment = connected && environment === null;
  const needsEnvironment =
    connected && environment !== null && !environment.ready;
  const chooseDisabled =
    busy || !connected || (!needsEnvironment && !readiness.canChooseMedia);

  return (
    <div className="page home-page">
      <header className="page-heading">
        <div>
          <h1>{t("工作台")}</h1>
          <p>
            {t(
              media
                ? "查看源文件信息，开始处理媒体。"
                : "导入视频或音频，查看媒体信息。",
            )}
          </p>
        </div>
        <div className="action-row page-heading-actions">
          {media && (
            <button
              className="button outlined"
              disabled={busy || !connected}
              onClick={chooseMedia}
            >
              <Icon name="plus" />
              {t("重新选择媒体")}
            </button>
          )}
          <button
            className="button outlined"
            disabled={busy || !connected}
            onClick={openProject}
          >
            <Icon name="folder" />
            {t("打开项目")}
          </button>
        </div>
      </header>
      {media ? (
        <>
          <MediaCard media={media} />
          <div className="ready-row">
            <div>
              <h3>{t("准备好处理了")}</h3>
              <p>{t("压缩、转换封装或提取轨道，查看实际执行参数。")}</p>
            </div>
            <button className="button primary" onClick={compress}>
              {t("处理媒体")}
              <Icon name="arrow" />
            </button>
          </div>
        </>
      ) : (
        <section
          className={`drop-zone ${busy ? "is-busy" : ""}`}
          aria-label={t("导入媒体")}
          data-file-drop-target
        >
          <div className="drop-illustration" aria-hidden="true">
            <span className="illustration-orbit orbit-one" />
            <span className="illustration-orbit orbit-two" />
            <div className="illustration-tile tile-back" />
            <div className="illustration-tile tile-front">
              <Icon name="video" />
              <span />
              <span />
            </div>
            <div className="illustration-plus">
              <Icon name="plus" />
            </div>
          </div>
          <div className="drop-content">
            <div className="section-eyebrow">{t("导入媒体")}</div>
            <h2>{t(busy ? "正在分析媒体…" : "选择文件，开始处理")}</h2>
            <p>
              {t("将视频或音频整批拖入窗口，或从设备中多选。")}
              <br />
              {t("媒体始终在本地处理。")}
            </p>
            <ProcessingReadiness readiness={readiness} />
            <button
              className="button primary"
              onClick={needsEnvironment ? configureEnvironment : chooseMedia}
              disabled={chooseDisabled}
            >
              <Icon
                name={
                  busy
                    ? "video"
                    : needsEnvironment
                      ? "settings"
                      : checkingEnvironment
                        ? "refresh"
                        : "plus"
                }
              />
              {t(
                busy
                  ? "正在读取媒体信息"
                  : needsEnvironment
                    ? "补全 FFmpeg"
                    : checkingEnvironment
                      ? "正在检查处理环境"
                      : "选择媒体文件",
              )}
            </button>
            <span className="drop-hint">
              {t("MP4 · MKV · MOV · WEBM · 音频文件")}
            </span>
            {!connected && (
              <span className="preview-note">
                {t("浏览器可预览界面。媒体操作需要 Wails 原生桥接。")}
              </span>
            )}
          </div>
        </section>
      )}
      {!media && (
        <details className="workflow-section" aria-label={t("处理流程")}>
          <summary className="section-title">
            <span>{t("处理流程")}</span>
            <span className="subtle-label">{t("简单的三步")}</span>
          </summary>
          <div className="workflow-grid">
            <article className="workflow-card">
              <span className="step-number">01</span>
              <Icon name="video" />
              <h3>{t("认识你的媒体")}</h3>
              <p>
                {t("分辨率、色彩和每一条轨道，")}
                <br />
                {t("由 ffprobe 精准分析。")}
              </p>
            </article>
            <article className="workflow-card">
              <span className="step-number">02</span>
              <Icon name="compress" />
              <h3>{t("找到合适的平衡")}</h3>
              <p>
                {t("H.264、HEVC 或 AV1。")}
                <br />
                {t("在质量、码率和体积之间选择。")}
              </p>
            </article>
            <article className="workflow-card">
              <span className="step-number">03</span>
              <Icon name="queue" />
              <h3>{t("看得见的处理过程")}</h3>
              <p>
                {t("实时进度，透明的执行参数。")}
                <br />
                {t("随时取消，由你掌控。")}
              </p>
            </article>
          </div>
        </details>
      )}
      <div className="local-note">
        <Icon name="info" />
        <span>{t("本地处理 · 离线可用 · 文件由你掌控")}</span>
      </div>
    </div>
  );
}
