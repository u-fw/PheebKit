import { useState } from "react";
import { API } from "../bridge";
import type { Capabilities, Job, Media, Settings } from "../bridge";
import { errorMessage, suggestedOutput } from "../format";
import { defaultSettings, makeProject, validateSettings } from "../project";
import { useExecutionPlan } from "../hooks/useExecutionPlan";
import { CommandPreview } from "./CommandPreview";
import { EncodingSettings, AdvancedSettings } from "./EncodingSettings";
import { Icon } from "./Icon";
import { MediaCard } from "./MediaCard";
import { OutputSummary } from "./OutputSummary";

type Props = {
  media: Media | null;
  capabilities: Capabilities | null;
  connected: boolean;
  chooseMedia: () => void;
  onJob: (job: Job) => void;
  showQueue: () => void;
};

export function Compress({
  media,
  capabilities,
  connected,
  chooseMedia,
  onJob,
  showQueue,
}: Props) {
  const [settings, setSettings] = useState<Settings>(defaultSettings);
  const [output, setOutput] = useState(() =>
    media
      ? suggestedOutput(
          media.source.uri,
          defaultSettings.codec,
          defaultSettings.audio,
        )
      : "",
  );
  const [customResolution, setCustomResolution] = useState(false);
  const [actionError, setActionError] = useState("");
  const [starting, setStarting] = useState(false);
  const [choosingOutput, setChoosingOutput] = useState(false);
  const [projectID] = useState(() => crypto.randomUUID());
  const preview = useExecutionPlan(
    media,
    settings,
    output,
    connected,
    projectID,
    capabilities,
  );
  const validation = media
    ? validateSettings(settings, output, media.source.uri, media)
    : [];
  const encoderAvailable = Boolean(
    capabilities?.videoEncoders.some(
      (encoder) =>
        encoder.codec === settings.codec &&
        encoder.available &&
        (settings.encoder === "auto" ||
          (settings.encoder === "cpu" && encoder.kind === "cpu") ||
          settings.encoder === encoder.name),
    ),
  );
  const hasVideo = Boolean(
    media?.streams.some(
      (stream) =>
        stream.codecType === "video" &&
        !stream.attachedPicture &&
        !stream.timedThumbnails,
    ),
  );

  function update<K extends keyof Settings>(key: K, value: Settings[K]) {
    setSettings((current) => ({
      ...current,
      [key]: value,
      ...(key === "codec" && current.encoder !== "cpu"
        ? { encoder: "auto" }
        : {}),
    }));
    setActionError("");
  }

  async function chooseOutput() {
    setChoosingOutput(true);
    setActionError("");
    try {
      const chosen = await API.chooseOutput(output);
      if (chosen) setOutput(chosen);
    } catch (cause) {
      setActionError(`选择输出文件失败：${errorMessage(cause)}`);
    } finally {
      setChoosingOutput(false);
    }
  }

  async function start() {
    if (
      !media ||
      starting ||
      validation.length ||
      !encoderAvailable ||
      !hasVideo ||
      !preview.plan ||
      preview.loading
    )
      return;
    setStarting(true);
    setActionError("");
    try {
      const job = await API.start(
        makeProject(media, settings, output, projectID),
      );
      onJob(job);
      showQueue();
    } catch (cause) {
      setActionError(`启动失败：${errorMessage(cause)}`);
    } finally {
      setStarting(false);
    }
  }

  if (!media)
    return (
      <div className="page">
        <header className="page-heading">
          <div>
            <div className="section-eyebrow">本地媒体处理</div>
            <h1>压缩视频</h1>
            <p>导入一个视频，开始配置压缩任务。</p>
          </div>
        </header>
        <section className="empty-state surface">
          <Icon name="compress" />
          <h2>先选择源媒体</h2>
          <p>压缩设置与执行参数会根据媒体信息生成。</p>
          <button
            className="button primary"
            disabled={!connected}
            onClick={chooseMedia}
          >
            <Icon name="plus" />
            选择媒体文件
          </button>
        </section>
      </div>
    );

  return (
    <div className="page compress-page">
      <header className="page-heading">
        <div>
          <div className="section-eyebrow">本地媒体处理</div>
          <h1>压缩视频</h1>
          <p>调整输出格式、处理方式和文件大小。</p>
        </div>
        <span className="chip accent-chip">
          <Icon name="video" />
          本地处理
        </span>
      </header>
      <MediaCard media={media} compact />
      <div className="compress-layout">
        <div className="settings-column">
          <EncodingSettings
            media={media}
            settings={settings}
            update={update}
            capabilities={capabilities}
            customResolution={customResolution}
            setCustomResolution={setCustomResolution}
            plan={preview.plan}
          />
          <AdvancedSettings settings={settings} update={update} />
          <section className="surface output-card">
            <div className="section-title">
              <h2>保存到</h2>
              <Icon name="folder" />
            </div>
            <label className="field-label" htmlFor="output-path">
              输出文件
            </label>
            <div className="output-field">
              <input
                id="output-path"
                value={output}
                onChange={(event) => {
                  setOutput(event.target.value);
                  setActionError("");
                }}
                spellCheck={false}
              />
              <button
                className="button outlined"
                onClick={() => void chooseOutput()}
                disabled={choosingOutput || !connected}
              >
                {choosingOutput ? "选择中…" : "浏览"}
              </button>
            </div>
            <span className="field-hint">使用新的输出文件名，保留源媒体。</span>
          </section>
        </div>
        <aside className="preview-column">
          <OutputSummary
            media={media}
            settings={settings}
            hasVideo={hasVideo}
            encoderAvailable={encoderAvailable}
            capabilitiesLoaded={capabilities !== null}
            validation={validation}
            actionError={actionError}
            previewError={preview.error}
            previewLoading={preview.loading}
            plan={preview.plan}
            connected={connected}
            starting={starting}
            canStart={
              connected &&
              Boolean(preview.plan) &&
              !preview.loading &&
              !starting &&
              encoderAvailable &&
              hasVideo &&
              validation.length === 0
            }
            start={() => void start()}
          />
          <CommandPreview
            plan={preview.plan}
            loading={preview.loading}
            error={preview.error}
          />
        </aside>
      </div>
    </div>
  );
}
