import { useEffect, useState } from "react";
import type { Job } from "../bridge";
import { formatBytes, formatDuration, savedPercent } from "../format";
import { Icon } from "./Icon";
import { formatTimecode } from "../trim";
import { API } from "../bridge";
import { errorMessage } from "../format";
import { jobErrorHelp } from "../jobError";

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
}: {
  job: Job;
  cancel: (id: string) => void;
  reuse: (id: string) => Promise<void>;
}) {
  const [actionError, setActionError] = useState("");
  const [busy, setBusy] = useState(false);
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
            {stateNames[state] || job.state}
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
      {job.trim && (
        <p className="field-hint">
          剪辑并压缩 · {formatTimecode(job.trim.startSeconds)} →{" "}
          {formatTimecode(job.trim.endSeconds)}
        </p>
      )}
      {active && (
        <>
          <div className="progress-label">
            <span>
              {state === "queued"
                ? "任务等待执行"
                : state === "preparing"
                  ? "正在准备执行计划"
                  : "编码进度"}
            </span>
            <strong>{Math.round(progress * 100)}%</strong>
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
        </>
      )}
      {state === "completed" && (
        <>
          <div className="result-banner">
            <Icon name="check" />
            <span>处理完成</span>
            <strong>{savedPercent(job.inputSize, job.outputSize)}</strong>
          </div>
          {job.trim && (
            <p className="field-hint">
              体积减少包含剪掉的内容，比较基准为完整源文件。
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
            </>
          )}
          {job.project && (
            <button
              className="button outlined"
              disabled={busy}
              onClick={() => void action("reuse")}
            >
              {busy ? "处理中…" : "沿用设置再处理"}
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
}: {
  jobs: Job[];
  cancel: (id: string) => void;
  refresh: () => void;
  compress: () => void;
  reuse: (id: string) => Promise<void>;
}) {
  const [filter, setFilter] = useState<"all" | "active" | "completed">("all");
  const active = jobs.filter((job) =>
    activeStates.has(job.state.toLowerCase()),
  );
  const completed = jobs.filter(
    (job) => job.state.toLowerCase() === "completed",
  );
  const visible =
    filter === "active" ? active : filter === "completed" ? completed : jobs;
  const [storageError, setStorageError] = useState("");
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
        </div>
        <span className="subtle-label">
          {active.length ? `${active.length} 项等待或执行中` : "工作台准备就绪"}
        </span>
      </div>
      {visible.length ? (
        <div className="job-list">
          {[...visible].reverse().map((job) => (
            <JobCard key={job.id} job={job} cancel={cancel} reuse={reuse} />
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
