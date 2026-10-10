import type { ExecutionPlan, Media, Settings } from "../bridge";
import { formatBytes } from "../format";
import { codecOptions } from "./EncodingSettings";
import { Icon } from "./Icon";
import { formatTimecode, selectedDuration, trimError } from "../trim";

export function OutputSummary({
  media,
  settings,
  hasVideo,
  encoderAvailable,
  capabilitiesLoaded,
  validation,
  actionError,
  previewError,
  previewLoading,
  plan,
  connected,
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
  previewError: string;
  previewLoading: boolean;
  plan: ExecutionPlan | null;
  connected: boolean;
  starting: boolean;
  canStart: boolean;
  start: () => void;
}) {
  const video = media.streams.find(
    (stream) =>
      stream.codecType === "video" &&
      !stream.attachedPicture &&
      !stream.timedThumbnails,
  );
  const rangeInvalid = Boolean(
    trimError(settings, media.durationSeconds, video?.frameRate),
  );
  const estimate =
    settings.mode === "size"
      ? formatBytes(settings.targetSizeMB * 1_000_000)
      : settings.mode === "bitrate"
        ? formatBytes(
            ((settings.bitrateKbps * 1000) / 8) *
              selectedDuration(settings, media.durationSeconds),
          )
        : null;
  return (
    <section className="surface estimate-card">
      <div className="section-eyebrow">输出概览</div>
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
          <span>
            {settings.mode === "bitrate"
              ? "视频流估算"
              : settings.mode === "size"
                ? `${settings.targetSizeMB} MB 目标`
                : "输出大小"}
          </span>
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
        <span>处理范围</span>
        <strong>{settings.trim ? "剪辑并压缩" : "完整视频"}</strong>
        <span>输出时长</span>
        <strong>
          {rangeInvalid
            ? "待修正范围"
            : formatTimecode(
                selectedDuration(settings, media.durationSeconds),
              ) || "未知"}
        </strong>
        {settings.trim && (
          <>
            <span>起止时间</span>
            <strong>
              {formatTimecode(settings.trim.startSeconds) || "—"} →{" "}
              {formatTimecode(settings.trim.endSeconds) || "—"}
            </strong>
          </>
        )}
        <span>分辨率</span>
        <strong>
          {settings.resolution ? `${settings.resolution}p` : "原始"}
        </strong>
        <span>音频</span>
        <strong>{settings.audio.toUpperCase()}</strong>
        <span>处理方式</span>
        <strong>
          {plan
            ? `${plan.encoderKind === "gpu" ? "GPU" : "CPU"} · ${plan.encoderLabel}`
            : settings.encoder === "auto"
              ? "自动 · 优先 GPU"
              : settings.encoder === "cpu"
                ? "CPU"
                : settings.encoder}
        </strong>
      </div>
      {!hasVideo && (
        <p className="inline-error">源媒体没有视频轨，请选择视频文件。</p>
      )}
      {capabilitiesLoaded && !encoderAvailable && (
        <p className="inline-error">
          所选编码方式在当前设备上不可用，请更换处理方式或视频编码。
        </p>
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
      {previewError && (
        <p className="inline-error" role="alert">
          执行方案无法生成：{previewError}
        </p>
      )}
      {!connected && (
        <p className="inline-error">
          原生媒体处理服务未连接，请在 kompeg 应用中操作。
        </p>
      )}
      {(previewLoading || !capabilitiesLoaded) &&
        !validation.length &&
        !previewError &&
        connected && (
          <p className="plan-status" role="status">
            {!capabilitiesLoaded
              ? "正在检测设备与编码器…"
              : "正在检查输出设置…"}
          </p>
        )}
      <button
        className="button primary start-button"
        onClick={start}
        disabled={!canStart}
      >
        {starting
          ? "正在创建任务…"
          : previewLoading
            ? "正在检查设置…"
            : settings.trim
              ? "开始剪辑并压缩"
              : "开始压缩"}
        <Icon name="arrow" />
      </button>
      <span className="field-hint">任务进入队列后可查看进度和取消。</span>
    </section>
  );
}
