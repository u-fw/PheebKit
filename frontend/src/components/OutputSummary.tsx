import { t } from "../i18n";
import type { ExecutionPlan, Media, Settings } from "../bridge";
import { formatBytes } from "../format";
import { codecOptions } from "./EncodingSettings";
import { Icon } from "./Icon";
import { formatTimecode, selectedDuration, trimError } from "../trim";
import { operationLabels, operationOf, selectedTracks } from "../project";
import { audioProcessingSummary } from "../audioProcessing";

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
  const operation = operationOf(settings);
  const compress = operation === "compress";
  const subtitleCount = selectedTracks(media, "subtitle", settings).length;
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
    compress && settings.mode === "size"
      ? formatBytes(settings.targetSizeMB * 1_000_000)
      : compress && settings.mode === "bitrate"
        ? formatBytes(
            ((settings.bitrateKbps * 1000) / 8) *
              selectedDuration(settings, media.durationSeconds),
          )
        : null;
  return (
    <section className="surface estimate-card">
      <div className="section-eyebrow">{t("输出概览")}</div>
      {settings.audioProcessing && (
        <p>
          {t("声音处理：")} {t(audioProcessingSummary(settings))}
        </p>
      )}
      <h2>
        {compress
          ? codecOptions.find((codec) => codec.value === settings.codec)?.title
          : t(operationLabels[operation])}
      </h2>
      <div className="size-comparison">
        <div>
          <span>{t("原始大小")}</span>
          <strong>{t(formatBytes(media.sizeBytes))}</strong>
        </div>
        <Icon name="arrow" />
        <div>
          <span>
            {t(
              !compress
                ? "输出大小"
                : settings.mode === "bitrate"
                  ? "视频流估算"
                  : settings.mode === "size"
                    ? `${settings.targetSizeMB} MB 目标`
                    : "输出大小",
            )}
          </span>
          <strong>
            {t(
              estimate
                ? `≈ ${estimate}`
                : compress
                  ? "按内容决定"
                  : "完成后显示",
            )}
          </strong>
        </div>
      </div>
      {settings.recovery && (
        <p>
          {t("恢复：")}{" "}
          {t(
            settings.recovery.mode === "delete"
              ? "删除失败区间"
              : "黑画面与静音填补",
          )}
          {t(" ")}
          {t("· 预计")}
          {t(" ")}
          {t(
            formatTimecode(
              plan?.durationSeconds ??
                selectedDuration(settings, media.durationSeconds),
            ),
          )}{" "}
          {t("。完整解码复检通过后保存。")}
        </p>
      )}
      <p>
        {t(
          !compress
            ? operation === "remux"
              ? "保留原始编码，转换容器。实际大小取决于保留的轨道和容器开销。"
              : "仅保存所选轨道，完成后显示实际输出大小。"
            : estimate
              ? "估算供参考，以完成后的实际大小为准。"
              : "质量模式无法准确预测体积。处理完成后会显示实际节省空间。",
        )}
      </p>
      <div className="summary-divider" />
      <div className="output-summary">
        <span>{t("处理范围")}</span>
        <strong>
          {t(
            settings.segments?.length
              ? `${settings.segments.length} 段拼接`
              : settings.trim
                ? "所选片段"
                : "完整媒体",
          )}
        </strong>
        <span>{t("输出时长")}</span>
        <strong>
          {t(
            rangeInvalid
              ? "待修正范围"
              : formatTimecode(
                  selectedDuration(settings, media.durationSeconds),
                ) || "未知",
          )}
        </strong>
        {settings.trim && (
          <>
            <span>{t("起止时间")}</span>
            <strong>
              {t(formatTimecode(settings.trim.startSeconds) || "—")} →{t(" ")}
              {t(formatTimecode(settings.trim.endSeconds) || "—")}
            </strong>
          </>
        )}
        {compress && (
          <>
            <span>{t("分辨率")}</span>
            <strong>
              {t(settings.resolution ? `${settings.resolution}p` : "原始")}
            </strong>
          </>
        )}
        {operation !== "extract-subtitle" && (
          <>
            <span>{t("音频")}</span>
            <strong>
              {t(
                selectedTracks(media, "audio", settings).length
                  ? operation === "remux"
                    ? "保留原音频"
                    : settings.audio === "copy"
                      ? "保留原音频"
                      : settings.audio.toUpperCase()
                  : "不保留",
              )}
            </strong>
          </>
        )}
        {operation !== "extract-audio" && (
          <>
            <span>{t("字幕")}</span>
            <strong>
              {t(
                settings.subtitleBurn
                  ? `烧录到画面${subtitleCount ? t(` + ${subtitleCount} 条软字幕`) : ""}`
                  : subtitleCount
                    ? `${subtitleCount} 条软字幕`
                    : "不保留",
              )}
            </strong>
          </>
        )}
        <span>{t("处理方式")}</span>
        <strong>
          {t(
            !compress
              ? operation === "remux"
                ? "直接复制轨道"
                : "本地处理"
              : plan
                ? `${plan.encoderKind === "gpu" ? "GPU" : "CPU"} · ${plan.encoderLabel}`
                : settings.encoder === "auto"
                  ? "自动 · 优先 GPU"
                  : settings.encoder === "cpu"
                    ? "CPU"
                    : settings.encoder,
          )}
        </strong>
      </div>
      {compress && !hasVideo && (
        <p className="inline-error">
          {t("源媒体没有视频轨，请选择视频文件。")}
        </p>
      )}
      {compress && capabilitiesLoaded && !encoderAvailable && (
        <p className="inline-error">
          {t("所选编码方式在当前设备上不可用，请更换处理方式或视频编码。")}
        </p>
      )}
      {validation.length > 0 && (
        <div className="validation-errors" role="alert">
          {validation.map((message) => (
            <p key={message}>{t(message)}</p>
          ))}
        </div>
      )}
      {actionError && (
        <p className="inline-error" role="alert">
          {t(actionError)}
        </p>
      )}
      {previewError && (
        <p className="inline-error" role="alert">
          {t("执行方案无法生成：")} {t(previewError)}
        </p>
      )}
      {!connected && (
        <p className="inline-error">
          {t("原生媒体处理服务未连接，请在 菲比工具箱 应用中操作。")}
        </p>
      )}
      {(previewLoading || (compress && !capabilitiesLoaded)) &&
        !validation.length &&
        !previewError &&
        connected && (
          <p className="plan-status" role="status">
            {t(
              compress && !capabilitiesLoaded
                ? "正在检测设备与编码器…"
                : "正在检查输出设置…",
            )}
          </p>
        )}
      <button
        className="button primary start-button"
        onClick={start}
        disabled={!canStart}
      >
        {t(
          starting
            ? "正在创建任务…"
            : previewLoading
              ? "正在检查设置…"
              : !compress
                ? `开始${t(operationLabels[operation])}`
                : settings.recovery
                  ? "开始恢复并压缩"
                  : settings.trim
                    ? "开始剪辑并压缩"
                    : "开始压缩",
        )}
        <Icon name="arrow" />
      </button>
      <span className="field-hint">
        {t("任务进入队列后可查看进度和取消。")}
      </span>
    </section>
  );
}
