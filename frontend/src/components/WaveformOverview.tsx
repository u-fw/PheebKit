import { useEffect, useRef, useState } from "react";
import { API } from "../bridge";
import type { AudioWaveform, Media } from "../bridge";
import { t } from "../i18n";
import { errorMessage } from "../format";
import { formatTimecode } from "../trim";
import { MaterialSelect } from "./MaterialSelect";

export function WaveformOverview({
  media,
  position,
  visible,
  seek,
}: {
  media: Media;
  position: number;
  visible: boolean;
  seek: (seconds: number) => void;
}) {
  const tracks = media.streams.filter((stream) => stream.codecType === "audio");
  const [track, setTrack] = useState(tracks[0]?.index ?? -1);
  const [result, setResult] = useState<AudioWaveform | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const lifecycle = useRef({ generation: 0 });
  useEffect(() => {
    const state = lifecycle.current;
    return () => {
      state.generation++;
    };
  }, [visible]);
  if (!tracks.length || !(media.durationSeconds > 0)) return null;
  const start = Math.min(
    Math.max(0, position),
    Math.max(0, media.durationSeconds - 0.1),
  );
  const span = Math.min(120, media.durationSeconds - start);
  async function load() {
    if (busy || !visible) return;
    const request = ++lifecycle.current.generation;
    setBusy(true);
    setError("");
    try {
      const value = await API.getAudioWaveform(
        media.source.uri,
        start,
        span,
        track,
      );
      if (request === lifecycle.current.generation) setResult(value);
    } catch (cause) {
      if (request === lifecycle.current.generation)
        setError(errorMessage(cause));
    } finally {
      // A hidden window may finish its bounded request. Always release the
      // button, but never publish stale data or errors from that generation.
      setBusy(false);
    }
  }
  return (
    <div className="waveform-overview">
      <div className="waveform-controls">
        <MaterialSelect
          label={t("波形预览音轨")}
          value={String(track)}
          options={tracks.map((stream) => ({
            value: String(stream.index),
            label: `#${stream.index} · ${stream.codecName} · ${stream.language || "—"}`,
            translate: false,
          }))}
          disabled={busy}
          onChange={(value) => {
            setTrack(Number(value));
            setResult(null);
          }}
        />
        <button
          className="button text"
          onClick={() => void load()}
          disabled={busy}
        >
          {t(busy ? "正在读取波形…" : "读取当前位置波形")}
        </button>
      </div>
      <p className="field-hint">
        {t("单声道概览，每次最多 120 秒；用于定位，不用于测量响度或判断音质。")}
      </p>
      {busy && (
        <p role="status" className="field-hint">
          {t("正在读取波形…")}
        </p>
      )}
      {error && (
        <p role="alert" className="inline-error">
          {t(error)}
        </p>
      )}
      {result && (
        <>
          <svg
            className="waveform-image"
            viewBox="0 0 512 80"
            role="img"
            aria-label={t("音频波形 {{0}} 至 {{1}}", [
              formatTimecode(result.startSeconds),
              formatTimecode(result.startSeconds + result.durationSeconds),
            ])}
          >
            <line
              x1="0"
              x2="512"
              y1="40"
              y2="40"
              className="waveform-baseline"
            />
            <path
              d={result.peaks
                .map(
                  (peak, index) =>
                    `M${((index * 512) / result.peaks.length).toFixed(2)},${(40 - peak * 38).toFixed(2)}v${Math.max(0.5, peak * 76).toFixed(2)}`,
                )
                .join(" ")}
            />
          </svg>
          <input
            type="range"
            aria-label={t("波形预览位置")}
            aria-valuetext={formatTimecode(position)}
            min={result.startSeconds}
            max={result.startSeconds + result.durationSeconds}
            step={0.001}
            value={Math.max(
              result.startSeconds,
              Math.min(result.startSeconds + result.durationSeconds, position),
            )}
            onChange={(event) => seek(Number(event.target.value))}
          />
          <div className="action-row">
            <button
              className="button text"
              onClick={() => seek(result.startSeconds)}
            >
              {formatTimecode(result.startSeconds)}
            </button>
            <span aria-hidden="true">→</span>
            <button
              className="button text"
              onClick={() => seek(result.startSeconds + result.durationSeconds)}
            >
              {formatTimecode(result.startSeconds + result.durationSeconds)}
            </button>
          </div>
        </>
      )}
    </div>
  );
}
