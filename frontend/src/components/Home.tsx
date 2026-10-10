import type { Media } from "../bridge";
import { Icon } from "./Icon";
import { MediaCard } from "./MediaCard";

export function Home({
  media,
  busy,
  connected,
  chooseMedia,
  compress,
}: {
  media: Media | null;
  busy: boolean;
  connected: boolean;
  chooseMedia: () => void;
  compress: () => void;
}) {
  return (
    <div className="page home-page">
      <header className="page-heading">
        <div>
          <h1>工作台</h1>
          <p>
            {media
              ? "查看源文件信息，开始处理媒体。"
              : "导入视频或音频，查看媒体信息。"}
          </p>
        </div>
        {media && (
          <button
            className="button outlined"
            disabled={busy || !connected}
            onClick={chooseMedia}
          >
            <Icon name="plus" />
            更换媒体
          </button>
        )}
      </header>
      {media ? (
        <>
          <MediaCard media={media} />
          <div className="ready-row">
            <div>
              <h3>准备好处理了</h3>
              <p>选择编码、调整质量，查看实际执行参数。</p>
            </div>
            <button className="button primary" onClick={compress}>
              压缩视频
              <Icon name="arrow" />
            </button>
          </div>
        </>
      ) : (
        <section
          className={`drop-zone ${busy ? "is-busy" : ""}`}
          aria-label="导入媒体"
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
            <div className="section-eyebrow">导入媒体</div>
            <h2>{busy ? "正在分析媒体…" : "选择文件，开始处理"}</h2>
            <p>
              将视频或音频拖入窗口，或从设备中选择。
              <br />
              媒体始终在本地处理。
            </p>
            <button
              className="button primary"
              onClick={chooseMedia}
              disabled={busy || !connected}
            >
              <Icon name={busy ? "video" : "plus"} />
              {busy ? "正在读取媒体信息" : "选择媒体文件"}
            </button>
            <span className="drop-hint">MP4 · MKV · MOV · WEBM · 音频文件</span>
            {!connected && (
              <span className="preview-note">
                浏览器可预览界面。媒体操作需要 Wails 原生桥接。
              </span>
            )}
          </div>
        </section>
      )}
      {!media && (
        <section className="workflow-section" aria-label="处理流程">
          <div className="section-title">
            <h2>处理流程</h2>
            <span className="subtle-label">简单的三步</span>
          </div>
          <div className="workflow-grid">
            <article className="workflow-card">
              <span className="step-number">01</span>
              <Icon name="video" />
              <h3>认识你的媒体</h3>
              <p>
                分辨率、色彩和每一条轨道，
                <br />由 ffprobe 精准分析。
              </p>
            </article>
            <article className="workflow-card">
              <span className="step-number">02</span>
              <Icon name="compress" />
              <h3>找到合适的平衡</h3>
              <p>
                H.264、HEVC 或 AV1。
                <br />
                在质量、码率和体积之间选择。
              </p>
            </article>
            <article className="workflow-card">
              <span className="step-number">03</span>
              <Icon name="queue" />
              <h3>看得见的处理过程</h3>
              <p>
                实时进度，透明的执行参数。
                <br />
                随时取消，由你掌控。
              </p>
            </article>
          </div>
        </section>
      )}
      <div className="local-note">
        <Icon name="info" />
        <span>本地处理 · 离线可用 · 文件由你掌控</span>
      </div>
    </div>
  );
}
