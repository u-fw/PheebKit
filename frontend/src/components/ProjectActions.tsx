import { t } from "../i18n";
import { useState } from "react";
import { API } from "../bridge";
import type { Job, Project } from "../bridge";
import { errorMessage } from "../format";
import { operationOf } from "../project";
import { MaterialSelect } from "./MaterialSelect";

export function ProjectActions({
  project,
  connected,
  load,
  onJob,
  showQueue,
  onSaved,
  dirty = false,
  autosaveMessage,
  autosaveError = false,
}: {
  project: Project;
  connected: boolean;
  load: () => Promise<void>;
  onJob: (job: Job) => void;
  showQueue: () => void;
  onSaved?: (project: Project) => void;
  dirty?: boolean;
  autosaveMessage?: string;
  autosaveError?: boolean;
}) {
  const [busy, setBusy] = useState(false);
  const [seconds, setSeconds] = useState("10");
  const [message, setMessage] = useState("");
  async function perform(action: "save" | "load" | "sample") {
    if (busy) return;
    setBusy(true);
    setMessage("");
    try {
      if (action === "load") await load();
      else if (action === "save") {
        const path = await API.saveProject(project);
        if (path) {
          setMessage(`项目已保存：${path}`);
          onSaved?.(project);
        }
      } else {
        onJob(await API.startSample(project, Number(seconds)));
        showQueue();
      }
    } catch (cause) {
      setMessage(`操作失败：${errorMessage(cause)}`);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="surface settings-card project-actions">
      <div className="section-title">
        <h2>{t("项目与试压")}</h2>
        <span className="subtle-label" role="status">
          {t(dirty ? "未保存修改" : "可继续编辑")}
        </span>
      </div>
      <div className="action-row">
        <button
          className="button outlined"
          disabled={busy || !connected}
          onClick={() => void perform("save")}
        >
          {t("保存项目")}
        </button>
        <button
          className="button text"
          disabled={busy || !connected}
          onClick={() => void perform("load")}
        >
          {t("打开项目")}
        </button>
      </div>
      {autosaveMessage && (
        <p
          className={autosaveError ? "inline-error" : "field-hint"}
          role={autosaveError ? "alert" : "status"}
        >
          {t(autosaveMessage)}
        </p>
      )}
      <p className="field-hint">
        {t(
          "项目保存当前文件、轨道、剪辑范围和全部设置，打开后重新检查源文件，由你确认执行。",
        )}
      </p>
      {operationOf(project.settings) === "compress" && (
        <>
          <div className="saved-settings-row">
            <MaterialSelect
              label={t("样片时长")}
              editable
              value={seconds}
              options={Array.from({ length: 30 }, (_, index) => ({
                value: String(index + 1),
                label: `${index + 1} 秒`,
              }))}
              onChange={setSeconds}
            />
            <button
              className="button outlined"
              disabled={busy || !connected || !project.output.uri}
              onClick={() => void perform("sample")}
            >
              {t(busy ? "处理中…" : "生成样片")}
            </button>
          </div>
          <p className="field-hint">
            {t(
              "从当前剪辑起点（多片段使用列表第一段，默认开头）试压，最长不超过该段范围。样片保存在输出文件夹，完成后可在队列比较画面；为准确截取，Copy 音频会在样片中改为 AAC。声音处理会应用到样片，但整片淡入淡出不会应用；动态归一化可能与整片不同。样片体积和速度只供参考。",
            )}
          </p>
        </>
      )}
      {message && (
        <p className="field-hint" role="status">
          {t(message)}
        </p>
      )}
    </section>
  );
}
