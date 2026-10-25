import { t } from "../i18n";
import { useState } from "react";
import type { Media, Settings } from "../bridge";
import { formatTimecode, selectedDuration, trimError } from "../trim";
import { TimePicker } from "./TimePicker";
import { ClipPreview } from "./ClipPreview";
import { SegmentsEditor } from "./SegmentsEditor";

export function TrimSettings({
  media,
  settings,
  onChange,
  visible = true,
  exportParts,
  canExport = false,
  preview = true,
}: {
  media: Media;
  settings: Settings;
  onChange: (settings: Settings) => void;
  visible?: boolean;
  exportParts?: () => void;
  canExport?: boolean;
  preview?: boolean;
}) {
  const trim = settings.trim;
  const total = media.durationSeconds;
  const known = Number.isFinite(total) && total >= 0.001;
  const max = known ? Math.floor(total * 1000) / 1000 : 0;
  const [range, setRange] = useState(
    () => trim ?? { startSeconds: 0, endSeconds: max },
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
      onChange({ ...settings, trim: null, segments: null });
      return;
    }
    onChange({
      ...settings,
      trim: range,
      segments: null,
      audio: hasAudio && settings.audio === "copy" ? "aac" : settings.audio,
    });
  }

  function change(boundary: "startSeconds" | "endSeconds", value: number) {
    if (!trim) return;
    const next = { ...trim, [boundary]: value };
    setRange(next);
    onChange({
      ...settings,
      trim: next,
    });
  }

  return (
    <section
      className="surface settings-card trim-card"
      aria-labelledby="trim-heading"
    >
      <div className="section-title">
        <h2 id="trim-heading">{t("处理范围")}</h2>
      </div>
      <div className="segmented" role="group" aria-label={t("处理范围")}>
        <button
          type="button"
          aria-pressed={!trim && !settings.segments?.length}
          className={!trim && !settings.segments?.length ? "selected" : ""}
          onClick={() => enable(false)}
        >
          {t("完整视频")}
        </button>
        <button
          type="button"
          aria-pressed={Boolean(trim)}
          className={trim ? "selected" : ""}
          disabled={!known}
          onClick={() => enable(true)}
        >
          {t("剪辑并压缩")}
        </button>
        <button
          type="button"
          aria-pressed={Boolean(settings.segments?.length)}
          className={settings.segments?.length ? "selected" : ""}
          disabled={!known}
          onClick={() =>
            onChange({
              ...settings,
              trim: null,
              segments: settings.segments?.length
                ? settings.segments
                : [trim ?? range],
              audio:
                hasAudio && settings.audio === "copy" ? "aac" : settings.audio,
            })
          }
        >
          {t("多片段拼接")}
        </button>
      </div>
      <p className="field-hint" id="trim-help">
        {t(
          known
            ? `源视频总时长 ${formatTimecode(total)}。${t(settings.segments?.length ? "按列表保留片段并拼接。" : trim ? "仅保留所选片段，在同一次处理中完成压缩。" : "保留完整时长，并按下方设置压缩。")}`
            : "视频时长未知，暂不支持选择片段。",
        )}
      </p>
      {Boolean(settings.segments?.length) && (
        <SegmentsEditor
          media={media}
          settings={settings}
          onChange={onChange}
          visible={visible}
          exportParts={exportParts}
          canExport={canExport}
        />
      )}
      {trim && (
        <>
          <div className="action-row">
            <strong>
              {t("保留")}
              {t(" ")}
              {t(
                formatTimecode(
                  Math.max(0, trim.endSeconds - trim.startSeconds),
                ),
              )}
            </strong>
            <span className="field-hint">
              {t("删除")}
              {t(" ")}
              {t(
                formatTimecode(
                  Math.max(
                    0,
                    total - Math.max(0, trim.endSeconds - trim.startSeconds),
                  ),
                ),
              )}
            </span>
            <button
              className="button text"
              onClick={() => change("startSeconds", 0)}
            >
              {t("起点归零")}
            </button>
            <button
              className="button text"
              onClick={() => change("endSeconds", max)}
            >
              {t("终点归尾")}
            </button>
            <button
              className="button text"
              onClick={() => {
                const next = { startSeconds: 0, endSeconds: max };
                setRange(next);
                onChange({ ...settings, trim: next });
              }}
            >
              {t("重置范围")}
            </button>
          </div>
          <div className="field-grid trim-fields">
            {(["startSeconds", "endSeconds"] as const).map((boundary) => {
              const label =
                boundary === "startSeconds" ? "开始时间" : "结束时间";
              return (
                <div key={boundary}>
                  <TimePicker
                    label={t(label)}
                    value={trim[boundary]}
                    max={max}
                    onChange={(value) => change(boundary, value)}
                    invalid={Boolean(error)}
                    describedBy={`trim-help trim-format${error ? " trim-error" : ""}`}
                  />
                  <input
                    className="trim-slider"
                    type="range"
                    min="0"
                    max={max}
                    step="0.001"
                    value={Math.min(max, Math.max(0, trim[boundary]))}
                    aria-label={t("{{0}}滑块", [label])}
                    aria-valuetext={
                      formatTimecode(trim[boundary]) || t("无效时间")
                    }
                    onChange={(event) =>
                      change(boundary, Number(event.target.value))
                    }
                  />
                </div>
              );
            })}
          </div>
          <p className="field-hint" id="trim-format">
            {t(
              "点击数字直接输入时、分、秒，右侧箭头打开候选列表。使用 ±0.1 秒按钮微调，滑块保留毫秒精度。实际切点受视频帧和音频采样精度限制。",
            )}
          </p>
          {error ? (
            <p id="trim-error" className="inline-error" role="alert">
              {t(error)}
            </p>
          ) : (
            <p className="trim-duration" role="status">
              {t("保留时长")}
              {t(" ")}
              <strong>
                {t(formatTimecode(selectedDuration(settings, total)))}
              </strong>
            </p>
          )}
          {hasAudio && (
            <p className="field-hint">
              {t(
                "剪辑时使用 AAC / Opus 对齐音频；从 Copy 切入时自动选择 AAC。",
              )}
            </p>
          )}
          {preview && (
            <ClipPreview
              media={media}
              start={trim.startSeconds}
              end={trim.endSeconds}
              visible={visible}
              onBoundary={change}
            />
          )}
        </>
      )}
    </section>
  );
}
