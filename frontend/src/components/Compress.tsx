import { useCallback, useEffect, useState, type ReactNode } from "react";
import { draftFingerprint, inOutputDirectory, type Draft } from "../draft";
import { directoryOf } from "../batch";
import { API } from "../bridge";
import type { Capabilities, Job, Media, Project, Settings } from "../bridge";
import { errorMessage } from "../format";
import {
  changeOperation,
  applyPresetSettings,
  defaultSettings,
  makeProject,
  operationExtensions,
  operationLabels,
  operationOf,
  outputExtension,
  suggestedOperationOutput,
  settingsForMedia,
  validateSettings,
} from "../project";
import { useExecutionPlan } from "../hooks/useExecutionPlan";
import { CommandPreview } from "./CommandPreview";
import {
  EncodingSettings,
  AdvancedSettings,
  AudioEncodingSettings,
} from "./EncodingSettings";
import { Icon } from "./Icon";
import { MediaCard } from "./MediaCard";
import { OutputSummary } from "./OutputSummary";
import { TrimSettings } from "./TrimSettings";
import { SavedSettings } from "./SavedSettings";
import { OperationSettings } from "./OperationSettings";
import { TrackSettings } from "./TrackSettings";
import { TransformSettings } from "./TransformSettings";
import { MaterialSelect } from "./MaterialSelect";

type Props = {
  media: Media | null;
  capabilities: Capabilities | null;
  connected: boolean;
  chooseMedia: () => void;
  onJob: (job: Job) => void;
  showQueue: () => void;
  initialSettings?: Settings;
  reused?: boolean;
  visible?: boolean;
  onSaved?: (value: Settings) => void;
  initialOutput?: string;
  renderBatch?: (settings: Settings) => ReactNode;
  projectActions?: (
    project: Project,
    markSaved: (project: Project) => void,
  ) => ReactNode;
  onDraft?: (draft: Draft) => void;
  outputDirectory?: string;
  onOutputDirectory?: (directory: string) => void;
  onNotice?: (message: string) => void;
};

