import { useEffect, useRef, useState } from "react";
import { errorMessage } from "../format";
export function UnsavedDialog({
  save,
  discard,
  dismiss,
}: {
  save: () => Promise<boolean>;
  discard: () => Promise<void>;
  dismiss: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    dialog.current?.showModal();
  }, []);
  async function proceed(saving: boolean) {
    setBusy(true);
    setError("");
    try {
      if (saving && !(await save())) return;
      await discard();
      dismiss();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }
  return (
    <dialog
      ref={dialog}
      className="surface exit-dialog"
      aria-labelledby="unsaved-title"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) dismiss();
      }}
    >
      <h2 id="unsaved-title">当前项目有未保存的修改</h2>
      <p>继续会替换当前编辑。可以先保存项目，或放弃修改后继续。</p>
      {error && (
        <p className="inline-error" role="alert">
          {error}
        </p>
      )}
      <div className="action-row">
        <button
          className="button outlined"
          autoFocus
          disabled={busy}
          onClick={dismiss}
        >
          继续编辑
        </button>
        <button
          className="button text"
          disabled={busy}
          onClick={() => void proceed(false)}
        >
          放弃并继续
        </button>
        <button
          className="button primary"
          disabled={busy}
          onClick={() => void proceed(true)}
        >
          {busy ? "处理中…" : "保存项目并继续"}
        </button>
      </div>
    </dialog>
  );
}
