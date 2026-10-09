import { useState } from "react";
import type { Job } from "../bridge";
import { formatBytes, formatDuration, savedPercent } from "../format";
import { Icon } from "./Icon";

const stateNames: Record<string, string> = {
  queued: "等待中",
  preparing: "准备中",
  running: "处理中",
  cancelling: "正在取消",
  cancelled: "已取消",
  completed: "已完成",
  failed: "失败",
};
const activeStates = new Set(["queued", "preparing", "running", "cancelling"]);

function JobCard({ job, cancel }: { job: Job; cancel: (id: string) => void }) {
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
              <dt>Speed</dt>
              <dd>{job.speed ? `${job.speed.toFixed(2)}×` : "—"}</dd>
            </div>
            <div>
              <dt>Out time</dt>
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
        <p className="inline-error" role="alert">
          {job.error}
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
}: {
  jobs: Job[];
  cancel: (id: string) => void;
  refresh: () => void;
  compress: () => void;
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
  return (
    <div className="page queue-page">
      <header className="page-heading">
        <div>
          <div className="section-eyebrow">JOB QUEUE</div>
          <h1>每一项任务，都有迹可循。</h1>
          <p>实时进度、执行日志和实际处理结果。</p>
        </div>
        <button className="button outlined" onClick={refresh}>
          刷新队列
        </button>
      </header>
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
            <JobCard key={job.id} job={job} cancel={cancel} />
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
