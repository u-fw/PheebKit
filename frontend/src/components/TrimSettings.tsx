import { useState } from "react";
import type { Media, Settings } from "../bridge";
import {
  formatTimecode,
  parseTimecode,
  selectedDuration,
  trimError,
} from "../trim";

export function TrimSettings({
  media,
  settings,
  onChange,
}: {
  media: Media;
  settings: Settings;
  onChange: (settings: Settings) => void;
}) {
  const trim = settings.trim;
  const total = media.durationSeconds;
  const known = Number.isFinite(total) && total >= 0.001;
  const max = known ? Math.floor(total * 1000) / 1000 : 0;
  const [startText, setStartText] = useState(() =>
    formatTimecode(trim?.startSeconds ?? 0),
  );
  const [endText, setEndText] = useState(() =>
    formatTimecode(trim?.endSeconds ?? max),
  );
  const video = media.streams.find(
    (stream) =>
      stream.codecType === "video" &&
      !stream.attachedPicture &&
      !stream.timedThumbnails,
  );
  const error = trimError(settings, total, video?.frameRate);
  const hasAudio = media.streams.some((stream) => stream.codecType === "audio");

  function enable(enabled: boolean) {
    if (!enabled) {
      onChange({ ...settings, trim: null });
      return;
    }
    const startSeconds = parseTimecode(startText);
    const endSeconds = parseTimecode(endText);
    onChange({
      ...settings,
      trim: { startSeconds, endSeconds },
      audio: hasAudio && settings.audio === "copy" ? "aac" : settings.audio,
    });
  }

  function change(boundary: "startSeconds" | "endSeconds", text: string) {
    if (!trim) return;
    if (boundary === "startSeconds") setStartText(text);
    else setEndText(text);
    onChange({
      ...settings,
      trim: { ...trim, [boundary]: parseTimecode(text) },
    });
  }

  return (
    <section
      className="surface settings-card trim-card"
      aria-labelledby="trim-heading"
    >
      <div className="section-title">
        <h2 id="trim-heading">处理范围</h2>
      </div>
      <div className="segmented" aria-label="处理范围">
        <button
          type="button"
          aria-pressed={!trim}
          className={!trim ? "selected" : ""}
          onClick={() => enable(false)}
        >
          完整视频
        </button>
        <button
          type="button"
          aria-pressed={Boolean(trim)}
          className={trim ? "selected" : ""}
          disabled={!known}
          onClick={() => enable(true)}
        >
          剪辑并压缩
        </button>
      </div>
      <p className="field-hint" id="trim-help">
        {known
          ? `源视频总时长 ${formatTimecode(total)}。${trim ? "仅保留所选片段，在同一次处理中完成压缩。" : "保留完整时长，并按下方设置压缩。"}`
          : "视频时长未知，暂不支持选择片段。"}
      </p>
      {trim && (
        <>
          <div className="field-grid trim-fields">
            {(["startSeconds", "endSeconds"] as const).map((boundary) => {
              const label =
                boundary === "startSeconds" ? "开始时间" : "结束时间";
              const text = boundary === "startSeconds" ? startText : endText;
              return (
                <div key={boundary}>
                  <label className="field-label" htmlFor={`trim-${boundary}`}>
                    {label}
                  </label>
                  <input
                    id={`trim-${boundary}`}
                    value={text}
                    onChange={(event) => change(boundary, event.target.value)}
                    spellCheck={false}
                    aria-invalid={Boolean(error)}
                    aria-describedby={`trim-help trim-format${error ? " trim-error" : ""}`}
                  />
                  <input
                    className="trim-slider"
                    type="range"
                    min="0"
                    max={max}
                    step="0.001"
                    value={Math.min(max, Math.max(0, trim[boundary]))}
                    aria-label={`${label}滑块`}
                    aria-valuetext={
                      formatTimecode(trim[boundary]) || "无效时间"
                    }
                    onChange={(event) =>
                      change(
                        boundary,
                        formatTimecode(Number(event.target.value)),
                      )
                    }
                  />
                </div>
              );
            })}
          </div>
          <p className="field-hint" id="trim-format">
            支持
            时:分:秒.毫秒、分:秒，或秒数。实际切点受视频帧和音频采样精度限制。
          </p>
          {error ? (
            <p id="trim-error" className="inline-error" role="alert">
              {error}
            </p>
          ) : (
            <p className="trim-duration" role="status">
              保留时长{" "}
              <strong>
                {formatTimecode(selectedDuration(settings, total))}
              </strong>
            </p>
          )}
          {hasAudio && (
            <p className="field-hint">
              剪辑时使用 AAC / Opus 对齐音频；从 Copy 切入时自动选择 AAC。
            </p>
          )}
        </>
      )}
    </section>
  );
}
