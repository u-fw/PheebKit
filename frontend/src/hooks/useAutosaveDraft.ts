import { useEffect, useRef, useState } from "react";
import { API, type EditorPreferences } from "../bridge";
import type { Draft } from "../draft";
import { draftFingerprint } from "../draft";
import { errorMessage } from "../format";
import { getLanguage, t } from "../i18n";

export function useAutosaveDraft(
  draft: Draft | null,
  connected: boolean,
  preferences: EditorPreferences | undefined,
) {
  const current = useRef(draft);
  const revision = useRef(0);
  const mounted = useRef(true);
  const inFlight = useRef(false);
  const lastFingerprint = useRef("");
  const [message, setMessage] = useState<string | number>("");
  const [error, setError] = useState("");
  const nextRevision = () =>
    (revision.current = Math.max(Date.now(), revision.current + 1));
  useEffect(() => {
    current.current = draft;
  }, [draft]);
  const id = draft?.project.id;
  const dirty = draft?.dirty ?? false;
  const ready = Boolean(preferences);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  useEffect(() => {
    if (!connected || !ready) return;
    const request = nextRevision();
    lastFingerprint.current = "";
    void API.clearRecoveryDraft(request).then(
      () => {
        if (mounted.current && revision.current === request) {
          setMessage("");
          setError("");
        }
      },
      (cause) => {
        if (mounted.current && revision.current === request)
          setError(`草稿清理失败，已有记录仍保留：${errorMessage(cause)}`);
      },
    );
  }, [connected, id, dirty, ready]);
  useEffect(() => {
    if (!connected || !preferences?.autosave) return;
    const timer = setInterval(() => {
      const value = current.current;
      if (!value?.dirty || inFlight.current) return;
      const fingerprint = draftFingerprint(value.project);
      if (fingerprint === lastFingerprint.current) return;
      const request = nextRevision();
      const snapshot = structuredClone(value.project);
      inFlight.current = true;
      setMessage("正在自动保存草稿…");
      void API.saveRecoveryDraft(snapshot, request)
        .then(
          () => {
            if (revision.current === request) {
              lastFingerprint.current = fingerprint;
              if (mounted.current) {
                setMessage(Date.now());
                setError("");
              }
            }
          },
          (cause) => {
            if (mounted.current && revision.current === request)
              setError(`自动草稿保存失败：${errorMessage(cause)}`);
          },
        )
        .finally(() => {
          inFlight.current = false;
        });
    }, preferences.intervalSeconds * 1000);
    return () => {
      clearInterval(timer);
    };
  }, [connected, preferences?.autosave, preferences?.intervalSeconds]);
  return {
    message: !preferences
      ? "正在读取恢复设置…"
      : preferences.autosave
        ? (typeof message === "number"
            ? t("最近自动保存：{{0}}", [
                new Date(message).toLocaleTimeString(getLanguage()),
              ])
            : message) ||
          `自动草稿：每 ${preferences.intervalSeconds} 秒检查修改`
        : "自动草稿已关闭",
    error,
  };
}
