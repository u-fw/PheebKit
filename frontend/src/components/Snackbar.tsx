import { useEffect, useRef, useState } from "react";
import { t } from "../i18n";
import type { SnackbarNotice } from "../snackbar";

export function Snackbar({
  notice,
  dismiss,
}: {
  notice: SnackbarNotice;
  dismiss: (id?: number) => void;
}) {
  const [pointerInside, setPointerInside] = useState(false);
  const [focusInside, setFocusInside] = useState(false);
  const [actionBusy, setActionBusy] = useState(false);
  const [actionFailed, setActionFailed] = useState(false);
  const remaining = useRef(notice.duration ?? 6000);
  const paused = pointerInside || focusInside || actionBusy || actionFailed;

  useEffect(() => {
    if (paused) return;
    const started = performance.now();
    const timer = window.setTimeout(
      () => dismiss(notice.id),
      remaining.current,
    );
    return () => {
      window.clearTimeout(timer);
      remaining.current = Math.max(
        0,
        remaining.current - (performance.now() - started),
      );
    };
  }, [dismiss, notice.id, paused]);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key !== "Escape" || actionBusy) return;
      event.preventDefault();
      dismiss(notice.id);
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [actionBusy, dismiss, notice.id]);

  async function runAction() {
    if (!notice.action || actionBusy) return;
    setActionBusy(true);
    setActionFailed(false);
    try {
      await notice.action.run();
      dismiss(notice.id);
    } catch {
      setActionFailed(true);
    } finally {
      setActionBusy(false);
    }
  }

  return (
    <div className="snackbar-host" aria-live="polite" aria-atomic="true">
      <section
        className="snackbar"
        role="status"
        aria-busy={actionBusy}
        onPointerEnter={() => setPointerInside(true)}
        onPointerLeave={() => setPointerInside(false)}
        onFocusCapture={() => setFocusInside(true)}
        onBlurCapture={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget as Node | null))
            setFocusInside(false);
        }}
      >
        <div className="snackbar-copy">
          <span>{notice.message}</span>
          {actionFailed && (
            <small className="snackbar-error">
              {t("操作未完成，请稍后重试。")}
            </small>
          )}
        </div>
        {notice.action && (
          <button
            className="snackbar-action"
            type="button"
            disabled={actionBusy}
            onClick={() => void runAction()}
          >
            {t(actionBusy ? "正在撤销…" : notice.action.label)}
          </button>
        )}
      </section>
    </div>
  );
}
