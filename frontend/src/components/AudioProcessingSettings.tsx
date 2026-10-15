import type { AudioProcessing, Capabilities, Media, Settings } from "../bridge";
import {
  audioProcessingErrors,
  audioSampleRates,
  defaultAudioProcessing,
} from "../audioProcessing";
import { selectedTracks } from "../project";
import { selectedDuration } from "../trim";
import type { UpdateSettings } from "./EncodingSettings";
import { MaterialSelect } from "./MaterialSelect";

export function AudioProcessingSettings({
  media,
  settings,
  capabilities,
  update,
}: {
  media: Media;
  settings: Settings;
  capabilities: Capabilities | null;
  update: UpdateSettings;
}) {
  const p = settings.audioProcessing;
  const tracks = selectedTracks(media, "audio", settings);
  const errors = audioProcessingErrors(
    settings,
    selectedDuration(settings, media.durationSeconds),
    tracks.length,
  );
  function edit<K extends keyof AudioProcessing>(
    key: K,
    value: AudioProcessing[K],
  ) {
    if (p) update("audioProcessing", { ...p, [key]: value });
  }
  const missing =
    p && capabilities?.filters
      ? [
          "aresample",
          ...(p.normalize ? ["loudnorm"] : []),
          ...(p.denoiseReductionDB > 0 ? ["afftdn"] : []),
          ...((p.equalizer ?? []).some((b) => b.gainDB !== 0)
            ? ["equalizer"]
            : []),
          ...(p.fadeInSeconds || p.fadeOutSeconds ? ["afade"] : []),
        ].filter((f) => !capabilities.filters?.includes(f))
      : [];
  return (
    <details className="surface advanced-card audio-processing">
      <summary>
        <span>声音处理</span>
        <span>{p ? "已启用" : "可选"}</span>
      </summary>
      <div className="advanced-content">
        <p className="field-hint">
          分别处理每条选中音轨；拼接任务在拼接后统一处理，独立导出则逐个文件处理。
        </p>
        <label className="track-choice">
          <input
            type="checkbox"
            checked={Boolean(p)}
            onChange={(event) =>
              update(
                "audioProcessing",
                event.target.checked ? defaultAudioProcessing() : null,
              )
            }
          />
          <span>启用声音处理</span>
        </label>
        {p && (
          <>
            <p className="field-hint">
              选中 {tracks.length} 条音轨
              {tracks[0]?.sampleRate
                ? ` · 首轨 ${tracks[0].sampleRate / 1000} kHz / ${tracks[0].channels || "未知"} 声道`
                : ""}
              。启用后默认输出 48 kHz，可自行调整。
            </p>
            <div className="field-grid">
              <MaterialSelect
                label="输出采样率"
                value={String(p.sampleRate)}
                onChange={(v) => edit("sampleRate", Number(v))}
                options={[
                  { value: "0", label: "自动 · 48 kHz" },
                  ...audioSampleRates
                    .filter((rate) =>
                      settings.audio === "opus"
                        ? rate === 48000
                        : settings.audio === "mp3"
                          ? rate <= 48000
                          : true,
                    )
                    .map((rate) => ({
                      value: String(rate),
                      label: `${rate / 1000} kHz`,
                    })),
                ]}
              />
              <MaterialSelect
                label="输出声道"
                value={p.channels || "source"}
                onChange={(v) => edit("channels", v)}
                options={[
                  { value: "source", label: "保留源声道" },
                  { value: "mono", label: "单声道" },
                  { value: "stereo", label: "立体声" },
                ]}
              />
            </div>
            <label className="track-choice">
              <input
                type="checkbox"
                checked={Boolean(p.normalize)}
                onChange={(event) =>
                  edit(
                    "normalize",
                    event.target.checked
                      ? { integratedLUFS: -16, truePeakDBTP: -1.5, rangeLU: 11 }
                      : null,
                  )
                }
              />
              <span>动态响度归一化 · 单遍</span>
            </label>
            {p.normalize && (
              <>
                <div className="audio-parameter-grid">
                  {(
                    [
                      {
                        key: "integratedLUFS",
                        label: "目标响度 (LUFS)",
                        min: -70,
                        max: -5,
                        step: 0.5,
                      },
                      {
                        key: "truePeakDBTP",
                        label: "真峰值上限 (dBTP)",
                        min: -9,
                        max: 0,
                        step: 0.1,
                      },
                      {
                        key: "rangeLU",
                        label: "目标响度范围 (LU)",
                        min: 1,
                        max: 50,
                        step: 1,
                      },
                    ] as const
                  ).map((field) => (
                    <label className="field-label" key={field.key}>
                      {field.label}
                      <input
                        type="number"
                        min={field.min}
                        max={field.max}
                        step={field.step}
                        value={p.normalize![field.key]}
                        onChange={(event) =>
                          edit("normalize", {
                            ...p.normalize!,
                            [field.key]: Number(event.target.value),
                          })
                        }
                      />
                    </label>
                  ))}
                </div>
                <p className="field-hint">
                  自动平衡响度，可能改变动态。目标不等于编码后实测值；样片与整片可能不同，请试听确认。
                </p>
              </>
            )}
            <label className="track-choice">
              <input
                type="checkbox"
                checked={p.denoiseReductionDB > 0}
                onChange={(event) =>
                  edit("denoiseReductionDB", event.target.checked ? 12 : 0)
                }
              />
              <span>FFT 降噪</span>
            </label>
            <>
              <div className="field-grid">
                <label className="field-label">
                  降噪强度 (dB)
                  <input
                    type="number"
                    min="0"
                    max="97"
                    step="0.01"
                    value={p.denoiseReductionDB}
                    onBlur={(event) => {
                      event.currentTarget.value = String(p.denoiseReductionDB);
                    }}
                    onChange={(e) =>
                      edit("denoiseReductionDB", Number(e.target.value))
                    }
                  />
                </label>
                <label className="field-label">
                  噪声底 (dB)
                  <input
                    type="number"
                    min="-80"
                    max="-20"
                    step="1"
                    value={p.noiseFloorDB}
                    onChange={(e) =>
                      edit("noiseFloorDB", Number(e.target.value))
                    }
                  />
                </label>
              </div>
              <p className="field-hint">
                强度 0 关闭降噪，输入正值即启用；参数保持可编辑，支持 0.01–97
                dB。
              </p>
            </>
            <div className="section-title">
              <h3>均衡器</h3>
              <span className="field-hint">最多 8 段；增益 0 为旁路</span>
            </div>
            {(p.equalizer ?? []).map((band, index) => (
              <fieldset className="audio-eq-band" key={index}>
                <legend>频段 {index + 1}</legend>
                <div className="audio-parameter-grid">
                  {(
                    [
                      {
                        key: "frequencyHz",
                        label: "中心频率 (Hz)",
                        min: 20,
                        max: 20000,
                        step: 1,
                      },
                      { key: "q", label: "Q 值", min: 0.1, max: 20, step: 0.1 },
                      {
                        key: "gainDB",
                        label: "增益 (dB)",
                        min: -24,
                        max: 24,
                        step: 0.5,
                      },
                    ] as const
                  ).map((field) => (
                    <label className="field-label" key={field.key}>
                      频段 {index + 1} {field.label}
                      <input
                        type="number"
                        min={field.min}
                        max={field.max}
                        step={field.step}
                        value={band[field.key]}
                        onChange={(event) =>
                          edit(
                            "equalizer",
                            (p.equalizer ?? []).map((b, i) =>
                              i === index
                                ? {
                                    ...b,
                                    [field.key]: Number(event.target.value),
                                  }
                                : b,
                            ),
                          )
                        }
                      />
                    </label>
                  ))}
                </div>
                <button
                  type="button"
                  className="button text"
                  onClick={() =>
                    edit(
                      "equalizer",
                      (p.equalizer ?? []).filter((_, i) => i !== index),
                    )
                  }
                >
                  移除频段 {index + 1}
                </button>
              </fieldset>
            ))}
            <button
              type="button"
              className="button outlined"
              disabled={(p.equalizer?.length ?? 0) >= 8}
              onClick={() =>
                edit("equalizer", [
                  ...(p.equalizer ?? []),
                  { frequencyHz: 1000, q: 1, gainDB: 0 },
                ])
              }
            >
              添加均衡频段
            </button>
            <p className="field-hint">
              Q
              越高，影响频带越窄。有效中心频率必须小于输出采样率的一半。大幅增益可能产生失真。
            </p>
            <div className="field-grid">
              <label className="field-label">
                输出开头淡入 (秒)
                <input
                  type="number"
                  min="0"
                  max="600"
                  step=".1"
                  value={p.fadeInSeconds}
                  onChange={(e) =>
                    edit("fadeInSeconds", Number(e.target.value))
                  }
                />
              </label>
              <label className="field-label">
                输出结尾淡出 (秒)
                <input
                  type="number"
                  min="0"
                  max="600"
                  step=".1"
                  value={p.fadeOutSeconds}
                  onChange={(e) =>
                    edit("fadeOutSeconds", Number(e.target.value))
                  }
                />
              </label>
            </div>
            <p className="field-hint">
              0 为关闭；淡入淡出总长不能超过输出时长。样片不应用整片淡入淡出。
            </p>
            <button
              type="button"
              className="button text"
              onClick={() =>
                update("audioProcessing", defaultAudioProcessing())
              }
            >
              重置声音处理参数
            </button>
            {(settings.audio === "copy" || settings.audio === "none") && (
              <button
                type="button"
                className="button outlined"
                onClick={() =>
                  update(
                    "audio",
                    settings.codec === "av1" &&
                      settings.operation !== "extract-audio"
                      ? "opus"
                      : "aac",
                  )
                }
              >
                切换到可处理的音频编码
              </button>
            )}
          </>
        )}
        {(errors.length > 0 || missing.length > 0) && (
          <div className="field-notice error-notice" role="alert">
            <div>
              {errors.map((e) => (
                <p key={e}>{e}</p>
              ))}
              {missing.length > 0 && (
                <p>
                  当前 FFmpeg 缺少滤镜：{missing.join("、")}。请切换完整构建。
                </p>
              )}
            </div>
          </div>
        )}
      </div>
    </details>
  );
}
