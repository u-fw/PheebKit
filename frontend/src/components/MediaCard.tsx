import { t, getLanguage } from "../i18n";
import type { Media } from "../bridge";
import { formatBitrate, formatBytes, formatDuration } from "../format";
import { useThumbnail } from "../hooks/useThumbnail";
import { Icon } from "./Icon";
import { streamLabel, videoLabel } from "../mediaLabels";
import { videoDisplaySize } from "../project";

export function MediaCard({
  media,
  compact = false,
}: {
  media: Media;
  compact?: boolean;
}) {
  const videos = media.streams.filter(
    (stream) =>
      stream.codecType === "video" &&
      !stream.attachedPicture &&
      !stream.timedThumbnails,
  );
  const audio = media.streams.filter((stream) => stream.codecType === "audio");
  const subtitles = media.streams.filter(
    (stream) => stream.codecType === "subtitle",
  );
  const video = videos[0];
  const cover = useThumbnail(media);
  const extension = media.name.match(/\.([a-z0-9]{1,8})$/i)?.[1];
  const format =
    extension?.toUpperCase() ||
    media.container.split(",")[0]?.toUpperCase() ||
    "MEDIA";
  return (
    <article
      className={`media-card surface ${compact ? "compact" : ""}`}
      aria-label={t("媒体信息")}
    >
      <div
        className={`media-visual ${cover.thumbnail ? "has-thumbnail" : ""}`}
        aria-hidden="true"
      >
        {cover.thumbnail ? (
          <img
            className="media-thumbnail"
            src={cover.thumbnail.dataUrl}
            alt=""
            width={cover.thumbnail.width}
            height={cover.thumbnail.height}
          />
        ) : (
          <div className="film-frame">
            <Icon name={video ? "video" : "audio"} />
          </div>
        )}
        <span>
          {t(format)}{" "}
          {t(
            cover.loading
              ? " · 读取封面…"
              : cover.failed
                ? " · 无可用封面"
                : "",
          )}
        </span>
      </div>
      <div className="media-details">
        <div className="section-eyebrow">
          {t("源媒体")}
          <span className="tiny-dot" /> {t("已分析")}
        </div>
        <h2 title={media.name}>{media.name}</h2>
        <div className="media-tags">
          {video && (
            <span className="chip">
              {videoDisplaySize(video).width} × {videoDisplaySize(video).height}
            </span>
          )}
          {video && (
            <span className="chip">
              {video.codecName.toUpperCase()}{" "}
              {t(video.bitDepth ? ` · ${video.bitDepth}-bit` : "")}
            </span>
          )}
          {video && (
            <span className="chip" title={t(`${video.frameRate} FPS`)}>
              {t(
                video.frameRate > 0
                  ? new Intl.NumberFormat(getLanguage(), {
                      maximumFractionDigits: 2,
                    }).format(video.frameRate)
                  : "未知",
              )}
              {t(" ")}
              FPS
            </span>
          )}
          {video?.hdr && (
            <span className="chip accent-chip">{t(video.hdr)}</span>
          )}
        </div>
        <dl className="media-metrics">
          <div>
            <dt>{t("文件大小")}</dt>
            <dd>{t(formatBytes(media.sizeBytes))}</dd>
          </div>
          <div>
            <dt>{t("时长")}</dt>
            <dd>{t(formatDuration(media.durationSeconds))}</dd>
          </div>
          <div>
            <dt>{t("平均码率")}</dt>
            <dd>{t(formatBitrate(media.bitrate))}</dd>
          </div>
        </dl>
        {!compact && (
          <div className="stream-list">
            <div>
              <Icon name="video" />
              <span>
                {t("视频 ·")} {videos.length} {t("轨")}
              </span>
              <span
                title={
                  video
                    ? t(
                        "像素格式：{{0}}；色彩矩阵：{{1}}；传递特性：{{2}}；原色：{{3}}",
                        [
                          video.pixelFormat || t("未标注"),
                          video.colorSpace || t("未标注"),
                          video.colorTransfer || t("未标注"),
                          video.colorPrimaries || t("未标注"),
                        ],
                      )
                    : undefined
                }
              >
                {t(video ? videoLabel(video) : "无视频轨")}
              </span>
            </div>
            <div>
              <Icon name="audio" />
              <span>
                {t("音频 ·")} {audio.length} {t("轨")}
              </span>
              <span>
                {t(
                  audio.length
                    ? audio.map(streamLabel).join(" / ")
                    : "无音频轨",
                )}
              </span>
            </div>
            <div>
              <Icon name="subtitle" />
              <span>
                {t("字幕 ·")} {subtitles.length} {t("轨")}
              </span>
              <span>
                {t(
                  subtitles.length
                    ? subtitles.map(streamLabel).join(" / ")
                    : "未检测到内嵌字幕",
                )}
              </span>
            </div>
          </div>
        )}
      </div>
    </article>
  );
}
