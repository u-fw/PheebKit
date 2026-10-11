import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { Media, Stream } from "./bridge";
import {
  changeOperation,
  applyPresetSettings,
  defaultSettings,
  operationOf,
  operationExtensions,
  selectedTracks,
  suggestedOperationOutput,
  settingsForMedia,
  videoDisplaySize,
  validateSettings,
} from "./project";
import { TrackSettings } from "./components/TrackSettings";
import { TransformSettings } from "./components/TransformSettings";
import { OutputSummary } from "./components/OutputSummary";
vi.mock("./bridge", () => ({ API: {} }));

const video: Stream = {
  index: 0,
  codecType: "video",
  codecName: "h264",
  width: 1920,
  height: 1080,
  frameRate: 30,
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
    uri: "C:\\素材\\video.mkv",
    displayName: "video.mkv",
    kind: "local-file",
  },
  name: "video.mkv",
  container: "mkv",
  sizeBytes: 10000,
  durationSeconds: 60,
  bitrate: 1000,
  streams: [
    video,
    {
      ...video,
      index: 2,
      codecType: "audio",
      codecName: "aac",
      language: "eng",
      title: "Commentary",
    },
    {
      ...video,
      index: 7,
      codecType: "audio",
      codecName: "opus",
      language: "zho",
      title: "原声",
    },
    {
      ...video,
      index: 9,
      codecType: "subtitle",
      codecName: "ass",
      language: "zho",
      title: "中文对白",
    },
  ],
};

