import { t } from "../i18n";
import {
  memo,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { Job } from "../bridge";
import { formatBytes, formatDuration, savedPercent } from "../format";
import { Icon } from "./Icon";
import { formatTimecode } from "../trim";
import { API } from "../bridge";
import { errorMessage } from "../format";
import { jobErrorHelp } from "../jobError";
import {
  estimatedRemainingSeconds,
  orderQueueJobs,
  type QueueScheduleStatus,
} from "../queueSchedule";
import type { SnackbarAction } from "../snackbar";

const stateNames: Record<string, string> = {
  queued: "等待中",
  preparing: "准备中",
  running: "处理中",
  cancelling: "正在取消",
  cancelled: "已取消",
  completed: "已完成",
  failed: "失败",
  interrupted: "已中断",
};
const activeStates = new Set(["queued", "preparing", "running", "cancelling"]);
const jobPageSize = 20;

const JobCard = memo(function JobCard({
  job,
  cancel,
  reuse,
  rank,
  pendingCount,
  paused,
  move,
  notify,
  compareSample,
  notificationTarget,
}: {
  job: Job;
  cancel: (id: string) => void;
  reuse: (id: string) => Promise<void>;
  rank: number;
  pendingCount: number;
  paused: boolean;
  move?: (id: string, direction: number) => Promise<unknown> | void;
  notify?: (message: string, options?: { action?: SnackbarAction }) => void;
  compareSample?: (id: string) => void;
  notificationTarget?: boolean;
}) {
  const [actionError, setActionError] = useState("");
  const [busy, setBusy] = useState(false);
  const [moving, setMoving] = useState(false);
  async function moveJob(direction: number) {
    setMoving(true);
    setActionError("");
    try {
      await move?.(job.id, direction);
      if (move && notify)
        notify(
          t(direction < 0 ? "已将 {{0}} 上移一位。" : "已将 {{0}} 下移一位。", [
            name,
          ]),
          {
            action: {
              label: t("撤销"),
              run: () => move(job.id, -direction),
            },
          },
        );
    } catch (cause) {
      setActionError(errorMessage(cause));
    } finally {
      setMoving(false);
    }
  }
  async function action(kind: "file" | "folder" | "reuse") {
    setBusy(true);
    setActionError("");
    try {
      if (kind === "reuse") await reuse(job.id);
      else await API.openJobOutput(job.id, kind === "folder");
    } catch (cause) {
      setActionError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }
  const state = job.state.toLowerCase();
  const active = activeStates.has(state);
  const progress = Math.min(1, Math.max(0, job.progress || 0));
  const name = job.input.displayName || job.input.uri;
  const remaining = estimatedRemainingSeconds(job);
  const sample = job.projectID.startsWith("sample-");
  const operation = job.project?.settings.operation;
  const operationName =
    operation === "remux"
      ? "转封装"
      : operation === "extract-audio"
        ? "提取音频"
        : operation === "extract-subtitle"
          ? "提取字幕"
          : "压缩";
  return (
    <article
      id={`job-${job.id}`}
      className={`surface job-card ${notificationTarget ? "notification-target" : ""}`}
      tabIndex={-1}
    >
      <div className="job-heading">
        <div
          className={`job-file-icon ${state === "completed" ? "completed" : ""}`}
        >
          <Icon name={state === "completed" ? "check" : "video"} />
        </div>
        <div>
          <h2 title={t(name)}>{t(name)}</h2>
          <span className={`job-state state-${state}`}>
            {t(stateNames[state] || job.state)} · {t(operationName)}
          </span>
        </div>
        {active && (
          <button
            className="button outlined cancel-button"
            onClick={() => cancel(job.id)}
            disabled={state === "cancelling"}
          >
            <Icon name="close" />
            {t(state === "cancelling" ? "正在取消" : "取消任务")}
          </button>
        )}
      </div>
      {sample && (
        <p className="field-hint">{t("样片试压 · 用于检查画质和设置")}</p>
      )}
      {state === "queued" && rank >= 0 && (
        <div className="queue-order-controls">
          <span className="field-hint">
            {t("第")} {rank + 1} {t("项")}{" "}
            {t(paused ? " · 已暂停启动" : " · 按顺序执行")}
          </span>
          {move && (
            <div className="action-row">
              <button
                className="button outlined"
                aria-label={t("将 {{0}} 上移一位", [name])}
                disabled={moving || rank === 0}
                onClick={() => void moveJob(-1)}
              >
                {t("上移")}
              </button>
              <button
                className="button outlined"
                aria-label={t("将 {{0}} 下移一位", [name])}
                disabled={moving || rank === pendingCount - 1}
                onClick={() => void moveJob(1)}
              >
                {t("下移")}
              </button>
            </div>
          )}
        </div>
      )}
      {job.trim && (
        <p className="field-hint">
          {t("剪辑并")} {t(operationName)} ·{" "}
          {t(formatTimecode(job.trim.startSeconds))} →{t(" ")}
          {t(formatTimecode(job.trim.endSeconds))}
        </p>
      )}
      {active && (
        <>
          <div className="progress-label">
            <span>
              {t(
                state === "queued"
                  ? paused
                    ? "已暂停启动后续任务"
                    : "任务等待执行"
                  : state === "preparing"
                    ? "正在准备执行计划"
                    : job.finishing
                      ? "正在校验并保存输出"
                      : "处理进度",
              )}
            </span>
            <strong>{Math.min(99, Math.floor(progress * 100))}%</strong>
          </div>
          <progress
            max="1"
            value={progress}
            aria-label={t("{{0}} 处理进度", [name])}
          />
          <dl className="job-metrics">
            <div>
              <dt>FPS</dt>
              <dd>{t(job.fps ? job.fps.toFixed(1) : "—")}</dd>
            </div>
            <div>
              <dt>{t("处理速度")}</dt>
              <dd>{t(job.speed ? `${job.speed.toFixed(2)}×` : "—")}</dd>
            </div>
            <div>
              <dt>{t("已处理时长")}</dt>
              <dd>{t(formatDuration(job.currentTime))}</dd>
            </div>
            <div>
              <dt>{t("总时长")}</dt>
              <dd>{t(formatDuration(job.totalTime))}</dd>
            </div>
          </dl>
          {state === "running" && !job.finishing && (
            <p className="field-hint queue-eta">
              {t("预计剩余：")}{" "}
              {t(
                remaining === null
                  ? "正在估算…"
                  : `约 ${formatDuration(remaining)}`,
              )}
              <span>{t("根据当前速度估算，实际耗时会变化。")}</span>
            </p>
          )}
        </>
      )}
      {state === "completed" && (
        <>
          {job.verification && (
            <div className="field-hint" role="status">
              <p>
                {t("输出结构已校验 · 视频")} {job.verification.videoStreams}{" "}
                {t("/ 音频")}
                {t(" ")}
                {job.verification.audioStreams} {t("/ 字幕")}
                {t(" ")}
                {job.verification.subtitleStreams}{" "}
                {t(
                  job.verification.fullDecodePassed
                    ? " · 完整解码复检通过"
                    : "",
                )}{" "}
                {t(
                  job.verification.durationSeconds > 0
                    ? ` · ${formatDuration(job.verification.durationSeconds)}`
                    : "",
                )}
              </p>
              <p>
                {t(
                  "已检查文件可解析及轨道结构；完整画面和声音请打开文件确认。",
                )}
              </p>
              {!!job.verification.warnings?.length && (
                <div className="output-review-note">
                  <strong>{t("需要检查")}</strong>
                  {job.verification.warnings.map((warning) => (
                    <p key={warning}>{t(warning)}</p>
                  ))}
                </div>
              )}
            </div>
          )}
          <div className="result-banner">
            <Icon name="check" />
            <span>{t("处理完成")}</span>
            <strong>
              {t(
                sample
                  ? `样片 ${formatDuration(job.totalTime)}`
                  : operation && operation !== "compress"
                    ? operationName
                    : savedPercent(job.inputSize, job.outputSize),
              )}
            </strong>
          </div>
          {job.trim && (
            <p className="field-hint">
              {t(
                sample
                  ? "样片仅包含所选片段，体积不能直接与完整源文件比较。"
                  : "体积减少包含剪掉的内容，比较基准为完整源文件。",
              )}
            </p>
          )}
          <dl className="job-metrics">
            <div>
              <dt>{t("原始大小")}</dt>
              <dd>{t(formatBytes(job.inputSize))}</dd>
            </div>
            <div>
              <dt>{t("输出大小")}</dt>
              <dd>{t(formatBytes(job.outputSize))}</dd>
            </div>
            <div>
              <dt>{t("耗时")}</dt>
              <dd>
                {t(
                  job.duration > 0 && job.duration < 60
                    ? `${job.duration.toFixed(2)} 秒`
                    : formatDuration(job.duration),
                )}
              </dd>
            </div>
            <div>
              <dt>{t("平均速度")}</dt>
              <dd>
                {t(
                  job.duration > 0 && job.totalTime > 0
                    ? `${(job.totalTime / job.duration).toFixed(2)}×`
                    : "—",
                )}
              </dd>
            </div>
          </dl>
        </>
      )}
      {job.error && (
        <div className="inline-error" role="alert">
          <p>
            {t(
              state === "interrupted"
                ? "上次处理已中断，请检查输出后重新执行。"
                : jobErrorHelp(job.error),
            )}
          </p>
          <details>
            <summary>{t("详细原因")}</summary>
            <p>{t(job.error)}</p>
          </details>
        </div>
      )}
      {!active && (
        <div className="action-row">
          {state === "completed" && (
            <>
              <button
                className="button outlined"
                disabled={busy}
                onClick={() => void action("file")}
              >
                {t("打开文件")}
              </button>
              <button
                className="button outlined"
                disabled={busy}
                onClick={() => void action("folder")}
              >
                {t("打开所在文件夹")}
              </button>
              {sample && compareSample && (
                <button
                  className="button outlined"
                  disabled={busy}
                  onClick={() => compareSample(job.id)}
                >
                  {t("比较样片")}
                </button>
              )}
            </>
          )}
          {job.project && (
            <button
              className="button outlined"
              disabled={busy}
              onClick={() => void action("reuse")}
            >
              {t(
                busy
                  ? "处理中…"
                  : state === "failed" ||
                      state === "interrupted" ||
                      state === "cancelled"
                    ? "调整并重试"
                    : "沿用设置再处理",
              )}
            </button>
          )}
        </div>
      )}
      {actionError && (
        <p className="inline-error" role="alert">
          {t(actionError)}
        </p>
      )}
      <div className="job-output">
        <Icon name="folder" />
        <span title={job.output.uri}>{job.output.uri}</span>
      </div>
      <details className="job-logs">
        <summary>
          {t("执行日志")}
          <span>
            {job.logs.length} {t("行")}
          </span>
        </summary>
        <pre>
          {job.logs.length
            ? job.logs.join("\n")
            : t("暂无日志。任务启动后，FFmpeg stderr 会显示在这里。")}
        </pre>
      </details>
    </article>
  );
});

export function Queue({
  jobs,
  cancel,
  refresh,
  compress,
  reuse,
  queueStatus,
  onPauseQueue,
  onMoveQueued,
  notify,
  compareSample,
  focusJobID,
  onFocusHandled,
}: {
  jobs: Job[];
  cancel: (id: string) => void;
  refresh: () => void;
  compress: () => void;
  reuse: (id: string) => Promise<void>;
  queueStatus?: QueueScheduleStatus | null;
  onPauseQueue?: (paused: boolean) => Promise<unknown> | void;
  onMoveQueued?: (id: string, direction: number) => Promise<unknown> | void;
  notify?: (message: string, options?: { action?: SnackbarAction }) => void;
  compareSample?: (id: string) => void;
  focusJobID?: string | null;
  onFocusHandled?: () => void;
}) {
  const [filter, setFilter] = useState<
    "all" | "active" | "completed" | "failed"
  >("all");
  const { active, completed, failed, visible } = useMemo(() => {
    const nextActive = jobs.filter((job) =>
      activeStates.has(job.state.toLowerCase()),
    );
    const nextCompleted = jobs.filter(
      (job) => job.state.toLowerCase() === "completed",
    );
    const nextFailed = jobs.filter((job) =>
      ["failed", "interrupted"].includes(job.state.toLowerCase()),
    );
    return {
      active: nextActive,
      completed: nextCompleted,
      failed: nextFailed,
      visible:
        filter === "active"
          ? nextActive
          : filter === "completed"
            ? nextCompleted
            : filter === "failed"
              ? nextFailed
              : jobs,
    };
  }, [filter, jobs]);
  const [storageError, setStorageError] = useState("");
  const [scheduleError, setScheduleError] = useState("");
  const [scheduling, setScheduling] = useState(false);
  const [renderLimit, setRenderLimit] = useState(jobPageSize);
  const pendingIDs = useMemo(
    () =>
      queueStatus?.pendingIDs ??
      jobs
        .filter((job) => job.state.toLowerCase() === "queued")
        .map((job) => job.id),
    [jobs, queueStatus?.pendingIDs],
  );
  const orderedVisible = useMemo(
    () => orderQueueJobs(visible, pendingIDs),
    [pendingIDs, visible],
  );
  const pendingRanks = useMemo(
    () => new Map(pendingIDs.map((id, index) => [id, index])),
    [pendingIDs],
  );
  const paused = queueStatus?.paused ?? false;
  const callbacks = useRef({
    cancel,
    reuse,
    onMoveQueued,
    notify,
    compareSample,
  });
  useLayoutEffect(() => {
    callbacks.current = { cancel, reuse, onMoveQueued, notify, compareSample };
  }, [cancel, compareSample, notify, onMoveQueued, reuse]);
  const stableCancel = useCallback(
    (id: string) => callbacks.current.cancel(id),
    [],
  );
  const stableReuse = useCallback(
    (id: string) => callbacks.current.reuse(id),
    [],
  );
  const stableMove = useCallback(
    (id: string, direction: number) =>
      callbacks.current.onMoveQueued?.(id, direction),
    [],
  );
  const stableNotify = useCallback(
    (message: string, options?: { action?: SnackbarAction }) =>
      callbacks.current.notify?.(message, options),
    [],
  );
  const stableCompare = useCallback(
    (id: string) => callbacks.current.compareSample?.(id),
    [],
  );
  async function pauseQueue() {
    const nextPaused = !paused;
    setScheduling(true);
    setScheduleError("");
    try {
      await onPauseQueue?.(nextPaused);
      if (onPauseQueue && notify)
        notify(
          t(nextPaused ? "已暂停后续任务。" : "已继续按顺序处理后续任务。"),
          {
            action: {
              label: t("撤销"),
              run: () => onPauseQueue(paused),
            },
          },
        );
    } catch (cause) {
      setScheduleError(errorMessage(cause));
    } finally {
      setScheduling(false);
    }
  }
  const terminalCount = jobs.length - active.length;
  useEffect(() => {
    let alive = true;
    void API.getWorkspace().then(
      (value) => {
        if (alive) setStorageError(value.error);
      },
      (cause) => {
        if (alive)
          setStorageError(`历史保存状态读取失败：${errorMessage(cause)}`);
      },
    );
    return () => {
      alive = false;
    };
  }, [terminalCount, jobs.length]);
  const focusIndex = focusJobID
    ? orderedVisible.findIndex((job) => job.id === focusJobID)
    : -1;
  const activeRenderLimit = Math.max(
    renderLimit,
    focusIndex >= 0 ? focusIndex + 1 : 0,
  );
  useEffect(() => {
    if (!focusJobID) return;
    const frame = requestAnimationFrame(() => {
      const target = document.getElementById(`job-${focusJobID}`);
      target?.scrollIntoView({ block: "center", behavior: "smooth" });
      target?.focus({ preventScroll: true });
    });
    const timer = window.setTimeout(() => onFocusHandled?.(), 2600);
    return () => {
      cancelAnimationFrame(frame);
      window.clearTimeout(timer);
    };
  }, [focusJobID, onFocusHandled, activeRenderLimit]);
  return (
    <div className="page queue-page">
      <header className="page-heading">
        <div>
          <h1>{t("任务队列")}</h1>
          <p>
            {t(
              "实时进度、执行日志与最近 200 条本地历史。中断任务不会自动继续。",
            )}
          </p>
        </div>
        <button className="button outlined" onClick={refresh}>
          {t("刷新队列")}
        </button>
      </header>
      {storageError && (
        <p className="inline-error" role="alert">
          {t(storageError)}
        </p>
      )}
      {onPauseQueue && (
        <div className="queue-scheduling surface" aria-busy={scheduling}>
          <div>
            <strong>
              {t(paused ? "后续任务已暂停启动" : "按队列顺序自动处理")}
            </strong>
            <p className="field-hint">
              {t("当前正在处理的任务会继续完成。等待中的任务可调整顺序。")}
            </p>
          </div>
          <button
            className="button outlined"
            disabled={scheduling || !queueStatus}
            onClick={() => void pauseQueue()}
          >
            {t(scheduling ? "正在更新…" : paused ? "继续队列" : "暂停后续任务")}
          </button>
        </div>
      )}
      {scheduleError && (
        <p className="inline-error" role="alert">
          {t(scheduleError)}
        </p>
      )}
      <div className="queue-toolbar">
        <div className="segmented">
          <button
            className={filter === "all" ? "selected" : ""}
            aria-pressed={filter === "all"}
            onClick={() => {
              setFilter("all");
              setRenderLimit(jobPageSize);
            }}
          >
            {t("全部")} {jobs.length}
          </button>
          <button
            className={filter === "active" ? "selected" : ""}
            aria-pressed={filter === "active"}
            onClick={() => {
              setFilter("active");
              setRenderLimit(jobPageSize);
            }}
          >
            {t("进行中")} {active.length}
          </button>
          <button
            className={filter === "completed" ? "selected" : ""}
            aria-pressed={filter === "completed"}
            onClick={() => {
              setFilter("completed");
              setRenderLimit(jobPageSize);
            }}
          >
            {t("已完成")} {completed.length}
          </button>
          <button
            className={filter === "failed" ? "selected" : ""}
            aria-pressed={filter === "failed"}
            onClick={() => {
              setFilter("failed");
              setRenderLimit(jobPageSize);
            }}
          >
            {t("失败 / 中断")} {failed.length}
          </button>
        </div>
        <span className="subtle-label">
          {t(
            active.length
              ? `${active.length} 项等待或执行中`
              : "工作台准备就绪",
          )}
        </span>
      </div>
      {visible.length ? (
        <div className="job-list">
          {orderedVisible.slice(0, activeRenderLimit).map((job) => (
            <JobCard
              key={job.id}
              job={job}
              cancel={stableCancel}
              reuse={stableReuse}
              rank={pendingRanks.get(job.id) ?? -1}
              pendingCount={pendingIDs.length}
              paused={paused}
              move={onMoveQueued ? stableMove : undefined}
              notify={notify ? stableNotify : undefined}
              compareSample={compareSample ? stableCompare : undefined}
              notificationTarget={focusJobID === job.id}
            />
          ))}
          {orderedVisible.length > activeRenderLimit && (
            <div className="action-row queue-load-more">
              <button
                className="button outlined"
                onClick={() =>
                  setRenderLimit((current) =>
                    Math.min(orderedVisible.length, current + jobPageSize),
                  )
                }
              >
                {t("加载更多任务")} ·{" "}
                {orderedVisible.length - activeRenderLimit}
              </button>
            </div>
          )}
        </div>
      ) : (
        <section className="empty-state surface">
          <Icon name="queue" />
          <h2>{t(filter === "all" ? "这里还很安静" : "没有符合条件的任务")}</h2>
          <p>
            {t(
              filter === "all"
                ? "创建压缩任务后，就可以在这里查看进度。"
                : "切换筛选条件，查看其他任务。",
            )}
          </p>
          {filter === "all" && (
            <button className="button primary" onClick={compress}>
              {t("创建压缩任务")}
              <Icon name="arrow" />
            </button>
          )}
        </section>
      )}
    </div>
  );
}
