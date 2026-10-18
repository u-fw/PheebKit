import { t } from "../i18n";
import { useEffect, useRef, useState } from "react";
import { API, type Media, type MediaHealth, type Settings } from "../bridge";
import { errorMessage } from "../format";
import { formatTimecode } from "../trim";
import { MaterialSelect } from "./MaterialSelect";

export function MediaRecovery({
  media,
  settings,
  connected,
  onChange,
}: {
  media: Media;
  settings: Settings;
  connected: boolean;
  onChange: (value: Settings) => void;
}) {
  const [report, setReport] = useState<MediaHealth | null>(null);
  const [mode, setMode] = useState("delete");
  const [windowSeconds, setWindowSeconds] = useState(5);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [epoch, setEpoch] = useState(0);
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  useEffect(() => {
    let stopped = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    async function poll() {
      try {
        const value = await API.getMediaHealthScan();
        if (stopped) return;
        setReport(value);
        if (value.state === "scanning" || value.state === "cancelling")
          timer = setTimeout(() => void poll(), 1000);
      } catch (e) {
        if (!stopped) setError(errorMessage(e));
      }
    }
    if (connected) void poll();
    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [connected, epoch]);
  const active = report?.state === "scanning" || report?.state === "cancelling";
  const matches =
    report?.uri === media.source.uri &&
    report.sourceFingerprint === media.sourceFingerprint;
  const completed = matches && report.state === "completed";
  const ranges = completed ? (report.damagedRanges ?? []) : [];
  const removed = ranges.reduce(
    (sum, r) => sum + r.endSeconds - r.startSeconds,
    0,
  );
  const recovery = settings.recovery;
  async function start() {
    setPending(true);
    setError("");
    try {
      const value = await API.startMediaHealthScan(
        media.source.uri,
        windowSeconds,
      );
      if (mounted.current) {
        setReport(value);
        setEpoch((v) => v + 1);
      }
    } catch (e) {
      if (mounted.current) setError(errorMessage(e));
    } finally {
      if (mounted.current) setPending(false);
    }
  }
  async function cancel() {
    if (!report) return;
    setPending(true);
    setError("");
    try {
      await API.cancelMediaHealthScan(report.id);
    } catch (e) {
      if (mounted.current) setError(errorMessage(e));
    } finally {
      if (mounted.current) setPending(false);
    }
  }
  return (
    <section className="surface settings-card media-recovery">
      <div className="section-title">
        <h2>{t("解码检查与恢复")}</h2>
        <span className="subtle-label">{t("生成新文件")}</span>
      </div>
      <p className="field-hint">
        {t(
          "按选定窗口逐段检查，包含解码所需的参考帧。失败窗口可能包含正常画面，无法恢复已丢失的内容，也不能检出所有视觉异常。",
        )}
      </p>
      <MaterialSelect
        label={t("扫描窗口")}
        value={String(windowSeconds)}
        onChange={(value) => setWindowSeconds(Number(value))}
        disabled={active || pending}
        options={[1, 2, 5, 10, 30].map((value) => ({
          value: String(value),
          label: t("{{0}} 秒窗口{{1}}", [
            value,
            t(value === 5 ? "（默认）" : ""),
          ]),
        }))}
      />
      <p className="field-hint">
        {t("窗口越小定位越细，但扫描更慢。最多 4320 个窗口、6 小时视频。")}
      </p>
      <div className="recovery-actions">
        <button
          className="button outlined"
          disabled={!connected || active || pending}
          onClick={() => void start()}
        >
          {t(pending && !active ? "正在启动…" : "扫描当前视频")}
        </button>
        {active && (
          <button
            className="button outlined"
            disabled={pending || report.state === "cancelling"}
            onClick={() => void cancel()}
          >
            {t(report.state === "cancelling" ? "正在取消…" : "取消扫描")}
          </button>
        )}
      </div>
      {active && (
        <div role="status">
          <p>
            {t(matches ? "正在检查当前视频" : "另一媒体正在扫描")} ·{t(" ")}
            {Math.floor(report.percent)}%
          </p>
          <progress
            aria-label={t("解码扫描进度")}
            max={100}
            value={report.percent}
          />
        </div>
      )}
      {matches && report.state === "failed" && (
        <p className="inline-error" role="alert">
          {t("扫描未完成：")} {t(report.error)} {t("。不使用不完整结果。")}
        </p>
      )}
      {matches && report.state === "cancelled" && (
        <p role="status">{t("扫描已取消，不使用不完整结果。")}</p>
      )}
      {completed && (
        <p role="status">
          {t("本次窗口")} {report.windowSeconds} {t("秒。")}{" "}
          {t(
            ranges.length
              ? `发现 ${ranges.length} 个失败区间，共 ${formatTimecode(removed)}。请确认后应用恢复方案。`
              : "未发现解码错误；这不代表不存在画面异常。",
          )}
        </p>
      )}
      {ranges.length > 0 && (
        <>
          <ol className="recovery-ranges">
            {ranges.map((r) => (
              <li key={r.startSeconds}>
                {t(formatTimecode(r.startSeconds))} →{t(" ")}
                {t(formatTimecode(r.endSeconds))}
              </li>
            ))}
          </ol>
          <MaterialSelect
            label={t("恢复策略")}
            value={mode}
            onChange={setMode}
            options={[
              { value: "delete", label: "删除失败区间（默认）" },
              { value: "black", label: "保留时长 · 黑画面与静音" },
            ]}
          />
          <p className="field-hint">
            {t(
              mode === "delete"
                ? `同时删除这些区间的画面与音频，预计保留 ${formatTimecode(Math.max(0, media.durationSeconds - removed))}。`
                : "以黑画面与静音替代失败区间；音轨统一为 48 kHz 立体声，预计保留原时长。",
            )}
            {t(" ")}
            {t(
              "应用后清除剪辑、字幕、声音处理和高级参数；源文件不变。请先恢复，再进行其他编辑。",
            )}
          </p>
          <button
            className="button outlined"
            disabled={active || pending || removed >= media.durationSeconds}
            onClick={() =>
              onChange({
                ...settings,
                trim: null,
                segments: null,
                subtitleBurn: null,
                audioProcessing: null,
                rawArguments: [],
                subtitleSelection: { mode: "none", indices: [] },
                audio:
                  settings.audio === "copy"
                    ? settings.codec === "av1"
                      ? "opus"
                      : "aac"
                    : settings.audio,
                recovery: {
                  mode,
                  sourceFingerprint: media.sourceFingerprint ?? "",
                  damagedRanges: ranges.map((r) => ({ ...r })),
                },
              })
            }
          >
            {t("确认区间并应用方案")}
          </button>
          {removed >= media.durationSeconds && (
            <p className="inline-error">
              {t("没有可保留的正常区间，不能恢复原视频内容。")}
            </p>
          )}
        </>
      )}
      {recovery && (
        <div className="recovery-intent">
          <p>
            {t("当前方案：")}{" "}
            {t(recovery.mode === "delete" ? "删除" : "黑画面与静音填补")}
            {t(" ")}
            {recovery.damagedRanges?.length ?? 0}
            {t(" ")}
            {t("个区间。保存前会完整解码复检，失败则不保存。")}
          </p>
          {recovery.mode === "black" && (
            <p className="field-hint">
              {t("失败区间使用黑画面与静音；音轨统一为 48 kHz 立体声。")}
            </p>
          )}
          <ol className="recovery-ranges">
            {recovery.damagedRanges?.map((r) => (
              <li key={r.startSeconds}>
                {t(formatTimecode(r.startSeconds))} →{t(" ")}
                {t(formatTimecode(r.endSeconds))}
              </li>
            ))}
          </ol>
          <button
            className="button text"
            onClick={() => onChange({ ...settings, recovery: null })}
          >
            {t("移除恢复方案")}
          </button>
        </div>
      )}
      {error && (
        <p className="inline-error" role="alert">
          {t(error)}
        </p>
      )}
    </section>
  );
}
