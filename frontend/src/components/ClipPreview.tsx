import { t } from "../i18n";
import { useEffect, useId, useRef, useState } from "react";
import { API } from "../bridge";
import type { Media, Thumbnail } from "../bridge";
import { errorMessage } from "../format";
import { formatTimecode } from "../trim";
import {
  boundedPosition,
  framePosition,
  selectionPlayback,
} from "../previewTimeline";
import { Filmstrip } from "./Filmstrip";
import { WaveformOverview } from "./WaveformOverview";

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
    uri: string;
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
          if (alive) setResult({ uri, seconds, image });
        },
        (cause) => {
          if (alive) setResult({ uri, seconds, error: errorMessage(cause) });
        },
      );
    }, 400);
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [uri, seconds, visible]);
  const current =
    result?.uri === uri && result.seconds === seconds ? result : null;
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
  showFrames = true,
}: {
  media: Media;
  start: number;
  end: number;
  visible: boolean;
  onBoundary?: (boundary: "startSeconds" | "endSeconds", value: number) => void;
  showFrames?: boolean;
}) {
  const positionID = useId();
  const video = useRef<HTMLVideoElement>(null);
  const lifetime = useRef({ active: false });
  const stopAt = useRef<number | null>(null);
  const [source, setSource] = useState("");
  const [position, setPosition] = useState(start);
  const [playing, setPlaying] = useState(false);
  const [loop, setLoop] = useState(false);
  const [failed, setFailed] = useState("");
  const total = media.durationSeconds;
  const frameRate =
    media.streams.find(
      (stream) =>
        stream.codecType === "video" &&
        !stream.attachedPicture &&
        !stream.timedThumbnails,
    )?.frameRate ?? 0;
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
    const state = lifetime.current;
    const player = video.current;
    state.active = visible;
    if (!visible) player?.pause();
    return () => {
      state.active = false;
      player?.pause();
    };
  }, [visible, source]);
  useEffect(() => {
    if (!visible || !playing) return;
    let tick = 0;
    function inspect() {
      const player = video.current;
      if (player && stopAt.current !== null) {
        const decision = selectionPlayback(
          player.currentTime,
          start,
          end,
          loop,
        );
        if (decision) {
          player.currentTime = decision.seek;
          if (decision.pause) {
            player.pause();
            stopAt.current = null;
          }
        }
      }
      tick = requestAnimationFrame(inspect);
    }
    tick = requestAnimationFrame(inspect);
    return () => cancelAnimationFrame(tick);
  }, [visible, playing, start, end, loop]);
  function seek(value: number) {
    const next = boundedPosition(value, total);
    setPosition(next);
    if (video.current && Number.isFinite(video.current.duration))
      video.current.currentTime = next;
  }
  function step(direction: -1 | 1) {
    video.current?.pause();
    stopAt.current = null;
    seek(
      framePosition(
        video.current?.currentTime ?? position,
        direction,
        frameRate,
        total,
      ),
    );
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
      if (!lifetime.current.active) player.pause();
    } catch (cause) {
      if (lifetime.current.active)
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
      if (!lifetime.current.active) player.pause();
    } catch (cause) {
      if (lifetime.current.active)
        setFailed(`无法播放原视频：${errorMessage(cause)}`);
    }
  }
  return (
    <div className="clip-preview">
      <div className="section-title">
        <h3>{t(onBoundary ? "剪辑预览" : "源媒体预览")}</h3>
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
            setPosition(player.currentTime);
          }}
          onPlay={(event) => {
            if (!lifetime.current.active) event.currentTarget.pause();
            else setPlaying(true);
          }}
          onPause={() => setPlaying(false)}
          onEnded={() => {
            if (
              lifetime.current.active &&
              loop &&
              stopAt.current !== null &&
              start < end
            ) {
              seek(start);
              const player = video.current;
              void player?.play().then(
                () => {
                  if (!lifetime.current.active) player.pause();
                },
                (cause: unknown) => {
                  if (lifetime.current.active)
                    setFailed(`无法播放原视频：${errorMessage(cause)}`);
                },
              );
            } else setPlaying(false);
          }}
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
      <div className="action-row preview-step-controls">
        <button
          className="button text"
          onClick={() => step(-1)}
          disabled={position <= 0}
        >
          {t(frameRate > 0 ? "上一帧" : "后退 0.1 秒")}
        </button>
        <button
          className="button text"
          onClick={() => step(1)}
          disabled={position >= total}
        >
          {t(frameRate > 0 ? "下一帧" : "前进 0.1 秒")}
        </button>
        <label className="preview-loop">
          <input
            type="checkbox"
            checked={loop}
            onChange={(event) => setLoop(event.target.checked)}
          />
          {t("循环所选片段")}
        </label>
      </div>
      <p className="field-hint">
        {t("按源平均帧率微调；可变帧率视频的定位为近似值。")}
      </p>
      <label className="field-label" htmlFor={positionID}>
        {t("预览位置")}
      </label>
      <input
        id={positionID}
        type="range"
        min="0"
        max={total}
        step="0.001"
        value={position}
        aria-valuetext={formatTimecode(position)}
        onChange={(event) => seek(Number(event.target.value))}
      />
      <Filmstrip
        key={`${media.source.uri}:${total}`}
        uri={media.source.uri}
        duration={total}
        visible={visible}
        seek={seek}
      />
      <WaveformOverview
        key={media.source.uri}
        media={media}
        position={position}
        visible={visible}
        seek={seek}
      />
      <div className="action-row">
        {onBoundary && (
          <button
            className="button outlined"
            disabled={position >= end}
            onClick={() =>
              onBoundary("startSeconds", Math.floor(position * 1000) / 1000)
            }
          >
            {t("当前位置设为起点")}
          </button>
        )}
        {onBoundary && (
          <button
            className="button outlined"
            disabled={position <= start}
            onClick={() =>
              onBoundary("endSeconds", Math.floor(position * 1000) / 1000)
            }
          >
            {t("当前位置设为终点")}
          </button>
        )}
        <button className="button text" onClick={() => seek(start)}>
          {t("定位起点")}
        </button>
        <button className="button text" onClick={() => seek(end)}>
          {t("定位终点")}
        </button>
      </div>
      {showFrames && (
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
      )}
      {showFrames && (
        <p className="field-hint">
          {t(
            "结束时间为不包含的边界；下方显示结束边界附近画面。预览用于定位，实际切点受帧率影响。",
          )}
        </p>
      )}
    </div>
  );
}
