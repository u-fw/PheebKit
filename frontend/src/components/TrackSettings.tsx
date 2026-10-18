import { t } from "../i18n";
import type { Media, Settings, Stream } from "../bridge";
import { operationOf, selectedTracks } from "../project";
import type { UpdateSettings } from "./EncodingSettings";
import { MaterialSelect } from "./MaterialSelect";

const languages: Record<string, string> = {
  chi: "中文",
  zho: "中文",
  zh: "中文",
  eng: "英语",
  en: "英语",
  jpn: "日语",
  ja: "日语",
  kor: "韩语",
  ko: "韩语",
  und: "语言未标注",
};
export function trackLabel(stream: Stream, ordinal: number) {
  return [
    t(stream.codecType === "audio" ? "音轨 {{0}}" : "字幕 {{0}}", [
      ordinal + 1,
    ]),
    Object.hasOwn(languages, stream.language)
      ? t(languages[stream.language])
      : stream.language || t("语言未标注"),
    stream.codecName.toUpperCase(),
    stream.title,
  ]
    .filter(Boolean)
    .join(" · ");
}

export function TrackSettings({
  media,
  settings,
  update,
}: {
  media: Media;
  settings: Settings;
  update: UpdateSettings;
}) {
  const operation = operationOf(settings);
  const kinds: ("audio" | "subtitle")[] =
    operation === "extract-audio"
      ? ["audio"]
      : operation === "extract-subtitle"
        ? ["subtitle"]
        : ["audio", "subtitle"];
  return (
    <section className="surface settings-card">
      <div className="section-title">
        <h2>{t("音轨与字幕")}</h2>
      </div>
      {kinds.map((kind) => {
        const tracks = media.streams.filter(
          (stream) => stream.codecType === kind,
        );
        const key = kind === "audio" ? "audioSelection" : "subtitleSelection";
        const selection = settings[key];
        const mode = selection?.mode ?? (kind === "audio" ? "all" : "none");
        const single = operation === `extract-${kind}`;
        const chosen = selectedTracks(media, kind, settings);
        return (
          <fieldset key={kind}>
            <legend>{t(kind === "audio" ? "音频" : "字幕")}</legend>
            {tracks.length === 0 ? (
              <p className="field-hint">
                {t("源媒体没有")} {t(kind === "audio" ? "音轨" : "字幕轨")}。
              </p>
            ) : single ? (
              <MaterialSelect
                label={t(kind === "audio" ? "要提取的音轨" : "要提取的字幕")}
                value={String(selection?.indices?.[0] ?? "")}
                options={tracks.map((stream, index) => ({
                  value: String(stream.index),
                  label: trackLabel(stream, index),
                  translate: false,
                }))}
                onChange={(value) =>
                  update(key, { mode: "selected", indices: [Number(value)] })
                }
              />
            ) : (
              <>
                <MaterialSelect
                  label={t(kind === "audio" ? "保留哪些音轨" : "保留哪些字幕")}
                  value={mode}
                  options={[
                    {
                      value: "all",
                      label: "全部保留",
                      description: "保留源媒体的所有轨道",
                    },
                    {
                      value: "none",
                      label:
                        kind === "audio" ? "不保留音频 · 静音" : "不保留字幕",
                      description: "输出文件不包含这些轨道",
                    },
                    {
                      value: "selected",
                      label: "选择轨道",
                      description: "逐条选择要保留的内容",
                    },
                  ]}
                  onChange={(value) =>
                    update(key, {
                      mode: value,
                      indices:
                        value === "selected"
                          ? selection?.indices?.length
                            ? selection.indices
                            : [tracks[0].index]
                          : [],
                    })
                  }
                />
                {mode === "selected" && (
                  <div className="track-list">
                    {tracks.map((stream, index) => (
                      <label className="track-choice" key={stream.index}>
                        <input
                          type="checkbox"
                          checked={Boolean(
                            selection?.indices?.includes(stream.index),
                          )}
                          onChange={(event) =>
                            update(key, {
                              mode: "selected",
                              indices: event.target.checked
                                ? [...(selection?.indices ?? []), stream.index]
                                : (selection?.indices ?? []).filter(
                                    (value) => value !== stream.index,
                                  ),
                            })
                          }
                        />
                        <span>{trackLabel(stream, index)}</span>
                      </label>
                    ))}
                  </div>
                )}
              </>
            )}
            {tracks.length > 0 && (
              <p className="field-hint">
                {t(
                  single
                    ? "一次提取一条轨道，源文件保持原样。"
                    : kind === "subtitle"
                      ? `当前将${t(chosen.length ? `保留 ${chosen.length} 条软字幕` : "移除软字幕轨道")}；画面烧录由下方字幕烧录设置控制。`
                      : `当前将${t(chosen.length ? `保留 ${chosen.length} 条音轨` : "输出静音视频")}。`,
                )}
              </p>
            )}
            {kind === "audio" &&
              settings.audio === "none" &&
              tracks.length > 0 && (
                <p className="field-hint">
                  {t(
                    "音频编码已设为静音；请在输出设置中改为保留原音频或重新编码，才能保留所选音轨。",
                  )}
                </p>
              )}
          </fieldset>
        );
      })}
    </section>
  );
}
