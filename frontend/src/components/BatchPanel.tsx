import { useState } from "react";
import { API } from "../bridge";
import type { Job, Media, Settings } from "../bridge";
import { defaultBatchTracks, directoryOf, makeBatchProjects } from "../batch";
import type { BatchTrackOverride } from "../batch";
import { errorMessage } from "../format";
import { operationLabels, operationOf, validateSettings } from "../project";
import { MaterialSelect } from "./MaterialSelect";

export function BatchPanel({
  files,
  settings,
  connected,
  onJobs,
  showQueue,
}: {
  files: Media[];
  settings: Settings;
  connected: boolean;
  onJobs: (jobs: Job[]) => void;
  showQueue: () => void;
}) {
  const [excluded, setExcluded] = useState<string[]>([]);
  const [directory, setDirectory] = useState(() =>
    directoryOf(files[0]?.source.uri ?? ""),
  );
  const [suffix, setSuffix] = useState(() => `pheeb-${Date.now()}`);
  const [overrides, setOverrides] = useState<
    Record<string, BatchTrackOverride>
  >({});
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const selected = files.filter((file) => !excluded.includes(file.source.uri));
  const operation = operationOf(settings);
  let projects = [] as ReturnType<typeof makeBatchProjects>;
  let namingError = "";
  try {
    projects = makeBatchProjects(
      selected,
      settings,
      directory,
      suffix,
      overrides,
    );
  } catch (cause) {
    namingError = errorMessage(cause);
  }
  const issues = projects.flatMap((project, index) =>
    validateSettings(
      project.settings,
      project.output.uri,
      selected[index].source.uri,
      selected[index],
    ).map((error) => `${selected[index].name}：${error}`),
  );
  async function chooseDirectory() {
    if (busy) return;
    setBusy(true);
    setMessage("");
    try {
      const path = await API.chooseOutputDirectory(directory);
      if (path) setDirectory(path);
    } catch (cause) {
      setMessage(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }
  async function submit() {
    if (busy || namingError || issues.length || !projects.length) return;
    setBusy(true);
    setMessage("");
    try {
      const jobs = await API.startBatch(projects);
      onJobs(jobs);
      showQueue();
    } catch (cause) {
      setMessage(`整批未提交：${errorMessage(cause)}`);
    } finally {
      setBusy(false);
    }
  }
  function trackOptions(file: Media, kind: "audio" | "subtitle") {
    const tracks = file.streams.filter((stream) => stream.codecType === kind);
    const extracting = operation === `extract-${kind}`;
    return [
      ...(!extracting
        ? [
            { value: "none", label: kind === "audio" ? "静音" : "不保留字幕" },
            { value: "all", label: kind === "audio" ? "全部音轨" : "全部字幕" },
          ]
        : []),
      ...tracks.map((stream, index) => ({
        value: String(stream.index),
        label: `${kind === "audio" ? "音轨" : "字幕"} ${index + 1} · ${stream.title || stream.language || stream.codecName}`,
      })),
      ...(extracting && !tracks.length
        ? [{ value: "none", label: "没有可提取的轨道" }]
        : []),
    ];
  }
  return (
    <section className="surface settings-card batch-panel">
      <div className="section-title">
        <h2>批量处理</h2>
        <span className="subtle-label">
          {selected.length} / {files.length} 个文件
        </span>
      </div>
      <p className="field-hint">
        使用上面的{operationLabels[operation]}
        通用设置，处理每个文件的完整时长。剪辑区间和裁剪边距只用于单文件；每个文件的音轨、字幕在下面单独选择。
      </p>
      <div className="field-grid">
        <div>
          <label className="field-label" htmlFor="batch-directory">
            输出文件夹
          </label>
          <input
            id="batch-directory"
            value={directory}
            onChange={(event) => setDirectory(event.target.value)}
          />
        </div>
        <div>
          <label className="field-label" htmlFor="batch-suffix">
            文件名后缀
          </label>
          <input
            id="batch-suffix"
            maxLength={64}
            value={suffix}
            onChange={(event) => setSuffix(event.target.value)}
          />
        </div>
      </div>
      <button
        className="button outlined"
        disabled={busy || !connected}
        onClick={() => void chooseDirectory()}
      >
        选择输出文件夹
      </button>
      <div className="batch-files">
        {files.map((file) => {
          const chosen = !excluded.includes(file.source.uri);
          const tracks =
            overrides[file.source.uri] ?? defaultBatchTracks(file, settings);
          const output = projects.find(
            (p) => p.inputs[0].uri === file.source.uri,
          )?.output.displayName;
          const setTrack = (kind: "audio" | "subtitle", value: string) =>
            setOverrides((current) => ({
              ...current,
              [file.source.uri]: { ...tracks, [kind]: value },
            }));
          return (
            <article className="batch-file" key={file.source.uri}>
              <label className="batch-file-name">
                <input
                  type="checkbox"
                  checked={chosen}
                  disabled={busy}
                  onChange={() =>
                    setExcluded((current) =>
                      chosen
                        ? [...current, file.source.uri]
                        : current.filter((uri) => uri !== file.source.uri),
                    )
                  }
                />
                <span>{file.name}</span>
              </label>
              {chosen && (
                <>
                  <div className="field-grid">
                    {operation !== "extract-subtitle" && (
                      <MaterialSelect
                        label={`${file.name} 的音轨`}
                        value={tracks.audio}
                        options={trackOptions(file, "audio")}
                        onChange={(value) => setTrack("audio", value)}
                      />
                    )}
                    {operation !== "extract-audio" && (
                      <MaterialSelect
                        label={`${file.name} 的字幕`}
                        value={tracks.subtitle}
                        options={trackOptions(file, "subtitle")}
                        onChange={(value) => setTrack("subtitle", value)}
                      />
                    )}
                  </div>
                  <p className="field-hint batch-output">
                    输出：{output || "请修正文件名"}
                  </p>
                </>
              )}
            </article>
          );
        })}
      </div>
      {(namingError || issues.length > 0) && (
        <p className="inline-error" role="alert">
          {namingError || issues.slice(0, 3).join("；")}
        </p>
      )}
      {message && (
        <p className="inline-error" role="alert">
          {message}
        </p>
      )}
      <button
        className="button outlined"
        disabled={
          busy ||
          !connected ||
          !projects.length ||
          Boolean(namingError) ||
          issues.length > 0
        }
        onClick={() => void submit()}
      >
        {busy ? "检查并提交中…" : `将 ${selected.length} 个文件加入队列`}
      </button>
      <p className="field-hint">
        提交前检查整批格式、轨道和路径。已存在的文件不会覆盖；当前任务按队列顺序逐个处理。
      </p>
    </section>
  );
}
