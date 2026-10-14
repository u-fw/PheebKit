import { useEffect, useState } from "react";
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

function JobCard({
  job,
  cancel,
  reuse,
  rank,
  pendingCount,
  paused,
  move,
  compareSample,
}: {
  job: Job;
  cancel: (id: string) => void;
  reuse: (id: string) => Promise<void>;
  rank: number;
  pendingCount: number;
  paused: boolean;
  move?: (id: string, direction: number) => Promise<unknown> | void;
  compareSample?: (id: string) => void;
}) {
  const [actionError, setActionError] = useState("");
  const [busy, setBusy] = useState(false);
  const [moving, setMoving] = useState(false);
  async function moveJob(direction: number) {
    setMoving(true);
    setActionError("");
    try {
      await move?.(job.id, direction);
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
    <article className="surface job-card">
      <div className="job-heading">
        <div
          className={`job-file-icon ${state === "completed" ? "completed" : ""}`}
        >
          <Icon name={state === "completed" ? "check" : "video"} />
        </div>
        <div>
          <h2 title={name}>{name}</h2>
          <span className={`job-state state-${state}`}>
            {stateNames[state] || job.state} · {operationName}
          </span>
        </div>
        {active && (
          <button
            className="button outlined cancel-button"
            onClick={() => cancel(job.id)}
            disabled={state === "cancelling"}
          >
            <Icon name="close" />
            {state === "cancelling" ? "正在取消" : "取消任务"}
          </button>
        )}
      </div>
      {sample && <p className="field-hint">样片试压 · 用于检查画质和设置</p>}
      {state === "queued" && rank >= 0 && (
        <div className="queue-order-controls">
          <span className="field-hint">
            第 {rank + 1} 项{paused ? " · 已暂停启动" : " · 按顺序执行"}
          </span>
          {move && (
            <div className="action-row">
              <button
                className="button outlined"
                aria-label={`将 ${name} 上移一位`}
                disabled={moving || rank === 0}
                onClick={() => void moveJob(-1)}
              >
                上移
              </button>
              <button
                className="button outlined"
                aria-label={`将 ${name} 下移一位`}
                disabled={moving || rank === pendingCount - 1}
                onClick={() => void moveJob(1)}
              >
                下移
              </button>
            </div>
          )}
        </div>
      )}
      {job.trim && (
        <p className="field-hint">
          剪辑并{operationName} · {formatTimecode(job.trim.startSeconds)} →{" "}
          {formatTimecode(job.trim.endSeconds)}
        </p>
      )}
      {active && (
        <>
          <div className="progress-label">
            <span>
              {state === "queued"
                ? paused
                  ? "已暂停启动后续任务"
                  : "任务等待执行"
                : state === "preparing"
                  ? "正在准备执行计划"
                  : job.finishing
                    ? "正在校验并保存输出"
                    : "处理进度"}
            </span>
            <strong>{Math.min(99, Math.floor(progress * 100))}%</strong>
          </div>
          <progress max="1" value={progress} aria-label={`${name} 处理进度`} />
          <dl className="job-metrics">
            <div>
              <dt>FPS</dt>
              <dd>{job.fps ? job.fps.toFixed(1) : "—"}</dd>
            </div>
            <div>
              <dt>处理速度</dt>
              <dd>{job.speed ? `${job.speed.toFixed(2)}×` : "—"}</dd>
            </div>
            <div>
              <dt>已处理时长</dt>
              <dd>{formatDuration(job.currentTime)}</dd>
            </div>
            <div>
              <dt>总时长</dt>
              <dd>{formatDuration(job.totalTime)}</dd>
            </div>
          </dl>
          {state === "running" && !job.finishing && (
            <p className="field-hint queue-eta">
              预计剩余：
              {remaining === null
                ? "正在估算…"
                : `约 ${formatDuration(remaining)}`}
              <span>根据当前速度估算，实际耗时会变化。</span>
            </p>
          )}
        </>
      )}
      {state === "completed" && (
        <>
          {job.verification && (
            <div className="field-hint" role="status">
              <p>
                输出结构已校验 · 视频 {job.verification.videoStreams} / 音频{" "}
                {job.verification.audioStreams} / 字幕{" "}
                {job.verification.subtitleStreams}
                {job.verification.durationSeconds > 0
                  ? ` · ${formatDuration(job.verification.durationSeconds)}`
                  : ""}
              </p>
              <p>已检查文件可解析及轨道结构；完整画面和声音请打开文件确认。</p>
              {!!job.verification.warnings?.length && (
                <div className="output-review-note">
                  <strong>需要检查</strong>
                  {job.verification.warnings.map((warning) => (
                    <p key={warning}>{warning}</p>
                  ))}
                </div>
              )}
            </div>
          )}
          <div className="result-banner">
            <Icon name="check" />
            <span>处理完成</span>
            <strong>
              {sample
                ? `样片 ${formatDuration(job.totalTime)}`
                : operation && operation !== "compress"
                  ? operationName
                  : savedPercent(job.inputSize, job.outputSize)}
            </strong>
          </div>
          {job.trim && (
            <p className="field-hint">
              {sample
                ? "样片仅包含所选片段，体积不能直接与完整源文件比较。"
                : "体积减少包含剪掉的内容，比较基准为完整源文件。"}
            </p>
          )}
          <dl className="job-metrics">
            <div>
              <dt>原始大小</dt>
              <dd>{formatBytes(job.inputSize)}</dd>
            </div>
            <div>
              <dt>输出大小</dt>
              <dd>{formatBytes(job.outputSize)}</dd>
            </div>
            <div>
              <dt>耗时</dt>
              <dd>
                {job.duration > 0 && job.duration < 60
                  ? `${job.duration.toFixed(2)} 秒`
                  : formatDuration(job.duration)}
              </dd>
            </div>
            <div>
              <dt>平均速度</dt>
              <dd>
                {job.duration > 0 && job.totalTime > 0
                  ? `${(job.totalTime / job.duration).toFixed(2)}×`
                  : "—"}
              </dd>
            </div>
          </dl>
        </>
      )}
      {job.error && (
        <div className="inline-error" role="alert">
          <p>
            {state === "interrupted"
              ? "上次处理已中断，请检查输出后重新执行。"
              : jobErrorHelp(job.error)}
          </p>
          <details>
            <summary>详细原因</summary>
            <p>{job.error}</p>
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
                打开文件
              </button>
              <button
                className="button outlined"
                disabled={busy}
                onClick={() => void action("folder")}
              >
                打开所在文件夹
              </button>
              {sample && compareSample && (
                <button
                  className="button outlined"
                  disabled={busy}
                  onClick={() => compareSample(job.id)}
                >
                  比较样片
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
              {busy
                ? "处理中…"
                : state === "failed" ||
                    state === "interrupted" ||
                    state === "cancelled"
                  ? "调整并重试"
                  : "沿用设置再处理"}
            </button>
          )}
        </div>
      )}
      {actionError && (
        <p className="inline-error" role="alert">
          {actionError}
        </p>
      )}
      <div className="job-output">
        <Icon name="folder" />
        <span title={job.output.uri}>{job.output.uri}</span>
      </div>
      <details className="job-logs">
        <summary>
          执行日志 <span>{job.logs.length} 行</span>
        </summary>
        <pre>
          {job.logs.length
            ? job.logs.join("\n")
            : "暂无日志。任务启动后，FFmpeg stderr 会显示在这里。"}
        </pre>
      </details>
    </article>
  );
}

export function Queue({
  jobs,
  cancel,
  refresh,
  compress,
  reuse,
  queueStatus,
  onPauseQueue,
  onMoveQueued,
  compareSample,
}: {
  jobs: Job[];
  cancel: (id: string) => void;
  refresh: () => void;
  compress: () => void;
  reuse: (id: string) => Promise<void>;
  queueStatus?: QueueScheduleStatus | null;
  onPauseQueue?: (paused: boolean) => Promise<unknown> | void;
  onMoveQueued?: (id: string, direction: number) => Promise<unknown> | void;
  compareSample?: (id: string) => void;
}) {
  const [filter, setFilter] = useState<
    "all" | "active" | "completed" | "failed"
  >("all");
  const active = jobs.filter((job) =>
    activeStates.has(job.state.toLowerCase()),
  );
  const completed = jobs.filter(
    (job) => job.state.toLowerCase() === "completed",
  );
  const failed = jobs.filter((job) =>
    ["failed", "interrupted"].includes(job.state.toLowerCase()),
  );
  const visible =
    filter === "active"
      ? active
      : filter === "completed"
        ? completed
        : filter === "failed"
          ? failed
          : jobs;
  const [storageError, setStorageError] = useState("");
  const [scheduleError, setScheduleError] = useState("");
  const [scheduling, setScheduling] = useState(false);
  const pendingIDs =
    queueStatus?.pendingIDs ??
    jobs
      .filter((job) => job.state.toLowerCase() === "queued")
      .map((job) => job.id);
  const paused = queueStatus?.paused ?? false;
  async function pauseQueue() {
    setScheduling(true);
    setScheduleError("");
    try {
      await onPauseQueue?.(!paused);
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
  return (
    <div className="page queue-page">
      <header className="page-heading">
        <div>
          <h1>任务队列</h1>
          <p>实时进度、执行日志与最近 200 条本地历史。中断任务不会自动继续。</p>
        </div>
        <button className="button outlined" onClick={refresh}>
          刷新队列
        </button>
      </header>
      {storageError && (
        <p className="inline-error" role="alert">
          {storageError}
        </p>
      )}
      {onPauseQueue && (
        <div className="queue-scheduling surface" aria-busy={scheduling}>
          <div>
            <strong>
              {paused ? "后续任务已暂停启动" : "按队列顺序自动处理"}
            </strong>
            <p className="field-hint">
              当前正在处理的任务会继续完成。等待中的任务可调整顺序。
            </p>
          </div>
          <button
            className="button outlined"
            disabled={scheduling || !queueStatus}
            onClick={() => void pauseQueue()}
          >
            {scheduling ? "正在更新…" : paused ? "继续队列" : "暂停后续任务"}
          </button>
        </div>
      )}
      {scheduleError && (
        <p className="inline-error" role="alert">
          {scheduleError}
        </p>
      )}
      <div className="queue-toolbar">
        <div className="segmented">
          <button
            className={filter === "all" ? "selected" : ""}
            aria-pressed={filter === "all"}
            onClick={() => setFilter("all")}
          >
            全部 {jobs.length}
          </button>
          <button
            className={filter === "active" ? "selected" : ""}
            aria-pressed={filter === "active"}
            onClick={() => setFilter("active")}
          >
            进行中 {active.length}
          </button>
          <button
            className={filter === "completed" ? "selected" : ""}
            aria-pressed={filter === "completed"}
            onClick={() => setFilter("completed")}
          >
            已完成 {completed.length}
          </button>
          <button
            className={filter === "failed" ? "selected" : ""}
            aria-pressed={filter === "failed"}
            onClick={() => setFilter("failed")}
          >
            失败 / 中断 {failed.length}
          </button>
        </div>
        <span className="subtle-label">
          {active.length ? `${active.length} 项等待或执行中` : "工作台准备就绪"}
        </span>
      </div>
      {visible.length ? (
        <div className="job-list">
          {orderQueueJobs(visible, pendingIDs).map((job) => (
            <JobCard
              key={job.id}
              job={job}
              cancel={cancel}
              reuse={reuse}
              rank={pendingIDs.indexOf(job.id)}
              pendingCount={pendingIDs.length}
              paused={paused}
              move={onMoveQueued}
              compareSample={compareSample}
            />
          ))}
        </div>
      ) : (
        <section className="empty-state surface">
          <Icon name="queue" />
          <h2>{filter === "all" ? "这里还很安静" : "没有符合条件的任务"}</h2>
          <p>
            {filter === "all"
              ? "创建压缩任务后，就可以在这里查看进度。"
              : "切换筛选条件，查看其他任务。"}
          </p>
          {filter === "all" && (
            <button className="button primary" onClick={compress}>
              创建压缩任务
              <Icon name="arrow" />
            </button>
          )}
        </section>
      )}
    </div>
  );
}
