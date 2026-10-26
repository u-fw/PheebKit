import { t } from "../i18n";
import type { JobActivitySummary } from "../jobActivity";
import { Icon } from "./Icon";

export function JobActivityIndicator({
  summary,
  onOpen,
}: {
  summary: JobActivitySummary;
  onOpen: () => void;
}) {
  if (!summary.activeCount) return null;
  const percent = Math.round(summary.progress * 100);
  const name =
    summary.current?.input.displayName || summary.current?.input.uri || "";
  const label = summary.paused ? "队列已暂停" : "正在处理";

  return (
    <>
      <button
        className="job-activity-chip"
        type="button"
        onClick={onOpen}
        aria-label={t("打开任务队列：{{0}}，{{1}} 个任务，{{2}}%", [
          label,
          summary.activeCount,
          percent,
        ])}
      >
        <span className="job-activity-symbol" aria-hidden="true">
          <Icon name="queue" />
        </span>
        <span className="job-activity-copy">
          <strong>{t(label)}</strong>
          <small title={name}>
            {t("{{0}} 个任务 · {{1}}%", [summary.activeCount, percent])}
          </small>
        </span>
        <span className="job-activity-percent">{percent}%</span>
        <span className="job-activity-track" aria-hidden="true">
          <span
            className="job-activity-value"
            style={{ transform: `scaleX(${summary.progress})` }}
          />
        </span>
      </button>
      <span className="topbar-job-progress" aria-hidden="true">
        <span
          className={summary.paused ? "paused" : ""}
          style={{ transform: `scaleX(${summary.progress})` }}
        />
      </span>
    </>
  );
}
