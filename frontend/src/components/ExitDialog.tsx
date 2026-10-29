import { t } from "../i18n";
import { useEffect, useRef, useState } from "react";
import { API } from "../bridge";
import { errorMessage } from "../format";

export function ExitDialog({
  request,
  dismiss,
  save,
}: {
  request: {
    jobs: number;
    downloading: boolean;
    unsaved?: boolean;
    shutdownStarted?: boolean;
    historySaveFailed?: boolean;
  };
  dismiss: () => void;
  save?: () => Promise<boolean>;
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
  async function quit(saveFirst = false, allowHistoryLoss = false) {
    setQuitting(true);
    setError("");
    try {
      if (saveFirst && (!save || !(await save()))) {
        setQuitting(false);
        return;
      }
      await API.confirmQuit(allowHistoryLoss);
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
        if (!quitting && !request.shutdownStarted) returnToWorkspace();
      }}
    >
      <h2 id="exit-title">
        {t(
          request.historySaveFailed
            ? "任务历史尚未保存"
            : request.shutdownStarted
              ? "正在停止并退出…"
              : request.unsaved
                ? "当前项目尚未保存"
                : "仍有工作正在进行",
        )}
      </h2>
      <p id="exit-description">
        {t(request.jobs > 0 ? `${request.jobs} 个任务仍在等待或处理中。` : "")}{" "}
        {t(request.downloading ? "FFmpeg 环境补全仍在进行。" : "")}
        {(request.jobs > 0 || request.downloading) &&
          t("退出会取消这些操作，已完成的文件和历史记录会保留。")}
        {request.unsaved && t("当前编辑的修改尚未保存为项目，退出后将丢失。")}
        {request.historySaveFailed &&
          t(
            "任务已停止，但本次任务历史未能写入磁盘。请重试保存，或明确选择不保存历史后退出；已生成的输出文件不受影响。",
          )}
      </p>
      {error && (
        <p role="alert" className="inline-error">
          {t(error)}
        </p>
      )}
      <div className="action-row">
        <button
          className="button outlined"
          onClick={returnToWorkspace}
          disabled={quitting || request.shutdownStarted}
          autoFocus
        >
          {t("返回工作台")}
        </button>
        <button
          className="button primary"
          onClick={() => void quit()}
          disabled={quitting}
        >
          {t(
            quitting
              ? "正在停止并退出…"
              : request.shutdownStarted
                ? "重试保存并退出"
                : request.unsaved
                  ? "放弃修改并退出"
                  : "取消任务并退出",
          )}
        </button>
        {request.unsaved && save && !request.shutdownStarted && (
          <button
            className="button primary"
            disabled={quitting}
            onClick={() => void quit(true)}
          >
            {t("保存项目并退出")}
          </button>
        )}
        {request.historySaveFailed && (
          <button
            className="button outlined"
            disabled={quitting}
            onClick={() => void quit(false, true)}
          >
            {t("不保存任务历史并退出")}
          </button>
        )}
      </div>
    </dialog>
  );
}
