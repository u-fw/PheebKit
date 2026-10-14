import type { Media, Settings } from "./bridge";
import { defaultSettings } from "./project";
export const scenarioPresets = [
  {
    id: "compatible",
    name: "兼容播放",
    description:
      "H.264 · 质量 23 · AAC 192 kbps，最高 1080p，便于常见设备播放。",
  },
  {
    id: "share",
    name: "轻量分享",
    description: "HEVC · 质量 28 · AAC 96 kbps，最高 720p；播放端需支持 HEVC。",
  },
  {
    id: "quality",
    name: "画质优先",
    description:
      "HEVC · 质量 20 · Slow · AAC 192 kbps，保持分辨率；有损编码，文件可能增大。",
  },
  {
    id: "size",
    name: "自定体积",
    description:
      "H.264 · 50 MB 起点 · AAC 96 kbps，最高 720p；实际体积是估算值，可继续修改。",
  },
];
export function buildScenario(id: string, media: Media): Settings {
  const video = media.streams.find(
    (s) => s.codecType === "video" && !s.attachedPicture && !s.timedThumbnails,
  );
  const height =
    video && Math.abs(video.displayRotation || 0) % 180 === 90
      ? video.width
      : (video?.height ?? 0);
  const capped = (cap: number) => (height > cap ? cap : 0);
  const base: Settings = {
    ...defaultSettings,
    operation: "compress",
    audio: "aac",
    rawArguments: [],
    audioBitrateKbps: 192,
  };
  switch (id) {
    case "compatible":
      return {
        ...base,
        codec: "h264",
        quality: 23,
        preset: "fast",
        resolution: capped(1080),
      };
    case "share":
      return {
        ...base,
        codec: "hevc",
        quality: 28,
        resolution: capped(720),
        audioBitrateKbps: 96,
      };
    case "quality":
      return {
        ...base,
        codec: "hevc",
        quality: 20,
        preset: "slow",
        resolution: 0,
      };
    case "size":
      return {
        ...base,
        codec: "h264",
        mode: "size",
        targetSizeMB: 50,
        resolution: capped(720),
        audioBitrateKbps: 96,
      };
    default:
      throw new Error("场景预设不存在");
  }
}
