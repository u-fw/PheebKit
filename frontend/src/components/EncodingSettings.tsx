import { useState } from "react";
import type { Capabilities, Settings } from "../bridge";
import { parseRawArguments } from "../project";
import { Icon } from "./Icon";
import { MaterialSelect } from "./MaterialSelect";

export type UpdateSettings = <K extends keyof Settings>(
  key: K,
  value: Settings[K],
) => void;
export const codecOptions = [
  { value: "h264", title: "H.264", note: "广泛兼容" },
  { value: "hevc", title: "H.265 / HEVC", note: "更小体积" },
  { value: "av1", title: "AV1", note: "新一代编码" },
];
const qualities = [
  { value: 30, label: "更小" },
  { value: 26, label: "均衡" },
  { value: 22, label: "高清" },
  { value: 18, label: "极高" },
];
const modes = [
  { value: "quality", label: "质量优先" },
  { value: "bitrate", label: "目标码率" },
  { value: "size", label: "目标体积" },
];
const resolutions = [
  { value: "0", label: "保持原始分辨率", description: "沿用源视频尺寸" },
  { value: "2160", label: "2160p · 4K", description: "超高清" },
  { value: "1440", label: "1440p · QHD", description: "清晰与体积兼顾" },
  { value: "1080", label: "1080p · Full HD", description: "通用高清" },
  { value: "custom", label: "自定义高度", description: "保持宽高比" },
];
const audioOptions = [
  { value: "copy", label: "Copy", description: "保留原始音频" },
  { value: "aac", label: "AAC", description: "通用兼容" },
  { value: "opus", label: "Opus", description: "高效编码" },
];
const presets = [
  { value: "fast", label: "快速", description: "Fast · 优先处理速度" },
  { value: "medium", label: "均衡", description: "Medium · 速度与压缩兼顾" },
  {
    value: "slow",
    label: "精细",
    description: "Slow · 用更多时间换取压缩效率",
  },
];

