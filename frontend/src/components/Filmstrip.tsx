import { useEffect, useRef, useState } from "react";
import { API } from "../bridge";
import type { Thumbnail } from "../bridge";
import { errorMessage } from "../format";
import { t } from "../i18n";
import { filmstripPositions } from "../previewTimeline";
import { formatTimecode } from "../trim";

type Cell = { seconds: number; image?: Thumbnail; error?: string };

// One in-flight frame at a time. Hiding/unmounting stops the remaining work;
// the existing backend thumbnail gate and timeout bound the current request.
export function Filmstrip({
  uri,
  duration,
  visible,
  seek,
}: {
  uri: string;
  duration: number;
  visible: boolean;
  seek: (seconds: number) => void;
}) {
  const [enabled, setEnabled] = useState(false);
  const [cells, setCells] = useState<Cell[]>([]);
  const [attempt, setAttempt] = useState(0);
  const cache = useRef<Cell[]>([]);
  const positions = filmstripPositions(duration);
  const count = positions.length;
  useEffect(() => {
    if (!visible || !enabled) return;
    let alive = true;
    async function load() {
      const results: Cell[] = [...cache.current];
      for (const seconds of filmstripPositions(duration)) {
        if (!alive) break;
        if (results.some((cell) => cell.seconds === seconds)) continue;
        let cell: Cell;
        try {
          cell = { seconds, image: await API.getVideoFrame(uri, seconds) };
        } catch (cause) {
          cell = { seconds, error: errorMessage(cause) };
        }
        if (!alive) break;
        results.push(cell);
        cache.current = results;
        setCells([...results]);
      }
    }
    void load();
    return () => {
      alive = false;
    };
  }, [uri, duration, visible, enabled, attempt]);
  const failed = cells.some((cell) => cell.error);
  return (
    <div className="filmstrip-overview">
      <div className="action-row">
        <button
          className="button text"
          disabled={!count || (enabled && cells.length < count)}
          onClick={() => {
            cache.current = [];
            setCells([]);
            setEnabled(true);
            setAttempt((value) => value + 1);
          }}
        >
          {t(
            !enabled
              ? "加载时间轴缩略图"
              : cells.length < count
                ? "正在读取缩略图…"
                : "重新加载缩略图",
          )}
        </button>
        {enabled && (
          <span className="field-hint" role="status">
            {t("已读取 {{0}} / {{1}}", [cells.length, count])}
          </span>
        )}
      </div>
      {enabled && (
        <div
          className="filmstrip-grid"
          role="group"
          aria-label={t("时间轴缩略图")}
        >
          {positions.map((seconds) => {
            const cell = cells.find((item) => item.seconds === seconds);
            return (
              <button
                className="filmstrip-cell"
                key={seconds}
                onClick={() => seek(seconds)}
                aria-label={t("定位到 {{0}}", [formatTimecode(seconds)])}
              >
                <span className="filmstrip-image">
                  {cell?.image ? (
                    <img
                      src={cell.image.dataUrl}
                      alt=""
                      width={cell.image.width}
                      height={cell.image.height}
                    />
                  ) : (
                    <span>{t(cell?.error ? "不可用" : "等待画面")}</span>
                  )}
                </span>
                <span>{formatTimecode(seconds)}</span>
              </button>
            );
          })}
        </div>
      )}
      {failed && (
        <p className="field-hint" role="status">
          {t("部分缩略图无法读取；仍可点击时间点定位。")}
        </p>
      )}
      {enabled && (
        <p className="field-hint">
          {t("仅抽取最多 8 个源画面用于定位，不代表逐帧解码校验。")}
        </p>
      )}
    </div>
  );
}
