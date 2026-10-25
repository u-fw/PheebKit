import { t } from "../i18n";
import { useCallback, useEffect, useId, useRef, useState } from "react";
import { API } from "../bridge";
import type { SampleComparison as Comparison, Thumbnail } from "../bridge";
import { errorMessage, formatBytes } from "../format";
import { formatTimecode } from "../trim";
import { comparisonDuration } from "../previewTimeline";
import { ComparisonPlayer } from "./ComparisonPlayer";

export function SampleComparison({
  id,
  close,
}: {
  id: string;
  close: () => void;
}) {
  const [data, setData] = useState<Comparison | null>(null);
  const [mode, setMode] = useState<"playback" | "frames">("playback");
  const [split, setSplit] = useState(false);
  const [reveal, setReveal] = useState(50);
  const [position, setPosition] = useState(0);
  const [frames, setFrames] = useState<{
    original: Thumbnail;
    output: Thumbnail;
    position: number;
  } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [playbackError, setPlaybackError] = useState("");
  const pending = useRef<Promise<void> | null>(null);
  const sliderID = useId();
  const revealID = useId();
  const unavailable = useCallback((message: string) => {
    setPlaybackError(message);
    setMode("frames");
  }, []);
  useEffect(() => {
    let alive = true;
    void API.getSampleComparison(id).then(
      (value) => {
        if (alive) {
          setData(value);
          setLoading(false);
        }
      },
      (cause: unknown) => {
        if (alive) {
          setError(errorMessage(cause));
          setLoading(false);
        }
      },
    );
    return () => {
      alive = false;
    };
  }, [id]);
  useEffect(() => {
    if (
      !data ||
      mode !== "frames" ||
      comparisonDuration(
        data.original.durationSeconds,
        data.output.durationSeconds,
        data.startSeconds,
        data.durationSeconds,
      ) <= 0
    )
      return;
    const comparison = data;
    let alive = true;
    const timer = setTimeout(() => {
      setLoading(true);
      setError("");
      async function read() {
        // Finish the previous bounded pair before starting the latest seek.
        // Superseded timers/waiters never enqueue additional FFmpeg work.
        await pending.current;
        if (!alive) return;
        const task = Promise.all([
          API.getVideoFrame(
            comparison.original.source.uri,
            comparison.startSeconds + position,
          ),
          API.getVideoFrame(comparison.output.source.uri, position),
        ]).then(
          ([original, output]) => {
            if (alive) {
              setFrames({ original, output, position });
              setLoading(false);
            }
          },
          (cause: unknown) => {
            if (alive) {
              setError(errorMessage(cause));
              setLoading(false);
            }
          },
        );
        pending.current = task;
        await task;
      }
      void read();
    }, 300);
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [data, position, mode]);
  const duration = data
    ? comparisonDuration(
        data.original.durationSeconds,
        data.output.durationSeconds,
        data.startSeconds,
        data.durationSeconds,
      )
    : 0;
  const visibleFrames = frames?.position === position ? frames : null;
  return (
    <section
      className="surface settings-card sample-comparison"
      aria-label={t("样片画面比较")}
    >
      <div className="section-title">
        <h2>{t("样片画面比较")}</h2>
        <button className="button text" onClick={close}>
          {t("关闭比较")}
        </button>
      </div>
      <p className="field-hint">
        {t(
          "对照相同时间位置的画面。静帧缩小到 512 像素；一个样片不能代表整片的质量或体积。",
        )}
      </p>
      {data && duration > 0 && (
        <>
          <div className="segmented" role="group" aria-label={t("比较方式")}>
            <button
              className={mode === "playback" ? "selected" : ""}
              aria-pressed={mode === "playback"}
              onClick={() => {
                setPlaybackError("");
                setMode("playback");
              }}
            >
              {t("播放对照")}
            </button>
            <button
              className={mode === "frames" ? "selected" : ""}
              aria-pressed={mode === "frames"}
              onClick={() => setMode("frames")}
            >
              {t("静帧比较")}
            </button>
          </div>
          <p className="field-hint">
            {t("原片偏移")} {formatTimecode(data.startSeconds)} · {t("样片 ·")}{" "}
            {formatBytes(data.output.sizeBytes)}
          </p>
          {playbackError && (
            <p className="notice" role="status">
              {t("同步播放不可用，已切换到 FFmpeg 静帧比较：")}{" "}
              {t(playbackError)}
            </p>
          )}
          {mode === "playback" ? (
            <ComparisonPlayer
              data={data}
              position={position}
              onPosition={setPosition}
              unavailable={unavailable}
            />
          ) : (
            <>
              <div
                className="segmented"
                role="group"
                aria-label={t("静帧布局")}
              >
                <button
                  className={!split ? "selected" : ""}
                  aria-pressed={!split}
                  onClick={() => setSplit(false)}
                >
                  {t("并排对比")}
                </button>
                <button
                  className={split ? "selected" : ""}
                  aria-pressed={split}
                  onClick={() => setSplit(true)}
                >
                  {t("滑动对比")}
                </button>
              </div>
              {split ? (
                <>
                  <div
                    className="comparison-split"
                    aria-busy={loading || !visibleFrames}
                  >
                    {visibleFrames ? (
                      <>
                        <img
                          src={visibleFrames.output.dataUrl}
                          alt={t("右侧样片画面")}
                        />
                        <img
                          className="comparison-original"
                          src={visibleFrames.original.dataUrl}
                          alt={t("左侧原片画面")}
                          style={{ clipPath: `inset(0 ${100 - reveal}% 0 0)` }}
                        />
                        <span
                          className="comparison-divider"
                          style={{ left: `${reveal}%` }}
                          aria-hidden="true"
                        />
                      </>
                    ) : (
                      <span>{t("正在读取画面…")}</span>
                    )}
                  </div>
                  <label className="field-label" htmlFor={revealID}>
                    {t("原片显示比例")} · {reveal}%
                  </label>
                  <input
                    id={revealID}
                    type="range"
                    min={0}
                    max={100}
                    value={reveal}
                    onChange={(event) => setReveal(Number(event.target.value))}
                  />
                  <p className="field-hint">
                    {t(
                      "左侧原片，右侧样片；两者按预览框适配，裁剪或旋转后不保证像素位置一致。",
                    )}
                  </p>
                </>
              ) : (
                <div
                  className="comparison-grid"
                  aria-busy={loading || !visibleFrames}
                >
                  <figure className="position-frame">
                    <div className="frame-image">
                      {visibleFrames ? (
                        <img
                          src={visibleFrames.original.dataUrl}
                          alt={t("原片 {{0}} 的画面", [
                            formatTimecode(data.startSeconds + position),
                          ])}
                        />
                      ) : (
                        <span>{t("正在读取原片画面…")}</span>
                      )}
                    </div>
                    <figcaption>
                      {t("原片 ·")} {data.original.name}
                    </figcaption>
                  </figure>
                  <figure className="position-frame">
                    <div className="frame-image">
                      {visibleFrames ? (
                        <img
                          src={visibleFrames.output.dataUrl}
                          alt={t("样片 {{0}} 的画面", [
                            formatTimecode(position),
                          ])}
                        />
                      ) : (
                        <span>{t("正在读取样片画面…")}</span>
                      )}
                    </div>
                    <figcaption>
                      {t("样片 ·")} {data.output.name}
                    </figcaption>
                  </figure>
                </div>
              )}
              <label className="field-label" htmlFor={sliderID}>
                {t("比较位置 ·")} {formatTimecode(position)} /{" "}
                {formatTimecode(duration)}
              </label>
              <input
                id={sliderID}
                type="range"
                min={0}
                max={Math.max(0, duration - 0.001)}
                step={0.001}
                value={position}
                aria-valuetext={formatTimecode(position)}
                onChange={(event) => setPosition(Number(event.target.value))}
              />
            </>
          )}
        </>
      )}
      {loading && (mode === "frames" || !data) && (
        <p role="status" className="field-hint">
          {t("正在读取画面…")}
        </p>
      )}
      {data && duration <= 0 && (
        <p className="inline-error" role="alert">
          {t("样片可比较时长无效，请重新生成样片。")}
        </p>
      )}
      {error && (
        <p className="inline-error" role="alert">
          {t(error)}
        </p>
      )}
    </section>
  );
}