export function Compress({
  media,
  capabilities,
  connected,
  chooseMedia,
  onJob,
  showQueue,
  initialSettings,
  reused = false,
  visible = true,
  onSaved,
  initialOutput,
  renderBatch,
  projectActions,
  onDraft,
  outputDirectory,
  onOutputDirectory,
  onNotice,
}: Props) {
  const [settings, setSettings] = useState<Settings>(() =>
    media
      ? settingsForMedia(initialSettings ?? defaultSettings, media)
      : (initialSettings ?? defaultSettings),
  );
  const [output, setOutput] = useState(
    () =>
      initialOutput ??
      (media
        ? inOutputDirectory(
            suggestedOperationOutput(
              media.source.uri,
              initialSettings ?? defaultSettings,
            ).replace(/(\.[^.]+)$/, reused ? `-${Date.now()}$1` : "$1"),
            outputDirectory,
          )
        : ""),
  );
  const [customResolution, setCustomResolution] = useState(
    () =>
      ![0, 720, 1080, 1440, 2160].includes(
        (initialSettings ?? defaultSettings).resolution,
      ),
  );
  const [actionError, setActionError] = useState("");
  const [starting, setStarting] = useState(false);
  const [choosingOutput, setChoosingOutput] = useState(false);
  const [advancedRevision, setAdvancedRevision] = useState(0);
  const [settingsNotice, setSettingsNotice] = useState("");
  const [projectID] = useState(() => crypto.randomUUID());
  const [baseline, setBaseline] = useState(() =>
    media
      ? draftFingerprint(makeProject(media, settings, output, projectID))
      : "",
  );
  const markSaved = useCallback(
    (project: Project) => setBaseline(draftFingerprint(project)),
    [],
  );
  useEffect(() => {
    if (!media || !onDraft) return;
    const project = makeProject(media, settings, output, projectID);
    onDraft({
      project,
      dirty: draftFingerprint(project) !== baseline,
      markSaved,
    });
  }, [media, settings, output, projectID, baseline, markSaved, onDraft]);
  const operation = operationOf(settings);
  const compress = operation === "compress";
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
  const encoderAvailable =
    !compress ||
    Boolean(
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
    const next = {
      ...settings,
      [key]: value,
      ...(key === "codec" && settings.encoder !== "cpu"
        ? { encoder: "auto" }
        : {}),
    };
    setSettings((current) => ({
      ...current,
      [key]: value,
      ...(key === "codec" && current.encoder !== "cpu"
        ? { encoder: "auto" }
        : {}),
    }));
    if (key === "audio" || key === "codec")
      setOutput((current) =>
        current
          ? current.replace(
              /\.[^./\\]+$/,
              `.${outputExtension(next, current.split(".").pop())}`,
            )
          : current,
      );
    setActionError("");
  }

  async function chooseOutput() {
    setChoosingOutput(true);
    setActionError("");
    try {
      const chosen = await API.chooseOutput(output);
      if (chosen) {
        setOutput(chosen);
        await API.rememberOutputDirectory(chosen);
        onOutputDirectory?.(directoryOf(chosen));
      }
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
      (compress && !hasVideo) ||
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
      void API.rememberOutputDirectory(output)
        .then(() => onOutputDirectory?.(directoryOf(output)))
        .catch((cause: unknown) => {
          const message = `任务已提交，但输出目录未能记住：${errorMessage(cause)}`;
          setActionError(message);
          onNotice?.(message);
        });
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
            <h1>媒体处理</h1>
            <p>导入媒体，选择压缩、转封装或提取轨道。</p>
          </div>
        </header>
        <section className="empty-state surface">
          <Icon name="compress" />
          <h2>先选择源媒体</h2>
          <p>处理设置与执行参数会根据媒体信息生成。</p>
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
          <h1>{operationLabels[operation] ?? "媒体处理"}</h1>
          <p>本地处理媒体，选择需要保留的内容与输出格式。</p>
        </div>
        <span className="chip accent-chip">
          <Icon name="video" />
          本地处理
        </span>
      </header>
      <MediaCard media={media} compact />
      <div className="compress-layout">
        <div className="settings-column">
          <SavedSettings
            onSaved={onSaved}
            settings={settings}
            apply={(value) => {
              const next = applyPresetSettings(settings, value, media);
              setAdvancedRevision((current) => current + 1);
              setSettingsNotice(
                operationOf(settings) === operationOf(value)
                  ? "配置已应用；当前剪辑范围保持原样，请检查轨道与输出设置。"
                  : "配置已应用，并切换任务；旧的剪辑范围和画面裁剪已清除。请检查输出设置。",
              );
              setSettings(next);
              setOutput((current) =>
                current
                  ? current.replace(
                      /\.[^./\\]+$/,
                      `.${outputExtension(next, current.split(".").pop())}`,
                    )
                  : current,
              );
              setCustomResolution(
                ![0, 720, 1080, 1440, 2160].includes(value.resolution),
              );
              setActionError("");
            }}
          />
          {settingsNotice && (
            <p className="field-hint" role="status">
              {settingsNotice}
            </p>
          )}
          {projectActions?.(
            makeProject(media, settings, output, projectID),
            markSaved,
          )}
          <OperationSettings
            media={media}
            settings={settings}
            onChange={(value) => {
              const next = changeOperation(settings, value, media);
              setSettingsNotice("");
              setAdvancedRevision((current) => current + 1);
              setSettings(next);
              const suffix = {
                compress: "compressed",
                remux: "remux",
                "extract-audio": "audio",
                "extract-subtitle": "subtitle",
              }[value];
              setOutput((current) =>
                current
                  ? current
                      .replace(
                        /-(compressed|remux|audio|subtitle)(?=\.[^./\\]+$)/,
                        `-${suffix}`,
                      )
                      .replace(/\.[^./\\]+$/, `.${outputExtension(next)}`)
                  : suggestedOperationOutput(media.source.uri, next),
              );
              setActionError("");
            }}
          />
          {compress && (
            <TrimSettings
              visible={visible}
              media={media}
              settings={settings}
              onChange={(next) => {
                setSettings(next);
                setActionError("");
              }}
            />
          )}
          {compress && (
            <EncodingSettings
              media={media}
              settings={settings}
              update={update}
              capabilities={capabilities}
              customResolution={customResolution}
              setCustomResolution={setCustomResolution}
              plan={preview.plan}
            />
          )}
          {operation === "extract-audio" && (
            <AudioEncodingSettings settings={settings} update={update} />
          )}
          <TrackSettings media={media} settings={settings} update={update} />
          {(compress || operation === "extract-audio") && (
            <TransformSettings
              media={media}
              settings={settings}
              update={update}
            />
          )}
          {compress && (
            <AdvancedSettings
              key={advancedRevision}
              settings={settings}
              update={update}
            />
          )}
          <section className="surface output-card">
            <div className="section-title">
              <h2>保存到</h2>
              <Icon name="folder" />
            </div>
            <MaterialSelect
              label="输出文件格式"
              value={output.split(".").pop()?.toLowerCase() ?? ""}
              options={operationExtensions(settings).map((extension) => ({
                value: extension,
                label: extension.toUpperCase(),
              }))}
              onChange={(extension) => {
                setOutput((current) =>
                  /\.[^./\\]+$/.test(current)
                    ? current.replace(/\.[^./\\]+$/, `.${extension}`)
                    : `${current}.${extension}`,
                );
                setActionError("");
              }}
            />
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
            <p className="field-hint">
              最终文件名：{output.split(/[\\/]/).pop() || "尚未设置"}
            </p>
            <div className="action-row">
              <button
                className="button text"
                disabled={!connected || choosingOutput || !output}
                onClick={async () => {
                  setChoosingOutput(true);
                  setActionError("");
                  try {
                    setOutput(await API.suggestOutputPath(output));
                  } catch (cause) {
                    setActionError(errorMessage(cause));
                  } finally {
                    setChoosingOutput(false);
                  }
                }}
              >
                自动避开重名
              </button>
              <span className="field-hint">
                不会覆盖源文件或已有结果。上次选择的输出文件夹会用于下次导入。
              </span>
            </div>
          </section>
          {renderBatch?.(settings)}
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
              (!compress || hasVideo) &&
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
