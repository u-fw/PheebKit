import type { Media, Settings } from "../bridge";
import { formatBytes } from "../format";
import { codecOptions } from "./EncodingSettings";
import { Icon } from "./Icon";

export function OutputSummary({
  media,
  settings,
  hasVideo,
  encoderAvailable,
  capabilitiesLoaded,
  validation,
  actionError,
  starting,
  canStart,
  start,
}: {
  media: Media;
  settings: Settings;
  hasVideo: boolean;
  encoderAvailable: boolean;
  capabilitiesLoaded: boolean;
  validation: string[];
  actionError: string;
  starting: boolean;
  canStart: boolean;
  start: () => void;
}) {
  const estimate =
    settings.mode === "size"
      ? formatBytes(settings.targetSizeMB * 1_000_000)
      : settings.mode === "bitrate"
        ? formatBytes(
            ((settings.bitrateKbps * 1000) / 8) * media.durationSeconds,
          )
        : null;
  return (
    <section className="surface estimate-card">
      <div className="section-eyebrow">OUTPUT AT A GLANCE</div>
      <h2>
        {codecOptions.find((codec) => codec.value === settings.codec)?.title}
      </h2>
      <div className="size-comparison">
        <div>
          <span>原始大小</span>
          <strong>{formatBytes(media.sizeBytes)}</strong>
        </div>
        <Icon name="arrow" />
        <div>
          <span>{settings.mode === "bitrate" ? "视频流估算" : "目标体积"}</span>
          <strong>{estimate ? `≈ ${estimate}` : "按内容决定"}</strong>
        </div>
      </div>
      <p>
        {estimate
          ? "估算供参考，以完成后的实际大小为准。"
          : "质量模式无法准确预测体积。处理完成后会显示实际节省空间。"}
      </p>
      <div className="summary-divider" />
      <div className="output-summary">
        <span>分辨率</span>
        <strong>
          {settings.resolution ? `${settings.resolution}p` : "原始"}
        </strong>
        <span>音频</span>
        <strong>{settings.audio.toUpperCase()}</strong>
        <span>处理方式</span>
        <strong>本地 CPU</strong>
      </div>
      {!hasVideo && (
        <p className="inline-error">
          源媒体没有视频轨。此 Spike 压缩流程需要视频文件。
        </p>
      )}
      {capabilitiesLoaded && !encoderAvailable && (
        <p className="inline-error">当前 FFmpeg 构建缺少所选视频编码器。</p>
      )}
      {validation.length > 0 && (
        <div className="validation-errors" role="alert">
          {validation.map((message) => (
            <p key={message}>{message}</p>
          ))}
        </div>
      )}
      {actionError && (
        <p className="inline-error" role="alert">
          {actionError}
        </p>
      )}
      <button
        className="button primary start-button"
        onClick={start}
        disabled={!canStart}
      >
        {starting ? "正在创建任务…" : "开始压缩"}
        <Icon name="arrow" />
      </button>
      <span className="field-hint">任务进入队列后可查看进度和取消。</span>
    </section>
  );
}