describe("processing operation safety", () => {
  it("checks source crop in the displayed orientation for rotated phone footage", () => {
    expect(
      videoDisplaySize({ width: 1920, height: 1080, displayRotation: 90 }),
    ).toEqual({ width: 1080, height: 1920 });
    expect(
      videoDisplaySize({ width: 1920, height: 1080, displayRotation: -90 }),
    ).toEqual({ width: 1080, height: 1920 });
    expect(
      videoDisplaySize({ width: 1920, height: 1080, displayRotation: 180 }),
    ).toEqual({ width: 1920, height: 1080 });
  });
  it("keeps old project defaults, all audio and no subtitle", () => {
    expect(operationOf(defaultSettings)).toBe("compress");
    expect(
      selectedTracks(media, "audio", defaultSettings).map(
        (stream) => stream.index,
      ),
    ).toEqual([2, 7]);
    expect(selectedTracks(media, "subtitle", defaultSettings)).toEqual([]);
    expect(
      selectedTracks(media, "audio", { ...defaultSettings, audio: "none" }),
    ).toEqual([]);
  });
  it("switches to extraction with a real global index and clears incompatible edits", () => {
    const edited = {
      ...defaultSettings,
      trim: { startSeconds: 3, endSeconds: 12 },
      transform: { rotation: 90, flipHorizontal: true, flipVertical: false },
      volumeDB: 3,
      rawArguments: ["-metadata", "title=Demo"],
    };
    const audio = changeOperation(edited, "extract-audio", media);
    expect(audio.audioSelection).toEqual({ mode: "selected", indices: [2] });
    expect(audio.trim).toBeNull();
    expect(audio.transform).toBeNull();
    expect(audio.volumeDB).toBe(0);
    expect(audio.rawArguments).toEqual([]);
    expect(suggestedOperationOutput(media.source.uri, audio)).toBe(
      "C:\\素材\\video-audio.mka",
    );
    const subtitle = changeOperation(audio, "extract-subtitle", media);
    expect(subtitle.subtitleSelection?.indices).toEqual([9]);
    expect(suggestedOperationOutput(media.source.uri, subtitle)).toBe(
      "C:\\素材\\video-subtitle.srt",
    );
    expect(changeOperation(edited, "remux", media).audio).toBe("copy");
  });
  it("selects only valid containers for codecs and independent audio exports", () => {
    expect(operationExtensions({ ...defaultSettings, audio: "opus" })).toEqual([
      "mkv",
    ]);
    expect(
      operationExtensions({ ...defaultSettings, codec: "av1", audio: "opus" }),
    ).toEqual(["mkv", "webm"]);
    for (const [audio, extension] of [
      ["aac", "m4a"],
      ["opus", "opus"],
      ["mp3", "mp3"],
      ["flac", "flac"],
      ["wav", "wav"],
    ])
      expect(
        operationExtensions({
          ...defaultSettings,
          operation: "extract-audio",
          audio,
        }),
      ).toEqual([extension]);
  });
  it("binds saved extraction defaults to current media and keeps explicitly selected tracks", () => {
    const generic = {
      ...defaultSettings,
      operation: "extract-audio",
      audio: "aac",
    };
    expect(settingsForMedia(generic, media).audioSelection?.indices).toEqual([
      2,
    ]);
    const explicit = {
      ...generic,
      audioSelection: { mode: "selected", indices: [7] },
    };
    expect(settingsForMedia(explicit, media).audioSelection?.indices).toEqual([
      7,
    ]);
    const preset = { ...generic, volumeDB: 3 };
    const applied = applyPresetSettings(
      {
        ...defaultSettings,
        trim: { startSeconds: 2, endSeconds: 9 },
        resolution: 720,
      },
      preset,
      media,
    );
    expect(applied.audioSelection?.indices).toEqual([2]);
    expect(applied.subtitleSelection?.mode).toBe("none");
    expect(applied.volumeDB).toBe(3);
    expect(applied.trim).toBeNull();
    expect(applied.resolution).toBe(0);
  });
  it("retains clip range, source crop and tracks when applying a configuration for the same task", () => {
    const current = {
      ...defaultSettings,
      trim: { startSeconds: 2, endSeconds: 9 },
      audioSelection: { mode: "selected", indices: [7] },
      transform: {
        rotation: 0,
        flipHorizontal: false,
        flipVertical: false,
        crop: { top: 2, bottom: 2, left: 2, right: 2 },
      },
    };
    const applied = applyPresetSettings(
      current,
      {
        ...defaultSettings,
        audio: "aac",
        transform: { rotation: 90, flipHorizontal: false, flipVertical: true },
      },
      media,
    );
    expect(applied.trim).toEqual(current.trim);
    expect(applied.audioSelection).toEqual(current.audioSelection);
    expect(applied.transform?.crop).toEqual(current.transform.crop);
    expect(applied.transform?.rotation).toBe(90);
    expect(applied.transform?.flipVertical).toBe(true);
  });
  it("does not apply irrelevant compression budgets to remux or extraction", () => {
    const remux = {
      ...defaultSettings,
      operation: "remux",
      quality: -1,
      resolution: 0,
      mode: "size",
      targetSizeMB: 0,
    };
    expect(validateSettings(remux, "out.mkv", media.source.uri, media)).toEqual(
      [],
    );
    expect(
      validateSettings(
        { ...remux, trim: { startSeconds: 1, endSeconds: 5 } },
        "out.mkv",
        media.source.uri,
        media,
      ),
    ).toContain("此任务保持原始内容，请先清除剪辑、画面调整、音量和附加参数。");
  });
  it("honors portable all/none track policies over current selected stream indexes", () => {
    const current = {
      ...defaultSettings,
      audioSelection: { mode: "selected", indices: [7] },
      subtitleSelection: { mode: "selected", indices: [9] },
    };
    const preset = {
      ...defaultSettings,
      audioSelection: { mode: "none", indices: [] },
      subtitleSelection: { mode: "all", indices: [] },
    };
    const next = applyPresetSettings(current, preset, media);
    expect(next.audioSelection).toEqual(preset.audioSelection);
    expect(next.subtitleSelection).toEqual(preset.subtitleSelection);
    expect(selectedTracks(media, "audio", next)).toEqual([]);
    expect(
      selectedTracks(media, "subtitle", next).map((stream) => stream.index),
    ).toEqual([9]);
  });
  it("catches foreign or empty selections, copy volume, crop bounds and source overwrite", () => {
    expect(
      validateSettings(
        {
          ...defaultSettings,
          audioSelection: { mode: "selected", indices: [0] },
        },
        "out.mp4",
        media.source.uri,
        media,
      ).join(" "),
    ).toContain("不属于当前媒体");
    expect(
      validateSettings(
        {
          ...defaultSettings,
          operation: "extract-audio",
          audioSelection: { mode: "selected", indices: [] },
        },
        "out.mka",
        media.source.uri,
        media,
      ).join(" "),
    ).toContain("提取音频需要选择一条音轨");
    expect(
      validateSettings(
        { ...defaultSettings, volumeDB: 3 },
        "out.mp4",
        media.source.uri,
        media,
      ).join(" "),
    ).toContain("重新编码音频");
    expect(
      validateSettings(
        {
          ...defaultSettings,
          audioSelection: { mode: "none", indices: [] },
          volumeDB: 3,
        },
        "out.mp4",
        media.source.uri,
        media,
      ),
    ).toEqual([]);
    expect(
      validateSettings(
        {
          ...defaultSettings,
          transform: {
            rotation: 0,
            flipHorizontal: false,
            flipVertical: false,
            crop: { top: 1, bottom: 0, left: 1920, right: 0 },
          },
        },
        "out.mp4",
        media.source.uri,
        media,
      ),
    ).toContain("裁剪后宽度和高度至少需要保留 2 像素。");
    expect(
      validateSettings(
        defaultSettings,
        "c:/素材/VIDEO.mkv",
        media.source.uri,
        media,
      ),
    ).toContain("输出文件不能覆盖源文件，请选择新路径。");
  });
});

