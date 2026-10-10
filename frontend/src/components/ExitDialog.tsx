import { useEffect, useRef, useState } from "react";
import { API } from "../bridge";
import { errorMessage } from "../format";

export function ExitDialog({
  request,
  dismiss,
}: {
  request: { jobs: number; downloading: boolean };
  dismiss: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [quitting, setQuitting] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    dialog.current?.showModal();
  }, []);
  function returnToWorkspace() {
    dialog.current?.close();
    dismiss();
  }
  async function quit() {
    setQuitting(true);
    try {
      await API.confirmQuit();
    } catch (cause) {
      setError(errorMessage(cause));
      setQuitting(false);
    }
  }
  return (
    <dialog
      ref={dialog}
      className="surface exit-dialog"
      aria-labelledby="exit-title"
      aria-describedby="exit-description"
      onCancel={(event) => {
        event.preventDefault();
        if (!quitting) returnToWorkspace();
      }}
    >
      <h2 id="exit-title">仍有工作正在进行</h2>
      <p id="exit-description">
        {request.jobs > 0 ? `${request.jobs} 个任务仍在等待或处理中。` : ""}
        {request.downloading ? "FFmpeg 环境补全仍在进行。" : ""}
        退出会取消这些操作，已完成的文件和历史记录会保留。
      </p>
      {error && (
        <p role="alert" className="inline-error">
          {error}
        </p>
      )}
      <div className="action-row">
        <button
          className="button outlined"
          onClick={returnToWorkspace}
          disabled={quitting}
          autoFocus
        >
          返回工作台
        </button>
        <button
          className="button primary"
          onClick={() => void quit()}
          disabled={quitting}
        >
          {quitting ? "正在停止并退出…" : "取消任务并退出"}
        </button>
      </div>
    </dialog>
  );
}
