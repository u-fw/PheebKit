import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { Capabilities, ExecutionPlan, Media, Stream } from "./bridge";
import {
  defaultSettings,
  targetSizeAudioError,
  validateSettings,
} from "./project";
import { EncodingSettings } from "./components/EncodingSettings";
import { OutputSummary } from "./components/OutputSummary";
import { MediaCard } from "./components/MediaCard";

vi.mock("./bridge", () => ({ API: { getThumbnail: vi.fn() } }));

const video: Stream = {
  index: 0,
  codecType: "video",
  codecName: "h264",
  width: 2560,
  height: 1440,
  frameRate: 60.0006823,
  pixelFormat: "yuv420p",
  bitDepth: 8,
  colorSpace: "bt709",
  colorTransfer: "bt709",
  colorPrimaries: "bt709",
  hdr: "",
  language: "",
  title: "",
  attachedPicture: false,
  timedThumbnails: false,
};
const media: Media = {
  source: {
    uri: "C:\\媒体\\sample.mp4",
    displayName: "sample.mp4",
    kind: "local-file",
  },
  name: "sample.mp4",
  container: "mov,mp4,m4a,3gp,3g2,mj2",
  sizeBytes: 523_940_000,
  durationSeconds: 143,
  bitrate: 30_600_000,
  streams: [
    video,
    { ...video, index: 1, codecType: "audio", codecName: "aac" },
  ],
};
const sizeSettings = { ...defaultSettings, mode: "size" };
const capabilities: Capabilities = {
  version: "ffmpeg test",
  encoders: ["libx265", "hevc_nvenc"],
  codecs: [],
  gpus: [{ name: "Test GPU", vendor: "NVIDIA" }],
  videoEncoders: [
    {
      name: "libx265",
      codec: "hevc",
      kind: "cpu",
      label: "HEVC 软件编码",
      available: true,
      reason: "",
      pixelFormats: [],
    },
    {
      name: "hevc_nvenc",
      codec: "hevc",
      kind: "gpu",
      label: "NVIDIA NVENC",
      available: true,
      reason: "实际编码测试通过",
      pixelFormats: [],
    },
  ],
};
const plan: ExecutionPlan = {
  jobID: "job",
  projectID: "project",
  input: media.source,
  output: { ...media.source, uri: "out.mp4" },
  args: [],
  preview: "",
  durationSeconds: 143,
  inputSizeBytes: media.sizeBytes,
  encoder: "hevc_nvenc",
  encoderKind: "gpu",
  encoderLabel: "NVIDIA NVENC",
  encoderReason: "实际编码测试通过",
};

describe("目标体积交互回归", () => {
  it("在含音频的源文件中提前解释 Copy 的预算限制，并允许 AAC 和静音视频", () => {
    expect(
      validateSettings(sizeSettings, "out.mp4", media.source.uri, media),
    ).toEqual([targetSizeAudioError(sizeSettings, media)]);
    expect(
      validateSettings(
        { ...sizeSettings, audio: "aac" },
        "out.mp4",
        media.source.uri,
        media,
      ),
    ).toEqual([]);
    expect(
      validateSettings(sizeSettings, "out.mp4", media.source.uri, {
        ...media,
        streams: [video],
      }),
    ).toEqual([]);
    expect(
      validateSettings(defaultSettings, "out.mp4", media.source.uri, media),
    ).toEqual([]);
  });
  it("未知时长无法计算目标体积，质量模式仍然可配置", () => {
    const unknown = { ...media, durationSeconds: 0 };
    expect(
      validateSettings(
        { ...sizeSettings, audio: "aac" },
        "out.mp4",
        media.source.uri,
        unknown,
      ),
    ).toContain("目标体积需要已知的视频时长，请换用质量优先模式。");
    expect(
      validateSettings(defaultSettings, "out.mp4", media.source.uri, unknown),
    ).toEqual([]);
  });
  it("错误与音频选择语义关联，并提供明确的 AAC 操作和 MB 换算", () => {
    const markup = renderToStaticMarkup(
      <EncodingSettings
        media={media}
        settings={sizeSettings}
        update={() => undefined}
        capabilities={capabilities}
        customResolution={false}
        setCustomResolution={() => undefined}
        plan={null}
      />,
    );
    expect(markup).toContain('aria-describedby="target-audio-error"');
    expect(markup).toContain('aria-invalid="true"');
    expect(markup).toContain("切换为 AAC");
    expect(markup).toContain("100 MB ≈ 95.37 MiB");
    expect(markup).toContain("Copy");
    expect(markup).toContain("自动 · 优先 GPU");
  });
  it("后端方案失败显示在开始按钮前，并展示方案实际使用的 GPU", () => {
    const markup = renderToStaticMarkup(
      <OutputSummary
        media={media}
        settings={defaultSettings}
        hasVideo
        encoderAvailable
        capabilitiesLoaded
        validation={[]}
        actionError=""
        previewError="目标目录不可写"
        previewLoading={false}
        plan={plan}
        connected
        starting={false}
        canStart={false}
        start={() => undefined}
      />,
    );
    expect(markup.indexOf("目标目录不可写")).toBeLessThan(
      markup.indexOf("开始压缩"),
    );
    expect(markup).toContain("GPU · NVIDIA NVENC");
    expect(markup).toContain('disabled=""');
  });
});

describe("媒体卡片信息", () => {
  it("MP4 不再误标 MOV，帧率保留可读精度与原始提示", () => {
    const markup = renderToStaticMarkup(<MediaCard media={media} compact />);
    expect(markup).toContain("MP4");
    expect(markup).not.toContain(">MOV");
    expect(markup).toContain("60 FPS");
    expect(markup).toContain('title="60.0006823 FPS"');
  });
});