describe("processing interface", () => {
  it("labels selected global tracks with language, codec and title", () => {
    const html = renderToStaticMarkup(
      <TrackSettings
        media={media}
        settings={{
          ...defaultSettings,
          audioSelection: { mode: "selected", indices: [7] },
        }}
        update={() => {}}
      />,
    );
    expect(html).toContain("音轨 2 · 中文 · OPUS · 原声");
    expect(html).toContain('type="checkbox"');
    expect(html).toContain('checked=""');
    expect(html).toContain("移除字幕");
    expect(html).toContain("不会把字幕烧录到画面");
  });
  it("gives volume copy recovery and explains original-only preview", () => {
    const html = renderToStaticMarkup(
      <TransformSettings
        media={media}
        settings={{
          ...defaultSettings,
          volumeDB: 2,
          transform: {
            rotation: 0,
            flipHorizontal: false,
            flipVertical: false,
            crop: { top: 0, bottom: 0, left: 0, right: 0 },
          },
        }}
        update={() => {}}
      />,
    );
    expect(html).toContain("切换为 AAC");
    expect(html).toContain("预览仍展示原片");
    expect(html).toContain("上边距 (px)");
    expect(html).toContain('step="2"');
  });
  it("does not advertise compression savings or GPU for remux", () => {
    const html = renderToStaticMarkup(
      <OutputSummary
        media={media}
        settings={{ ...defaultSettings, operation: "remux", mode: "size" }}
        hasVideo={false}
        encoderAvailable={false}
        capabilitiesLoaded={false}
        validation={[]}
        actionError=""
        previewError=""
        previewLoading={false}
        plan={null}
        connected
        starting={false}
        canStart
        start={() => {}}
      />,
    );
    expect(html).toContain("开始转换封装");
    expect(html).toContain("直接复制轨道");
    expect(html).not.toContain("节省空间");
    expect(html).not.toContain("所选编码方式");
    expect(html).not.toContain("正在检测设备");
    expect(html).not.toContain("GPU");
  });
});
