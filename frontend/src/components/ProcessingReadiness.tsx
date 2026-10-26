import { t } from "../i18n";
import type {
  ProcessingReadiness as Readiness,
  ReadinessSignal,
} from "../readiness";
import { Icon } from "./Icon";

const signalIcon: Record<
  ReadinessSignal["kind"],
  Parameters<typeof Icon>[0]["name"]
> = {
  ffmpeg: "terminal",
  acceleration: "video",
  storage: "home",
};

export function ProcessingReadiness({ readiness }: { readiness: Readiness }) {
  const signals = [readiness.ffmpeg, readiness.acceleration, readiness.storage];

  return (
    <div
      className="processing-readiness"
      role="status"
      aria-live="polite"
      aria-label={t("处理环境状态")}
    >
      {signals.map((signal) => (
        <ReadinessItem key={signal.kind} signal={signal} />
      ))}
    </div>
  );
}

function ReadinessItem({ signal }: { signal: ReadinessSignal }) {
  const label = t(signal.label);
  const detail = t(signal.detail, signal.detailValues);
  return (
    <div className="readiness-signal" data-tone={signal.tone}>
      <span className="readiness-icon" aria-hidden="true">
        <Icon name={signalIcon[signal.kind]} />
      </span>
      <span className="readiness-copy">
        <strong title={label}>{label}</strong>
        <small title={detail}>{detail}</small>
      </span>
    </div>
  );
}