export function EncodingSettings({
  settings,
  update,
  capabilities,
  customResolution,
  setCustomResolution,
}: {
  settings: Settings;
  update: UpdateSettings;
  capabilities: Capabilities | null;
  customResolution: boolean;
  setCustomResolution: (value: boolean) => void;
}) {
  return (
    <section className="surface settings-card">
      <div className="section-title">
        <h2>输出设置</h2>
        <span className="subtle-label">01 / FORMAT</span>
      </div>
      <fieldset>
        <legend>
          视频编码 <span>Codec</span>
        </legend>
        <div className="codec-options">
          {codecOptions.map((codec) => {
            const available = capabilities?.codecs.some(
              (entry) => entry.id === codec.value && entry.available,
            );
            return (
              <button
                type="button"
                className={`codec-option ${settings.codec === codec.value ? "selected" : ""}`}
                key={codec.value}
                aria-pressed={settings.codec === codec.value}
                disabled={capabilities !== null && !available}
                onClick={() => update("codec", codec.value)}
              >
                <span>
                  {codec.title}
                  {settings.codec === codec.value && <Icon name="check" />}
                </span>
                <small>
                  {capabilities !== null && !available
                    ? "当前构建不可用"
                    : codec.note}
                </small>
              </button>
            );
          })}
        </div>
      </fieldset>
      <fieldset>
        <legend>
          压缩模式 <span>Rate control</span>
        </legend>
        <div className="segmented mode-options">
          {modes.map((mode) => (
            <button
              type="button"
              className={settings.mode === mode.value ? "selected" : ""}
              key={mode.value}
              aria-pressed={settings.mode === mode.value}
              onClick={() => update("mode", mode.value)}
            >
              {settings.mode === mode.value && <Icon name="check" />}
              {mode.label}
            </button>
          ))}
        </div>
      </fieldset>
      {settings.mode === "quality" ? (
        <fieldset>
          <legend>
            画质 <span>Quality</span>
          </legend>
          <div className="segmented quality-options">
            {qualities.map((quality) => (
              <button
                type="button"
                className={settings.quality === quality.value ? "selected" : ""}
                key={quality.value}
                aria-pressed={settings.quality === quality.value}
                onClick={() => update("quality", quality.value)}
              >
                {settings.quality === quality.value && <Icon name="check" />}
                {quality.label}
              </button>
            ))}
          </div>
          <div className="field-hint split-hint">
            <span>更轻巧</span>
            <span>保留更多细节</span>
          </div>
        </fieldset>
      ) : (
        <label className="field-label target-field">
          {settings.mode === "bitrate"
            ? "目标视频码率 (kbps)"
            : "目标文件大小 (MB)"}
          <input
            type="number"
            min="1"
            value={
              settings.mode === "bitrate"
                ? settings.bitrateKbps
                : settings.targetSizeMB
            }
            onChange={(event) =>
              update(
                settings.mode === "bitrate" ? "bitrateKbps" : "targetSizeMB",
                Number(event.target.value),
              )
            }
          />
          <span className="field-hint">
            {settings.mode === "size"
              ? "根据时长计算视频码率；容器开销可能使实际大小略有差异。"
              : "音频和容器开销会计入最终文件大小。"}
          </span>
        </label>
      )}
      <div className="field-grid">
        <MaterialSelect
          label="分辨率"
          value={customResolution ? "custom" : String(settings.resolution)}
          options={resolutions}
          onChange={(value) => {
            const custom = value === "custom";
            setCustomResolution(custom);
            update("resolution", custom ? 720 : Number(value));
          }}
        />
        <MaterialSelect
          label="音频"
          value={settings.audio}
          options={audioOptions}
          onChange={(value) => update("audio", value)}
        />
      </div>
      {customResolution && (
        <label className="field-label">
          输出高度 (px)
          <input
            type="number"
            min="2"
            max="16384"
            step="2"
            value={settings.resolution}
            onChange={(event) =>
              update("resolution", Number(event.target.value))
            }
          />
          <span className="field-hint">保持宽高比；宽度由编码器自动计算。</span>
        </label>
      )}
      <div className="encoder-info">
        <Icon name="info" />
        <div>
          <strong>CPU 软件编码</strong>
          <span>使用设备上的可用编码器，保持兼容与可控。</span>
        </div>
      </div>
    </section>
  );
}

export function AdvancedSettings({
  settings,
  update,
}: {
  settings: Settings;
  update: UpdateSettings;
}) {
  const [rawText, setRawText] = useState(() =>
    settings.rawArguments.join("\n"),
  );
  return (
    <details className="surface advanced-card">
      <summary>
        <div>
          <Icon name="settings" />
          <span>高级设置</span>
        </div>
        <span>
          CRF · Preset · Raw argv
          <Icon name="chevron" />
        </span>
      </summary>
      <div className="advanced-content">
        <div className="field-grid">
          <label className="field-label">
            CRF
            <input
              type="number"
              min="0"
              max={settings.codec === "av1" ? 63 : 51}
              value={settings.quality}
              onChange={(event) =>
                update("quality", Number(event.target.value))
              }
            />
            <span className="field-hint">
              仅质量模式使用。数值越低，质量越高。
            </span>
          </label>
          <MaterialSelect
            label="编码速度 · Preset"
            value={settings.preset}
            options={presets}
            onChange={(value) => update("preset", value)}
          />
        </div>
        <label className="field-label">
          Raw FFmpeg Arguments
          <textarea
            rows={5}
            placeholder={"-metadata\ntitle=我的视频"}
            value={rawText}
            onChange={(event) => {
              setRawText(event.target.value);
              update("rawArguments", parseRawArguments(event.target.value));
            }}
            spellCheck={false}
          />
          <span className="field-hint">
            每行一个独立 argv 参数，以选项和值成对填写。支持
            -metadata、-maxrate、-bufsize
            等编码选项；输入、输出、映射和进度参数由 Core 管理。
          </span>
        </label>
      </div>
    </details>
  );
}
