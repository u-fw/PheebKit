import { t, getLanguage } from "../i18n";
import { useState } from "react";
import type { ProjectLibrary as Library } from "../bridge";
import { API } from "../bridge";
import { errorMessage } from "../format";

export function ProjectLibrary({
  library,
  connected,
  refresh,
  openRecent,
  restore,
}: {
  library: Library | null;
  connected: boolean;
  refresh: () => Promise<void>;
  openRecent: (id: string) => void;
  restore: (id: string) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function remove(id?: string) {
    setBusy(true);
    setError("");
    try {
      if (id) await API.removeRecoveryDraft(id);
      else await API.clearRecentProjects();
      await refresh();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="surface preferences-card project-library">
      <div className="section-title">
        <h2>{t("继续工作")}</h2>
        <button
          className="button text"
          disabled={!connected || busy}
          onClick={() => {
            setError("");
            void refresh().catch((cause) => setError(errorMessage(cause)));
          }}
        >
          {t("刷新")}
        </button>
      </div>
      {(error || library?.error) && (
        <p className="inline-error" role="alert">
          {t(error || library?.error)}
        </p>
      )}
      <h3>{t("自动草稿")}</h3>
      <p className="field-hint">
        {t(
          "恢复后先检查设置并保存项目。活动窗口的草稿不能恢复或删除；这里只保留当前文件的有效项目设置。",
        )}
      </p>
      {library?.drafts.length ? (
        <ul className="project-library-list">
          {library.drafts.map((d) => (
            <li key={d.id}>
              <div>
                <strong title={d.source}>
                  {d.source.split(/[\\/]/).pop() || t("未命名媒体")}
                </strong>
                <small>
                  {new Date(d.updatedAt).toLocaleString(getLanguage())}{" "}
                  {t(d.active ? " · 其他窗口正在编辑" : "")}
                </small>
              </div>
              <div className="action-row">
                <button
                  className="button outlined"
                  disabled={busy || !connected || d.active}
                  onClick={() => restore(d.id)}
                >
                  {t("恢复草稿")}
                </button>
                <button
                  className="button text"
                  disabled={busy || !connected || d.active}
                  onClick={() => void remove(d.id)}
                >
                  {t("删除草稿")}
                </button>
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <p className="field-hint">{t("没有待恢复的历史草稿。")}</p>
      )}
      <h3>{t("最近项目")}</h3>
      {library?.recent.length ? (
        <>
          <ul className="project-library-list">
            {library.recent.map((p) => (
              <li key={p.id}>
                <div>
                  <strong title={p.path}>{p.path.split(/[\\/]/).pop()}</strong>
                  <small>{p.path}</small>
                </div>
                <button
                  className="button outlined"
                  disabled={busy || !connected}
                  onClick={() => openRecent(p.id)}
                >
                  {t("打开项目")}
                </button>
              </li>
            ))}
          </ul>
          <button
            className="button text"
            disabled={busy || !connected}
            onClick={() => void remove()}
          >
            {t("清空最近项目记录")}
          </button>
        </>
      ) : (
        <p className="field-hint">
          {t("打开或保存 .ffproject 后会记录在这里。")}
        </p>
      )}
    </section>
  );
}
