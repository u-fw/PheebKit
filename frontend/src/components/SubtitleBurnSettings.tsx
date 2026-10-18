import { t } from "../i18n";
import { useId, useState } from "react";
import {
  API,
  type Capabilities,
  type Media,
  type Settings,
  type SubtitleBurn,
} from "../bridge";
import {
  defaultSubtitle,
  subtitleError,
  textSubtitleCodecs,
} from "../subtitles";
import { errorMessage } from "../format";
import { MaterialSelect } from "./MaterialSelect";
import { trackLabel } from "./TrackSettings";

export function SubtitleBurnSettings({
  media,
  settings,
  onChange,
  capabilities,
  connected,
}: {
  media: Media;
  settings: Settings;
  onChange: (s: Settings) => void;
  capabilities: Capabilities | null;
  connected: boolean;
}) {
  const [busy, setBusy] = useState(false);
  const pathID = useId();
  const [error, setError] = useState("");
  const b = settings.subtitleBurn;
  const tracks = media.streams.filter(
    (s) => s.codecType === "subtitle" && textSubtitleCodecs.has(s.codecName),
  );
  function update<K extends keyof SubtitleBurn>(
    key: K,
    value: SubtitleBurn[K],
  ) {
    if (b) onChange({ ...settings, subtitleBurn: { ...b, [key]: value } });
  }
  const validation = subtitleError(settings, media);
  return (
    <section className="surface settings-card subtitle-settings">
      <div className="section-title">
        <h2>{t("字幕烧录")}</h2>
        <span className="subtle-label">{t("文字字幕 → 画面")}</span>
      </div>
      <label className="track-choice">
        <input
          type="checkbox"
          checked={Boolean(b)}
          onChange={(event) => {
            const value = defaultSubtitle(tracks[0]?.index);
            if (!tracks.length) value.source = "external";
            onChange({
              ...settings,
              subtitleBurn: event.target.checked ? value : null,
            });
          }}
        />
        <span>{t("把字幕永久显示在视频画面中")}</span>
      </label>
      <p className="field-hint">
        {t(
          "烧录后无法关闭字幕。可与剪辑、拼接、画面调整一起编码；软字幕轨道由“音轨与字幕”单独控制。",
        )}
      </p>
      {capabilities?.filters && !capabilities.filters.includes("subtitles") && (
        <p className="inline-error">
          {t(
            "当前 FFmpeg 缺少 subtitles/libass 滤镜，请更换包含此能力的构建。",
          )}
        </p>
      )}
      {b && (
        <>
          <MaterialSelect
            label={t("烧录字幕来源")}
            value={b.source}
            options={[
              {
                value: "embedded",
                label: "内嵌文字字幕",
                disabled: !tracks.length,
              },
              { value: "external", label: "本地字幕文件" },
            ]}
            onChange={(source) =>
              onChange({
                ...settings,
                subtitleBurn: {
                  ...b,
                  source,
                  path: source === "embedded" ? "" : b.path,
                  streamIndex: tracks[0]?.index ?? 0,
                },
              })
            }
          />
          {b.source === "embedded" ? (
            <MaterialSelect
              label={t("烧录的字幕轨道")}
              value={String(b.streamIndex)}
              options={tracks.map((s, i) => ({
                value: String(s.index),
                label: trackLabel(s, i),
                translate: false,
              }))}
              onChange={(value) => update("streamIndex", Number(value))}
            />
          ) : (
            <div className="field-label">
              <label htmlFor={pathID}>{t("外挂字幕文件")}</label>
              <div className="output-field">
                <input
                  id={pathID}
                  value={b.path ?? ""}
                  onChange={(event) => update("path", event.target.value)}
                  spellCheck={false}
                />
                <button
                  className="button outlined"
                  disabled={!connected || busy}
                  onClick={async () => {
                    setBusy(true);
                    setError("");
                    try {
                      const path = await API.chooseSubtitleFile();
                      if (path) update("path", path);
                    } catch (cause) {
                      setError(errorMessage(cause));
                    } finally {
                      setBusy(false);
                    }
                  }}
                >
                  {t(busy ? "选择中…" : "浏览字幕")}
                </button>
              </div>
            </div>
          )}
          <p className="field-hint">
            {t(
              "支持 SRT / ASS / SSA / WebVTT，外挂字幕建议使用 UTF-8。ASS 内联样式可能覆盖全局外观；不含 OCR 或图片字幕。",
            )}
          </p>
          <div className="field-grid">
            <label className="field-label">
              {t("字幕偏移（秒）")}
              <input
                type="number"
                step="0.1"
                min="-86400"
                max="86400"
                value={b.offsetSeconds}
                onChange={(event) =>
                  update("offsetSeconds", Number(event.target.value))
                }
              />
            </label>
            <label className="field-label">
              {t("字号（8–144）")}
              <input
                type="number"
                min="8"
                max="144"
                step="1"
                value={b.fontSize}
                onChange={(event) =>
                  update("fontSize", Number(event.target.value))
                }
              />
            </label>
          </div>
          <p className="field-hint">
            {t(
              "正偏移让字幕延后，负偏移让字幕提前；时间始终对应源视频，拼接后自动随片段重排。",
            )}
          </p>
          <label className="field-label">
            {t("字体名称")}
            <input
              value={b.font}
              maxLength={64}
              onChange={(event) => update("font", event.target.value)}
            />
          </label>
          <p className="field-hint">
            {t(
              "使用系统已安装字体；找不到字体时 libass 会回退。字体与位置会反映在生成的样片中。",
            )}
          </p>
          <details className="subtitle-appearance">
            <summary>{t("颜色、描边与位置")}</summary>
            <div className="field-grid">
              <label className="field-label">
                {t("文字颜色")}
                <input
                  type="color"
                  value={b.color}
                  onChange={(event) => update("color", event.target.value)}
                />
              </label>
              <label className="field-label">
                {t("描边颜色")}
                <input
                  type="color"
                  value={b.outlineColor}
                  onChange={(event) =>
                    update("outlineColor", event.target.value)
                  }
                />
              </label>
              <label className="field-label">
                {t("描边（0–10）")}
                <input
                  type="number"
                  min="0"
                  max="10"
                  value={b.outline}
                  onChange={(event) =>
                    update("outline", Number(event.target.value))
                  }
                />
              </label>
              <label className="field-label">
                {t("阴影（0–10）")}
                <input
                  type="number"
                  min="0"
                  max="10"
                  value={b.shadow}
                  onChange={(event) =>
                    update("shadow", Number(event.target.value))
                  }
                />
              </label>
              <label className="field-label">
                {t("边距（0–500）")}
                <input
                  type="number"
                  min="0"
                  max="500"
                  value={b.margin}
                  onChange={(event) =>
                    update("margin", Number(event.target.value))
                  }
                />
              </label>
              <MaterialSelect
                label={t("字幕位置")}
                value={String(b.alignment)}
                options={[
                  "左下",
                  "下方居中",
                  "右下",
                  "左侧居中",
                  "画面中心",
                  "右侧居中",
                  "左上",
                  "上方居中",
                  "右上",
                ].map((label, i) => ({ value: String(i + 1), label }))}
                onChange={(value) => update("alignment", Number(value))}
              />
            </div>
            <label className="track-choice">
              <input
                type="checkbox"
                checked={b.bold}
                onChange={(event) => update("bold", event.target.checked)}
              />
              <span>{t("粗体")}</span>
            </label>
          </details>
          {(validation || error) && (
            <p className="inline-error" role="alert">
              {t(validation || error)}
            </p>
          )}
        </>
      )}
    </section>
  );
}
