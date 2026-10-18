import { t } from "../i18n";
import { useEffect, useRef, useState } from "react";
import { API } from "../bridge";
import type { Media, Thumbnail } from "../bridge";
import { errorMessage } from "../format";
import { formatTimecode } from "../trim";

function PositionFrame({
  uri,
  seconds,
  label,
  visible,
}: {
  uri: string;
  seconds: number;
  label: string;
  visible: boolean;
}) {
  const [result, setResult] = useState<{
    seconds: number;
    image?: Thumbnail;
    error?: string;
  } | null>(null);
  useEffect(() => {
    if (!visible) return;
    let alive = true;
    const timer = setTimeout(() => {
      void API.getVideoFrame(uri, seconds).then(
        (image) => {
          if (alive) setResult({ seconds, image });
        },
        (cause) => {
          if (alive) setResult({ seconds, error: errorMessage(cause) });
        },
      );
    }, 400);
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [uri, seconds, visible]);
  const current = result?.seconds === seconds ? result : null;
  return (
    <figure className="position-frame">
      <div className="frame-image">
        {current?.image ? (
          <img src={current.image.dataUrl} alt={t("{{0}}画面", [label])} />
        ) : (
          <span>{t(current?.error ? "画面读取失败" : "正在读取画面…")}</span>
        )}
      </div>
      <figcaption>
        {t(label)}
        <span>{t(formatTimecode(seconds))}</span>
      </figcaption>
      {current?.error && (
        <p className="field-hint" role="status">
          {t(current.error)}
        </p>
      )}
    </figure>
  );
}

export function ClipPreview({
  media,
  start,
  end,
  visible,
  onBoundary,
}: {
  media: Media;
  start: number;
  end: number;
  visible: boolean;
  onBoundary: (boundary: "startSeconds" | "endSeconds", value: number) => void;
}) {
  const video = useRef<HTMLVideoElement>(null);
  const stopAt = useRef<number | null>(null);
  const [source, setSource] = useState("");
  const [position, setPosition] = useState(start);
  const [playing, setPlaying] = useState(false);
  const [failed, setFailed] = useState("");
  const total = media.durationSeconds;
  useEffect(() => {
    let alive = true;
    void API.getMediaPreviewURL(media.source.uri).then(
      (url) => {
        if (alive) setSource(url);
      },
      (cause) => {
        if (alive) setFailed(errorMessage(cause));
      },
    );
    return () => {
      alive = false;
    };
  }, [media.source.uri]);
  useEffect(() => {
    if (!visible) video.current?.pause();
  }, [visible]);
  function seek(value: number) {
    const next = Math.min(total, Math.max(0, value));
    setPosition(next);
    if (video.current && Number.isFinite(video.current.duration))
      video.current.currentTime = next;
  }
  async function togglePlay() {
    stopAt.current = null;
    const player = video.current;
    if (!player) return;
    if (!player.paused) {
      player.pause();
      return;
    }
    try {
      await player.play();
    } catch (cause) {
      setFailed(`无法播放原视频：${errorMessage(cause)}`);
    }
  }
  async function playSelection() {
    const player = video.current;
    if (!player) return;
    seek(start);
    stopAt.current = end;
    try {
      await player.play();
    } catch (cause) {
      setFailed(`无法播放原视频：${errorMessage(cause)}`);
    }
  }
  return (
    <div className="clip-preview">
      <div className="section-title">
        <h3>{t("剪辑预览")}</h3>
        <span className="subtle-label">{t("原视频 · 不改变输出")}</span>
      </div>
      {source && !failed ? (
        <video
          ref={video}
          src={source}
          preload={visible ? "metadata" : "none"}
          playsInline
          aria-label={t("源视频剪辑预览")}
          onLoadedMetadata={() => seek(position)}
          onTimeUpdate={(event) => {
            const player = event.currentTarget;
            if (
              stopAt.current !== null &&
              player.currentTime >= stopAt.current
            ) {
              player.pause();
              player.currentTime = stopAt.current;
              stopAt.current = null;
            }
            setPosition(player.currentTime);
          }}
          onPlay={() => setPlaying(true)}
          onPause={() => setPlaying(false)}
          onEnded={() => setPlaying(false)}
          onError={() =>
            setFailed(
              "当前 WebView2 无法播放此视频编码，已改为 FFmpeg 位置画面预览。",
            )
          }
        />
      ) : (
        <PositionFrame
          uri={media.source.uri}
          seconds={position}
          label={t("当前位置")}
          visible={visible}
        />
      )}
      {failed && (
        <p className="field-hint" role="status">
          {t(failed)} {t("仍可拖动定位并设置起止时间。")}
        </p>
      )}
      <div className="action-row">
        <button
          className="button outlined"
          disabled={!source || Boolean(failed)}
          onClick={() => void togglePlay()}
        >
          {t(playing ? "暂停" : "播放")}
        </button>
        <button
          className="button outlined"
          disabled={!source || Boolean(failed) || start >= end}
          onClick={() => void playSelection()}
        >
          {t("播放所选片段")}
        </button>
        <strong>
          {t(formatTimecode(position))} / {t(formatTimecode(total))}
        </strong>
      </div>
      <label className="field-label" htmlFor="clip-position">
        {t("预览位置")}
      </label>
      <input
        id="clip-position"
        type="range"
        min="0"
        max={total}
        step="0.001"
        value={position}
        aria-valuetext={formatTimecode(position)}
        onChange={(event) => seek(Number(event.target.value))}
      />
      <div className="action-row">
        <button
          className="button outlined"
          disabled={position >= end}
          onClick={() =>
            onBoundary("startSeconds", Math.floor(position * 1000) / 1000)
          }
        >
          {t("当前位置设为起点")}
        </button>
        <button
          className="button outlined"
          disabled={position <= start}
          onClick={() =>
            onBoundary("endSeconds", Math.floor(position * 1000) / 1000)
          }
        >
          {t("当前位置设为终点")}
        </button>
        <button className="button text" onClick={() => seek(start)}>
          {t("定位起点")}
        </button>
        <button className="button text" onClick={() => seek(end)}>
          {t("定位终点")}
        </button>
      </div>
      <div className="boundary-frames">
        <PositionFrame
          uri={media.source.uri}
          seconds={Math.max(0, Math.min(total, start))}
          label={t("开始位置")}
          visible={visible}
        />
        <PositionFrame
          uri={media.source.uri}
          seconds={Math.max(0, Math.min(total, end))}
          label={t("结束边界附近")}
          visible={visible}
        />
      </div>
      <p className="field-hint">
        {t(
          "结束时间为不包含的边界；下方显示结束边界附近画面。预览用于定位，实际切点受帧率影响。",
        )}
      </p>
    </div>
  );
}
