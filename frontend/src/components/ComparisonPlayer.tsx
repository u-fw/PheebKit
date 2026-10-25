import { useEffect, useId, useRef, useState } from "react";
import { API } from "../bridge";
import type { SampleComparison } from "../bridge";
import { errorMessage } from "../format";
import { t } from "../i18n";
import { comparisonDuration, comparisonPosition } from "../previewTimeline";
import { formatTimecode } from "../trim";
import { MaterialSelect } from "./MaterialSelect";

export function ComparisonPlayer({
  data,
  position,
  onPosition,
  unavailable,
}: {
  data: SampleComparison;
  position: number;
  onPosition: (seconds: number) => void;
  unavailable: (message: string) => void;
}) {
  const source = useRef<HTMLVideoElement>(null);
  const sample = useRef<HTMLVideoElement>(null);
  const lifetime = useRef({ active: true, attempt: 0 });
  const [urls, setURLs] = useState<string[]>([]);
  const [ready, setReady] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [busy, setBusy] = useState(false);
  const [buffering, setBuffering] = useState(false);
  const [audio, setAudio] = useState("sample");
  const sliderID = useId();
  const duration = comparisonDuration(
    data.original.durationSeconds,
    data.output.durationSeconds,
    data.startSeconds,
    data.durationSeconds,
  );
  useEffect(() => {
    const state = lifetime.current;
    state.active = true;
    const original = source.current;
    const output = sample.current;
    void Promise.all([
      API.getMediaPreviewURL(data.original.source.uri),
      API.getMediaPreviewURL(data.output.source.uri),
    ]).then(
      (values) => {
        if (state.active) setURLs(values);
      },
      (cause: unknown) => {
        if (state.active) unavailable(errorMessage(cause));
      },
    );
    return () => {
      state.active = false;
      state.attempt++;
      original?.pause();
      output?.pause();
    };
  }, [data.original.source.uri, data.output.source.uri, unavailable]);
  useEffect(() => {
    if (!playing) return;
    let tick = 0;
    function sync() {
      const original = source.current;
      const output = sample.current;
      if (original && output) {
        if (output.currentTime >= duration) {
          original.pause();
          output.pause();
          setPlaying(false);
          onPosition(duration);
          return;
        }
        const target = comparisonPosition(
          output.currentTime,
          data.startSeconds,
          duration,
        ).source;
        if (Math.abs(original.currentTime - target) > 0.08 && !original.seeking)
          original.currentTime = target;
      }
      tick = requestAnimationFrame(sync);
    }
    tick = requestAnimationFrame(sync);
    return () => cancelAnimationFrame(tick);
  }, [playing, duration, data.startSeconds, onPosition]);
  function pause() {
    lifetime.current.attempt++;
    source.current?.pause();
    sample.current?.pause();
    setBusy(false);
    setPlaying(false);
  }
  function seek(value: number) {
    pause();
    const next = comparisonPosition(value, data.startSeconds, duration);
    if (source.current && source.current.readyState >= 1)
      source.current.currentTime = next.source;
    if (sample.current && sample.current.readyState >= 1)
      sample.current.currentTime = next.sample;
    onPosition(next.sample);
  }
  function waitForData() {
    if (!playing) return;
    pause();
    setBuffering(true);
  }
  async function play() {
    if (playing || busy) {
      pause();
      return;
    }
    const original = source.current;
    const output = sample.current;
    if (!original || !output || ready !== 3 || duration <= 0) return;
    const next = comparisonPosition(
      position >= duration ? 0 : position,
      data.startSeconds,
      duration,
    );
    original.currentTime = next.source;
    output.currentTime = next.sample;
    const attempt = ++lifetime.current.attempt;
    setBuffering(false);
    setBusy(true);
    try {
      await Promise.all([original.play(), output.play()]);
      if (!lifetime.current.active || attempt !== lifetime.current.attempt) {
        original.pause();
        output.pause();
        return;
      }
      setPlaying(true);
    } catch (cause) {
      original.pause();
      output.pause();
      if (lifetime.current.active && attempt === lifetime.current.attempt)
        unavailable(errorMessage(cause));
    } finally {
      if (lifetime.current.active && attempt === lifetime.current.attempt)
        setBusy(false);
    }
  }
  return (
    <div className="comparison-playback">
      <div className="comparison-grid">
        <figure className="position-frame">
          <video
            ref={source}
            src={urls[0]}
            preload="metadata"
            playsInline
            muted={audio !== "original"}
            aria-label={t("原片同步预览")}
            onWaiting={waitForData}
            onLoadedMetadata={(event) => {
              event.currentTarget.currentTime = data.startSeconds + position;
              setReady((value) => value | 1);
            }}
            onError={() => unavailable(t("当前播放器无法播放原片编码。"))}
          />
          <figcaption>
            {t("原片 ·")} {data.original.name}
          </figcaption>
        </figure>
        <figure className="position-frame">
          <video
            ref={sample}
            src={urls[1]}
            preload="metadata"
            playsInline
            muted={audio !== "sample"}
            aria-label={t("样片同步预览")}
            onWaiting={waitForData}
            onLoadedMetadata={(event) => {
              event.currentTarget.currentTime = position;
              setReady((value) => value | 2);
            }}
            onTimeUpdate={(event) =>
              onPosition(Math.min(duration, event.currentTarget.currentTime))
            }
            onPause={() => {
              lifetime.current.attempt++;
              source.current?.pause();
              setBusy(false);
              setPlaying(false);
            }}
            onEnded={pause}
            onError={() => unavailable(t("当前播放器无法播放样片编码。"))}
          />
          <figcaption>
            {t("样片 ·")} {data.output.name}
          </figcaption>
        </figure>
      </div>
      <div className="action-row">
        <button
          className="button outlined"
          disabled={ready !== 3 || duration <= 0}
          onClick={() => void play()}
        >
          {t(playing || busy ? "暂停" : "同步播放")}
        </button>
        <strong>
          {formatTimecode(position)} / {formatTimecode(duration)}
        </strong>
      </div>
      <MaterialSelect
        label={t("试听音频")}
        value={audio}
        options={[
          { value: "sample", label: "样片音频" },
          { value: "original", label: "原片音频" },
          { value: "none", label: "静音" },
        ]}
        onChange={setAudio}
      />
      {buffering && (
        <p className="field-hint" role="status">
          {t("预览缓冲已暂停，请稍后重新播放。")}
        </p>
      )}
      <label className="field-label" htmlFor={sliderID}>
        {t("同步预览位置")}
      </label>
      <input
        id={sliderID}
        type="range"
        min={0}
        max={duration}
        step={0.001}
        value={Math.min(duration, position)}
        aria-valuetext={formatTimecode(position)}
        onChange={(event) => seek(Number(event.target.value))}
      />
      <p className="field-hint">
        {t("原片偏移 {{0}}；以样片时间为基准同步，音频一次只播放一路。", [
          formatTimecode(data.startSeconds),
        ])}
      </p>
      <p className="field-hint">
        {t("原片试听使用播放器默认音轨；输出轨道以执行方案为准。")}
      </p>
    </div>
  );
}
