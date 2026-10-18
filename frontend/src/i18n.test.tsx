import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { t, setLanguage, getLanguage } from "./i18n";
import { MediaCard } from "./components/MediaCard";
import { MaterialSelect } from "./components/MaterialSelect";
import { SettingsPage } from "./components/SettingsPage";
import { trackLabel } from "./components/TrackSettings";
import { TrimSettings } from "./components/TrimSettings";
import { defaultSettings } from "./project";
import type { Media } from "./bridge";
import catalog from "../../locales/en.json";

let saved: Map<string, string>;
beforeEach(() => {
  saved = new Map();
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => saved.get(key) ?? null,
    setItem: (key: string, value: string) => saved.set(key, value),
  });
  setLanguage("zh-CN");
});
afterEach(() => {
  setLanguage("zh-CN");
  vi.unstubAllGlobals();
});

const action = async () => undefined;
function settings() {
  return renderToStaticMarkup(
    <SettingsPage
      platform="windows"
      connected
      capabilities={null}
      capabilityError=""
      environment={null}
      environmentBusy={false}
      refreshEnvironment={action}
      chooseFFmpeg={action}
      configureFFmpeg={action}
      ffmpegSetup={null}
      startFFmpegSetup={action}
      startCustomFFmpegSetup={action}
      cancelFFmpegSetup={action}
      theme="light"
      toggleTheme={() => undefined}
    />,
  );
}

describe("application localization", () => {
  it("changes both directions and persists the validated preference", () => {
    expect(t("任务队列")).toBe("任务队列");
    setLanguage("en-US");
    expect(t("任务队列")).toBe("Task queue");
    expect(saved.get("pheebkit-language")).toBe("en-US");
    setLanguage("zh-CN");
    expect(t("任务队列")).toBe("任务队列");
    expect(() => setLanguage("invalid" as "en-US")).toThrow(
      "Unsupported language",
    );
    expect(getLanguage()).toBe("zh-CN");
  });
  it("keeps parameters literal and translates native application errors", () => {
    setLanguage("en-US");
    const path = "C:/用户/设置 & <视频>.mp4";
    expect(t("项目已保存：{{0}}", [path])).toBe(`Project saved: ${path}`);
    expect(t(`源媒体文件不存在或不是普通文件，请重新导入。`)).toBe(
      "Source media is missing or not a regular file. Import it again.",
    );
    expect(t(`ffmpeg 路径必须是本机绝对路径。`)).toBe(
      "ffmpeg must use a local absolute path.",
    );
    expect(t("constructor")).toBe("constructor");
    expect(t("__proto__")).toBe("__proto__");
    expect(t("原始 FFmpeg 未知输出")).toBe("原始 FFmpeg 未知输出");
  });
  it("renders English settings and reports unavailable preference storage without blocking switching", () => {
    setLanguage("en-US");
    expect(settings()).toContain("Interface language");
    expect(settings()).toContain("GPU and encoders");
    vi.stubGlobal("localStorage", {
      setItem: () => {
        throw new Error("storage denied");
      },
    });
    setLanguage("zh-CN");
    expect(settings()).toContain("当前环境无法保存本机偏好");
    setLanguage("en-US");
    expect(settings()).toContain("cannot save local preferences");
  });
  it("preserves source filenames and user-owned option labels even when they match a translation key", () => {
    setLanguage("en-US");
    const media: Media = {
      source: { kind: "local-file", uri: "C:/测试/设置", displayName: "设置" },
      name: "设置",
      container: "mov",
      durationSeconds: 8,
      sizeBytes: 1000,
      bitrate: 1000,
      streams: [],
    };
    const before = JSON.stringify(media);
    const html = renderToStaticMarkup(<MediaCard media={media} />);
    expect(html).toContain('title="设置"');
    expect(html).toContain("Source media");
    expect(JSON.stringify(media)).toBe(before);
    const trim = renderToStaticMarkup(
      <TrimSettings
        media={media}
        settings={defaultSettings}
        onChange={() => undefined}
      />,
    );
    expect(trim).toContain(
      "Keep the full duration and compress using the settings below.",
    );
    expect(trim).not.toContain("保留完整时长");
    expect(
      trackLabel(
        {
          index: 0,
          codecType: "audio",
          codecName: "aac",
          language: "chi",
          title: "设置",
        } as Media["streams"][number],
        0,
      ),
    ).toBe("Audio track 1 · Chinese · AAC · 设置");
    const select = renderToStaticMarkup(
      <MaterialSelect
        label="已保存配置"
        value="user"
        options={[{ value: "user", label: "设置", translate: false }]}
        onChange={() => undefined}
      />,
    );
    expect(select).toContain("Saved presets");
    expect(select).toContain(">设置<");
    expect(select).not.toContain('aria-label=""');
  });
  it("has complete, nonempty translations with identical parameter identities", () => {
    const placeholders = (value: string) =>
      [...value.matchAll(/\{\{\d+\}\}/g)].map((match) => match[0]).sort();
    for (const [source, target] of Object.entries(catalog)) {
      expect(target.trim(), source).not.toBe("");
      expect(placeholders(target), source).toEqual(placeholders(source));
    }
  });
});
