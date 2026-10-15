import { describe, it, expect, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import {
  audioProcessingErrors,
  audioProcessingSummary,
  defaultAudioProcessing,
} from "./audioProcessing";
import {
  defaultSettings,
  changeOperation,
  applyPresetSettings,
} from "./project";
import { makeBatchProjects } from "./batch";
import { AudioProcessingSettings } from "./components/AudioProcessingSettings";
import type { Media, Settings } from "./bridge";
vi.mock("./bridge", () => ({ API: {} }));
const media: Media = {
  source: { uri: "C:/in.mp4", kind: "local-file", displayName: "in.mp4" },
  name: "in.mp4",
  container: "mp4",
  durationSeconds: 10,
  sizeBytes: 1000,
  bitrate: 500,
  streams: [
    {
      index: 0,
      codecType: "audio",
      codecName: "aac",
      sampleRate: 44100,
      channels: 2,
      width: 0,
      height: 0,
      frameRate: 0,
      pixelFormat: "",
      bitDepth: 0,
      colorSpace: "",
      colorTransfer: "",
      colorPrimaries: "",
      hdr: "",
      language: "",
      title: "",
      attachedPicture: false,
      timedThumbnails: false,
    },
  ],
};
function settings(): Settings {
  return {
    ...defaultSettings,
    audio: "aac",
    audioProcessing: defaultAudioProcessing(),
  };
}
describe("typed audio processing", () => {
  it("leaves existing projects disabled and clearly exposes rate/layout controls on opt-in", () => {
    expect(audioProcessingErrors(defaultSettings, 10, 1)).toEqual([]);
    const html = renderToStaticMarkup(
      <AudioProcessingSettings
        media={media}
        settings={settings()}
        capabilities={null}
        update={() => {}}
      />,
    );
    expect(html).toContain("启用声音处理");
    expect(html).toContain("降噪强度 (dB)");
    expect(html).toContain("参数保持可编辑");
    expect(html).toContain("输出采样率");
    expect(html).toContain("保留源声道");
    expect(html).toContain("样片不应用整片淡入淡出");
    expect(html).toContain("44.1 kHz");
  });
  it("rejects unsafe/incompatible ranges and exposes missing filters", () => {
    const s = settings();
    s.audio = "copy";
    s.audioProcessing!.fadeInSeconds = 8;
    s.audioProcessing!.fadeOutSeconds = 3;
    s.audioProcessing!.channels = "shell";
    expect(audioProcessingErrors(s, 10, 0).join(" ")).toContain("重新编码");
    expect(audioProcessingErrors(s, 10, 0).join(" ")).toContain("总时长");
    const html = renderToStaticMarkup(
      <AudioProcessingSettings
        media={media}
        settings={s}
        capabilities={{
          version: "fixture",
          encoders: [],
          codecs: [],
          gpus: [],
          videoEncoders: [],
          filters: [],
        }}
        update={() => {}}
      />,
    );
    expect(html).toContain("aresample");
    expect(html).toContain("切换到可处理的音频编码");
  });
  it("validates EQ Nyquist, finite values and codec sample-rate rules", () => {
    const s = settings();
    s.audioProcessing!.sampleRate = 8000;
    s.audioProcessing!.equalizer = [{ frequencyHz: 5000, q: 1, gainDB: 2 }];
    expect(audioProcessingErrors(s).join(" ")).toContain("一半");
    s.audioProcessing!.equalizer[0].gainDB = 0;
    expect(audioProcessingErrors(s)).toEqual([]);
    s.audioProcessing!.fadeInSeconds = NaN;
    expect(audioProcessingErrors(s).join(" ")).toContain("0–600");
    s.audioProcessing!.fadeInSeconds = 0;
    s.audio = "opus";
    expect(audioProcessingErrors(s).join(" ")).toContain("48000");
  });
  it("preserves portable controls, isolates batch settings, and clears them on operation changes", () => {
    const s = settings();
    s.audioProcessing!.equalizer = [{ frequencyHz: 1000, q: 1, gainDB: -3 }];
    expect(
      applyPresetSettings(defaultSettings, s, media).audioProcessing,
    ).toEqual(s.audioProcessing);
    const batch = makeBatchProjects(
      [media, { ...media, source: { ...media.source, uri: "C:/other.mp4" } }],
      s,
      "C:/out",
      "-done",
    );
    batch[0].settings.audioProcessing!.equalizer![0].gainDB = 9;
    expect(batch[1].settings.audioProcessing!.equalizer![0].gainDB).toBe(-3);
    expect(s.audioProcessing!.equalizer![0].gainDB).toBe(-3);
    expect(changeOperation(s, "remux", media).audioProcessing).toBeNull();
  });
  it("summarizes actual selected controls without claiming loudness compliance", () => {
    const s = settings();
    s.audioProcessing!.normalize = {
      integratedLUFS: -16,
      truePeakDBTP: -1.5,
      rangeLU: 11,
    };
    s.audioProcessing!.fadeInSeconds = 1;
    expect(audioProcessingSummary(s)).toContain("动态响度");
    expect(audioProcessingSummary(s)).toContain("48 kHz");
    const html = renderToStaticMarkup(
      <AudioProcessingSettings
        media={media}
        settings={s}
        capabilities={null}
        update={() => {}}
      />,
    );
    expect(html).toContain("单遍");
    expect(html).toContain("目标不等于编码后实测值");
  });
});
