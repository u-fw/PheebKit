import type { Media, Stream } from "../bridge";
import { formatBitrate, formatBytes, formatDuration } from "../format";
import { useThumbnail } from "../hooks/useThumbnail";
import { Icon } from "./Icon";

function streamLabel(stream: Stream): string {
  return [stream.codecName.toUpperCase(), stream.language, stream.title]
    .filter(Boolean)
    .join(" · ");
}

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
      aria-label="媒体信息"
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
          {format}
          {cover.loading ? " · 读取封面…" : cover.failed ? " · 无可用封面" : ""}
        </span>
      </div>
      <div className="media-details">
        <div className="section-eyebrow">
          源媒体 <span className="tiny-dot" /> 已分析
        </div>
        <h2 title={media.name}>{media.name}</h2>
        <div className="media-tags">
          {video && (
            <span className="chip">
              {video.width} × {video.height}
            </span>
          )}
          {video && (
            <span className="chip">
              {video.codecName.toUpperCase()}
              {video.bitDepth ? ` · ${video.bitDepth}-bit` : ""}
            </span>
          )}
          {video && (
            <span className="chip" title={`${video.frameRate} FPS`}>
              {video.frameRate > 0
                ? new Intl.NumberFormat("zh-CN", {
                    maximumFractionDigits: 2,
                  }).format(video.frameRate)
                : "未知"}{" "}
              FPS
            </span>
          )}
          {video?.hdr && <span className="chip accent-chip">{video.hdr}</span>}
        </div>
        <dl className="media-metrics">
          <div>
            <dt>文件大小</dt>
            <dd>{formatBytes(media.sizeBytes)}</dd>
          </div>
          <div>
            <dt>时长</dt>
            <dd>{formatDuration(media.durationSeconds)}</dd>
          </div>
          <div>
            <dt>平均码率</dt>
            <dd>{formatBitrate(media.bitrate)}</dd>
          </div>
        </dl>
        {!compact && (
          <div className="stream-list">
            <div>
              <Icon name="video" />
              <span>视频 · {videos.length} 轨</span>
              <span>
                {video
                  ? [video.pixelFormat, video.colorSpace, video.colorPrimaries]
                      .filter(Boolean)
                      .join(" · ") || "无色彩信息"
                  : "无视频轨"}
              </span>
            </div>
            <div>
              <Icon name="audio" />
              <span>音频 · {audio.length} 轨</span>
              <span>
                {audio.length ? audio.map(streamLabel).join(" / ") : "无音频轨"}
              </span>
            </div>
            <div>
              <Icon name="subtitle" />
              <span>字幕 · {subtitles.length} 轨</span>
              <span>
                {subtitles.length
                  ? subtitles.map(streamLabel).join(" / ")
                  : "无字幕轨"}
              </span>
            </div>
          </div>
        )}
      </div>
    </article>
  );
}
