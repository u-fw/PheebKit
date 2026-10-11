import { useEffect, useState } from "react";
import { API } from "../bridge";
import type { SampleComparison as Comparison, Thumbnail } from "../bridge";
import { errorMessage, formatBytes, formatDuration } from "../format";

export function SampleComparison({
  id,
  close,
}: {
  id: string;
  close: () => void;
}) {
  const [data, setData] = useState<Comparison | null>(null);
  const [position, setPosition] = useState(0);
  const [frames, setFrames] = useState<{
    original: Thumbnail;
    output: Thumbnail;
    position: number;
  } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
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
    if (!data) return;
    let alive = true;
    const timer = setTimeout(() => {
      setLoading(true);
      setError("");
      setFrames(null);
      void Promise.all([
        API.getVideoFrame(
          data.original.source.uri,
          data.startSeconds + position,
        ),
        API.getVideoFrame(
          data.output.source.uri,
          Math.min(position, data.output.durationSeconds),
        ),
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
    }, 300);
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [data, position]);
  const visibleFrames = frames?.position === position ? frames : null;
  return (
    <section
      className="surface settings-card sample-comparison"
      aria-label="样片画面比较"
    >
      <div className="section-title">
        <h2>样片画面比较</h2>
        <button className="button text" onClick={close}>
          关闭比较
        </button>
      </div>
      <p className="field-hint">
        对照相同时间位置的画面。预览缩小到 512
        像素，细节与音质请打开样片文件检查。
      </p>
      {data && (
        <>
          <div
            className="comparison-grid"
            aria-busy={loading || !visibleFrames}
          >
            <figure className="position-frame">
              <div className="frame-image">
                {visibleFrames ? (
                  <img
                    src={visibleFrames.original.dataUrl}
                    alt={`原片 ${formatDuration(data.startSeconds + position)} 的画面`}
                  />
                ) : (
                  <span>正在读取原片画面…</span>
                )}
              </div>
              <figcaption>原片 · {data.original.name}</figcaption>
            </figure>
            <figure className="position-frame">
              <div className="frame-image">
                {visibleFrames ? (
                  <img
                    src={visibleFrames.output.dataUrl}
                    alt={`样片 ${formatDuration(position)} 的画面`}
                  />
                ) : (
                  <span>正在读取样片画面…</span>
                )}
              </div>
              <figcaption>
                样片 · {formatBytes(data.output.sizeBytes)}
              </figcaption>
            </figure>
          </div>
          <label className="field-label" htmlFor="sample-position">
            比较位置 · {formatDuration(position)} /{" "}
            {formatDuration(data.durationSeconds)}
          </label>
          <input
            id="sample-position"
            type="range"
            min={0}
            max={Math.max(0, data.durationSeconds - 0.1)}
            step={0.1}
            value={position}
            onChange={(event) => setPosition(Number(event.target.value))}
          />
        </>
      )}
      {loading && (
        <p role="status" className="field-hint">
          正在读取画面…
        </p>
      )}
      {error && (
        <p className="inline-error" role="alert">
          {error}
        </p>
      )}
    </section>
  );
}
