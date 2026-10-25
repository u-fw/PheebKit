import { t } from "../i18n";
import { useEffect, useId, useMemo, useRef, useState } from "react";
import { API } from "../bridge";
import type { Job, Media, Settings, WorkspacePreferences } from "../bridge";
import { defaultBatchTracks, directoryOf, makeBatchProjects } from "../batch";
import type { BatchTrackOverride } from "../batch";
import { errorMessage } from "../format";
import { operationLabels, operationOf, selectedTracks } from "../project";
import {
  batchReviewKey,
  batchFileErrors,
  reviewBatch,
  type BatchReviewRow,
} from "../batchReview";
import { formatTimecode } from "../trim";
import { MaterialSelect } from "./MaterialSelect";

export function BatchPanel({
  files,
  settings,
  connected,
  onJobs,
  showQueue,
  outputDirectory,
  onOutputDirectory,
  onNotice,
  preferences,
  applyRecipe,
  manageRecipes,
}: {
  files: Media[];
  settings: Settings;
  connected: boolean;
  onJobs: (jobs: Job[]) => void;
  showQueue: () => void;
  outputDirectory?: string;
  onOutputDirectory?: (directory: string) => void;
  onNotice?: (message: string) => void;
  preferences?: WorkspacePreferences | null;
  applyRecipe?: (settings: Settings) => void;
  manageRecipes?: () => void;
}) {
  const [excluded, setExcluded] = useState<string[]>([]);
  const [directory, setDirectory] = useState(
    () => outputDirectory || directoryOf(files[0]?.source.uri ?? ""),
  );
  const [suffix, setSuffix] = useState(() => `pheeb-${Date.now()}`);
  const [overrides, setOverrides] = useState<
    Record<string, BatchTrackOverride>
  >({});
  const [busy, setBusy] = useState<"directory" | "review" | "submit" | null>(
    null,
  );
  const [cancelling, setCancelling] = useState(false);
  const [message, setMessage] = useState({ text: "", error: false });
  function report(text: string, error = true) {
    setMessage({ text, error });
  }
  const [recipeID, setRecipeID] = useState("");
  const [recipeNotice, setRecipeNotice] = useState("");
  const [review, setReview] = useState<{
    key: string;
    rows: Record<string, BatchReviewRow>;
    complete: boolean;
  } | null>(null);
  const epoch = useRef({ value: 0 });
  const progressID = useId();
  const selected = useMemo(
    () => files.filter((file) => !excluded.includes(file.source.uri)),
    [files, excluded],
  );
  const operation = operationOf(settings);
  const { projects, namingError } = useMemo(() => {
    try {
      return {
        projects: makeBatchProjects(
          selected,
          settings,
          directory,
          suffix,
          overrides,
        ),
        namingError: "",
      };
    } catch (cause) {
      return { projects: [], namingError: errorMessage(cause) };
    }
  }, [selected, settings, directory, suffix, overrides]);
  const reviewKey = useMemo(
    () => batchReviewKey(projects, selected),
    [projects, selected],
  );
  useEffect(() => {
    const state = epoch.current;
    return () => {
      state.value++;
    };
  }, [reviewKey]);
  const currentReview = review?.key === reviewKey ? review : null;
  const checked = Object.keys(currentReview?.rows ?? {}).length;
  const failed = Object.values(currentReview?.rows ?? {}).filter(
    (row) => row.errors.length > 0,
  ).length;
  const ready = Boolean(
    currentReview?.complete &&
    projects.length &&
    checked === projects.length &&
    projects.every(
      (project) =>
        currentReview.rows[project.id]?.plan &&
        !currentReview.rows[project.id]?.errors.length,
    ),
  );
  const recipe = preferences?.presets.find((item) => item.id === recipeID);
  const issues = projects.flatMap((project, index) =>
    batchFileErrors(project, selected[index]).map(
      (error) => `${selected[index].name}：${error}`,
    ),
  );
  async function chooseDirectory() {
    if (busy) return;
    setBusy("directory");
    report("", false);
    try {
      const path = await API.chooseOutputDirectory(directory);
      if (path) setDirectory(path);
    } catch (cause) {
      report(errorMessage(cause));
    } finally {
      setBusy(null);
    }
  }
  async function submit() {
    if (busy || !ready || namingError || issues.length || !projects.length)
      return;
    setBusy("submit");
    report("", false);
    try {
      const jobs = await API.startBatch(projects);
      setReview(null);
      onJobs(jobs);
      void API.rememberOutputDirectory(projects[0].output.uri)
        .then(() => onOutputDirectory?.(directory))
        .catch((cause: unknown) =>
          onNotice?.(`整批已提交，但输出目录未能记住：${errorMessage(cause)}`),
        );
      showQueue();
    } catch (cause) {
      setReview(null);
      report(`整批未提交：${errorMessage(cause)}`);
    } finally {
      setBusy(null);
    }
  }
  async function checkBatch() {
    if (busy || !connected || namingError || !projects.length) return;
    const generation = ++epoch.current.value;
    setBusy("review");
    setCancelling(false);
    report("", false);
    setReview({ key: reviewKey, rows: {}, complete: false });
    try {
      const complete = await reviewBatch(
        projects,
        selected,
        API.preview,
        () => epoch.current.value === generation,
        (row) => {
          setReview((current) =>
            current?.key === reviewKey
              ? { ...current, rows: { ...current.rows, [row.projectID]: row } }
              : current,
          );
        },
      );
      if (complete)
        setReview((current) =>
          current?.key === reviewKey ? { ...current, complete: true } : current,
        );
      else report("批量预检已停止，没有任务入队。", false);
    } catch (cause) {
      report(`批量预检失败：${errorMessage(cause)}`);
    } finally {
      setBusy(null);
      setCancelling(false);
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
        label: `${t(kind === "audio" ? "音轨" : "字幕")} ${index + 1} · ${stream.title || stream.language || stream.codecName}`,
        translate: false,
      })),
      ...(extracting && !tracks.length
        ? [{ value: "none", label: "没有可提取的轨道" }]
        : []),
    ];
  }
  return (
    <section className="surface settings-card batch-panel">
      <div className="section-title">
        <h2>{t("批量处理")}</h2>
        <span className="subtle-label">
          {selected.length} / {files.length} {t("个文件")}
        </span>
      </div>
      <p className="field-hint">
        {t("使用上面的")} {t(operationLabels[operation])}{" "}
        {t(
          "通用设置，处理每个文件的完整时长。剪辑、多片段、字幕烧录和裁剪边距只用于当前单文件；每个文件的音轨、软字幕在下面单独选择。",
        )}
      </p>
      {(applyRecipe || manageRecipes) && (
        <div className="batch-recipes">
          <h3>{t("批量配方")}</h3>
          <div className="action-row">
            <MaterialSelect
              label={t("已保存的通用配方")}
              value={recipe?.id ?? ""}
              options={[
                { value: "", label: "选择常用配置" },
                ...(preferences?.presets ?? []).map((item) => ({
                  value: item.id,
                  label: item.name,
                  translate: false,
                })),
              ]}
              onChange={setRecipeID}
              disabled={Boolean(busy)}
            />
            <button
              className="button outlined"
              disabled={!recipe || !applyRecipe || Boolean(busy)}
              onClick={() => {
                if (!recipe || !applyRecipe) return;
                setOverrides({});
                applyRecipe(structuredClone(recipe.settings));
                setRecipeNotice(
                  "配方已应用到通用参数，请检查每个文件的轨道，再重新预检。没有任务自动开始。",
                );
              }}
            >
              {t("应用配方")}
            </button>
            {manageRecipes && (
              <button
                className="button text"
                disabled={Boolean(busy)}
                onClick={manageRecipes}
              >
                {t("保存／导入／导出配方")}
              </button>
            )}
          </div>
          <p className="field-hint">
            {t(
              "配方复用常用配置，可继续修改全部通用参数；应用后重置逐文件轨道选择。文件路径、剪辑范围和轨道编号不会从配方复制。",
            )}
          </p>
          {recipeNotice && (
            <p className="field-hint" role="status">
              {t(recipeNotice)}
            </p>
          )}
        </div>
      )}
      <div className="field-grid">
        <div>
          <label className="field-label" htmlFor="batch-directory">
            {t("输出文件夹")}
          </label>
          <input
            id="batch-directory"
            disabled={Boolean(busy)}
            value={directory}
            onChange={(event) => setDirectory(event.target.value)}
          />
        </div>
        <div>
          <label className="field-label" htmlFor="batch-suffix">
            {t("文件名后缀")}
          </label>
          <input
            id="batch-suffix"
            disabled={Boolean(busy)}
            maxLength={64}
            value={suffix}
            onChange={(event) => setSuffix(event.target.value)}
          />
        </div>
      </div>
      <button
        className="button outlined"
        disabled={Boolean(busy) || !connected}
        onClick={() => void chooseDirectory()}
      >
        {t("选择输出文件夹")}
      </button>
      <div className="batch-files">
        {files.map((file) => {
          const chosen = !excluded.includes(file.source.uri);
          const tracks =
            overrides[file.source.uri] ?? defaultBatchTracks(file, settings);
          const output = projects.find(
            (p) => p.inputs[0].uri === file.source.uri,
          )?.output.displayName;
          const project = projects.find(
            (p) => p.inputs[0].uri === file.source.uri,
          );
          const result = project ? currentReview?.rows[project.id] : undefined;
          const localErrors = project ? batchFileErrors(project, file) : [];
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
                  disabled={Boolean(busy)}
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
                        label={t("{{0}} 的音轨", [file.name])}
                        value={tracks.audio}
                        options={trackOptions(file, "audio")}
                        disabled={Boolean(busy)}
                        onChange={(value) => setTrack("audio", value)}
                      />
                    )}
                    {operation !== "extract-audio" && (
                      <MaterialSelect
                        label={t("{{0}} 的字幕", [file.name])}
                        value={tracks.subtitle}
                        options={trackOptions(file, "subtitle")}
                        disabled={Boolean(busy)}
                        onChange={(value) => setTrack("subtitle", value)}
                      />
                    )}
                  </div>
                  <p className="field-hint batch-output">
                    {t("输出：")} {output || t("请修正文件名")}
                  </p>
                  <div className="batch-review-row">
                    <p className="field-hint">
                      {t(operationLabels[operation])} ·{" "}
                      {(result?.plan?.durationSeconds ?? file.durationSeconds) >
                      0
                        ? formatTimecode(
                            result?.plan?.durationSeconds ??
                              file.durationSeconds,
                          )
                        : t("未知")}
                    </p>
                    {result?.plan && project && (
                      <>
                        <p className="field-hint">
                          {t("处理方式")} ·{" "}
                          {operation === "compress"
                            ? `${result.plan.encoderKind === "gpu" ? "GPU" : "CPU"} · ${result.plan.encoderLabel}`
                            : t("直接处理轨道")}
                        </p>
                        {result.plan.encoderReason && (
                          <p className="field-hint">
                            {t(result.plan.encoderReason)}
                          </p>
                        )}
                        <p className="field-hint">
                          {t("音轨")} ·{" "}
                          {selectedTracks(file, "audio", project.settings)
                            .map((track) => `#${track.index}`)
                            .join(", ") || t("不保留")}{" "}
                          · {t("字幕")} ·{" "}
                          {selectedTracks(file, "subtitle", project.settings)
                            .map((track) => `#${track.index}`)
                            .join(", ") || t("不保留")}
                        </p>
                        <details>
                          <summary>{t("查看此文件的执行参数")}</summary>
                          <pre className="batch-command">
                            {result.plan.preview}
                          </pre>
                        </details>
                      </>
                    )}
                    {(result?.errors ?? localErrors).map((error) => (
                      <p className="inline-error" key={error}>
                        {t(error)}
                      </p>
                    ))}
                    <span className="subtle-label">
                      {t(
                        result
                          ? result.errors.length
                            ? "需要修正"
                            : "预检通过"
                          : "尚未预检",
                      )}
                    </span>
                  </div>
                </>
              )}
            </article>
          );
        })}
      </div>
      {(namingError || issues.length > 0) && (
        <p className="inline-error" role="alert">
          {t(namingError || issues.slice(0, 3).join("；"))}
        </p>
      )}
      {message.text && (
        <p
          className={message.error ? "inline-error" : "field-hint"}
          role={message.error ? "alert" : "status"}
        >
          {t(message.text)}
        </p>
      )}
      {currentReview?.complete && (
        <p className="field-hint" role="status">
          {t("预检完成：{{0}} 个通过，{{1}} 个需要修正。", [
            checked - failed,
            failed,
          ])}
        </p>
      )}
      {review && !currentReview && (
        <p className="field-hint" role="status">
          {t("批量设置或文件已改变，请重新预检。")}
        </p>
      )}
      <div className="action-row">
        <button
          className="button outlined"
          disabled={
            Boolean(busy) ||
            !connected ||
            !projects.length ||
            Boolean(namingError)
          }
          onClick={() => void checkBatch()}
        >
          {t(busy === "review" ? "正在逐文件预检…" : "检查整批方案")}
        </button>
        {busy === "review" && (
          <button
            className="button text"
            disabled={cancelling}
            onClick={() => {
              epoch.current.value++;
              setCancelling(true);
            }}
          >
            {t(cancelling ? "正在结束当前检查…" : "停止预检")}
          </button>
        )}
      </div>
      {busy === "review" && (
        <div className="batch-review-progress">
          <label htmlFor={progressID}>
            {t("已检查 {{0}} / {{1}}", [checked, projects.length])}
          </label>
          <progress id={progressID} value={checked} max={projects.length} />
        </div>
      )}
      <button
        className="button outlined"
        disabled={
          Boolean(busy) ||
          !ready ||
          !connected ||
          !projects.length ||
          Boolean(namingError) ||
          issues.length > 0
        }
        onClick={() => void submit()}
      >
        {t(
          busy === "submit"
            ? "检查并提交中…"
            : `将 ${selected.length} 个文件加入队列`,
        )}
      </button>
      <p className="field-hint">
        {t(
          "先检查整批方案，全部通过后由你确认入队。修改设置会使旧预检失效，入队时还会再次检查路径与文件；已有结果不会覆盖。",
        )}
      </p>
    </section>
  );
}
