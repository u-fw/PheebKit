import { useState } from "react";
import type { Capabilities, ExecutionPlan, Media, Settings } from "../bridge";
import { parseRawArguments, targetSizeAudioError } from "../project";
import { formatBytes } from "../format";
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
  media,
  settings,
  update,
  capabilities,
  customResolution,
  setCustomResolution,
  plan,
}: {
  media: Media;
  settings: Settings;
  update: UpdateSettings;
  capabilities: Capabilities | null;
  customResolution: boolean;
  setCustomResolution: (value: boolean) => void;
  plan: ExecutionPlan | null;
}) {
  const audioError = targetSizeAudioError(settings, media);
  const encoderOptions = [
    {
      value: "auto",
      label: "自动 · 优先 GPU",
      description: "通过实际编码测试后选用；不可用时使用 CPU",
    },
    { value: "cpu", label: "CPU · 软件编码", description: "使用软件编码器" },
    ...(capabilities?.videoEncoders
      .filter(
        (encoder) => encoder.codec === settings.codec && encoder.kind === "gpu",
      )
      .map((encoder) => ({
        value: encoder.name,
        label: encoder.label,
        description: encoder.available
          ? "GPU 编码测试通过"
          : encoder.reason || "当前设备不可用",
        disabled: !encoder.available,
      })) ?? []),
  ];
  return (
    <section className="surface settings-card">
      <div className="section-title">
        <h2>输出设置</h2>
      </div>
      <fieldset>
        <legend>视频编码</legend>
        <div className="codec-options">
          {codecOptions.map((codec) => {
            const available = capabilities?.videoEncoders.some(
              (entry) => entry.codec === codec.value && entry.available,
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
        <legend>压缩模式</legend>
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
          <legend>画质</legend>
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
            : "目标文件大小 (MB，十进制)"}
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
              ? `1 MB = 1,000,000 字节。${Number.isFinite(settings.targetSizeMB) && settings.targetSizeMB > 0 ? `${settings.targetSizeMB} MB ≈ ${formatBytes(settings.targetSizeMB * 1_000_000)}。` : ""}按时长分配视频和音频码率，实际大小可能有差异。`
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
          options={audioOptions.map((option) =>
            option.value === "copy" &&
            settings.trim &&
            media.streams.some((stream) => stream.codecType === "audio")
              ? {
                  ...option,
                  disabled: true,
                  description: "剪辑时请选择 AAC 或 Opus",
                }
              : option,
          )}
          onChange={(value) => update("audio", value)}
          describedBy={audioError ? "target-audio-error" : undefined}
          invalid={Boolean(audioError)}
        />
      </div>
      {audioError && (
        <div
          className="field-notice error-notice"
          id="target-audio-error"
          role="alert"
        >
          <Icon name="info" />
          <div>
            <p>{audioError}</p>
            <button
              type="button"
              className="button outlined"
              onClick={() => update("audio", "aac")}
            >
              切换为 AAC
            </button>
          </div>
        </div>
      )}
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
      <div className="processing-field">
        <MaterialSelect
          label="处理方式"
          value={settings.encoder}
          options={encoderOptions}
          onChange={(value) => update("encoder", value)}
          disabled={!capabilities}
        />
      </div>
      <div className="encoder-info" aria-live="polite">
        <Icon name="info" />
        <div>
          <strong>
            {plan
              ? `${plan.encoderKind === "gpu" ? "GPU" : "CPU"} · ${plan.encoderLabel}`
              : settings.encoder === "auto"
                ? "自动选择可用编码器"
                : settings.encoder === "cpu"
                  ? "CPU 软件编码"
                  : "指定 GPU 编码器"}
          </strong>
          <span>
            {plan?.encoderReason ||
              (capabilities
                ? "以完成设备检测后的执行方案为准。"
                : "正在检测 FFmpeg 编码器和设备支持…")}
          </span>
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
          质量 · 速度 · 更多参数
          <Icon name="chevron" />
        </span>
      </summary>
      <div className="advanced-content">
        <div className="field-grid">
          <label className="field-label">
            {settings.encoder === "cpu" ? "质量参数 · CRF" : "质量参数"}
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
              {settings.encoder === "cpu"
                ? "仅质量模式使用 CRF。数值越低，质量越高。"
                : "仅质量模式使用。CPU 使用 CRF，GPU 使用对应编码器的质量参数；数值越低，质量越高。"}
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
          附加 FFmpeg 参数
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
            -metadata、-maxrate、-bufsize 等编码选项。编码器子选项使用
            key=value，以冒号分隔；仅支持已知安全选项，文件读写与多遍编码选项不可使用。
          </span>
        </label>
      </div>
    </details>
  );
}
