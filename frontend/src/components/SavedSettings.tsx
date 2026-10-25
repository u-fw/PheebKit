import { t } from "../i18n";
import { useEffect, useState } from "react";
import { API } from "../bridge";
import type { Settings, WorkspacePreferences } from "../bridge";
import { errorMessage } from "../format";
import { MaterialSelect } from "./MaterialSelect";

export function SavedSettings({
  settings,
  apply,
  onSaved,
  onPreferences,
}: {
  settings: Settings;
  apply: (value: Settings) => void;
  onSaved?: (value: Settings) => void;
  onPreferences?: (value: WorkspacePreferences) => void;
}) {
  const [preferences, setPreferences] = useState<WorkspacePreferences>({
    version: 1,
    presets: [],
  });
  const [selected, setSelected] = useState("");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  useEffect(() => {
    let alive = true;
    void API.getWorkspace().then(
      (value) => {
        if (alive) {
          setPreferences(value.preferences);
          onPreferences?.(value.preferences);
          setMessage(value.error);
        }
      },
      (cause) => {
        if (alive) setMessage(errorMessage(cause));
      },
    );
    return () => {
      alive = false;
    };
  }, [onPreferences]);
  async function save(named: boolean) {
    setBusy(true);
    setMessage("");
    try {
      const next = await API.saveSettings(settings, named ? name.trim() : "");
      setPreferences(next);
      onPreferences?.(next);
      if (next.lastSettings) onSaved?.(next.lastSettings);
      setMessage(
        named
          ? "常用配置已保存（同名配置会更新）。"
          : "默认设置已保存，下次导入媒体时使用。",
      );
    } catch (cause) {
      setMessage(`保存失败：${errorMessage(cause)}`);
    } finally {
      setBusy(false);
    }
  }
  async function remove() {
    setBusy(true);
    setMessage("");
    try {
      const next = await API.deletePreset(selected);
      setPreferences(next);
      onPreferences?.(next);
      setSelected("");
      setMessage("常用配置已删除。");
    } catch (cause) {
      setMessage(`删除失败：${errorMessage(cause)}`);
    } finally {
      setBusy(false);
    }
  }
  async function share(importing: boolean) {
    if (busy) return;
    setBusy(true);
    setMessage("");
    try {
      if (importing) {
        const next = await API.importPreset();
        if (next) {
          setPreferences(next);
          onPreferences?.(next);
          setMessage("配置已导入，请选择并应用。当前设置和默认值未更改。");
        }
      } else {
        const path = await API.exportPreset(selected);
        if (path) setMessage(`配置已导出：${path}`);
      }
    } catch (cause) {
      setMessage(
        importing
          ? `导入失败：${errorMessage(cause)}`
          : `导出失败：${errorMessage(cause)}`,
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="surface settings-card saved-settings">
      <div className="section-title">
        <h2 tabIndex={-1}>{t("常用设置")}</h2>
        <span className="subtle-label">{t("保存在本机")}</span>
      </div>
      <div className="action-row">
        <button
          className="button outlined"
          onClick={() => void save(false)}
          disabled={busy}
        >
          {t("将当前设置设为默认")}
        </button>
        <button
          className="button text"
          disabled={!preferences.lastSettings}
          onClick={() => {
            if (preferences.lastSettings) apply(preferences.lastSettings);
          }}
        >
          {t("应用默认设置")}
        </button>
      </div>
      <div className="saved-settings-row">
        <MaterialSelect
          label={t("已保存配置")}
          value={selected}
          options={[
            { value: "", label: "选择常用配置" },
            ...preferences.presets.map((p) => ({
              value: p.id,
              label: p.name,
              translate: false,
            })),
          ]}
          onChange={setSelected}
        />
        <button
          className="button outlined"
          disabled={!selected}
          onClick={() => {
            const preset = preferences.presets.find((p) => p.id === selected);
            if (preset) apply(preset.settings);
          }}
        >
          {t("应用")}
        </button>
        <button
          className="button text"
          disabled={!selected || busy}
          onClick={() => void remove()}
        >
          {t("删除配置")}
        </button>
      </div>
      <label className="field-label" htmlFor="preset-name">
        {t("配置名称")}
      </label>
      <div className="saved-settings-row">
        <input
          id="preset-name"
          value={name}
          maxLength={40}
          onChange={(event) => setName(event.target.value)}
          placeholder={t("例如：聊天分享 · 1080p")}
        />
        <button
          className="button outlined"
          disabled={busy || !name.trim()}
          onClick={() => void save(true)}
        >
          {t("保存配置")}
        </button>
      </div>
      <p className="field-hint">
        {t(
          "保存处理方式、编码、音量与旋转等通用设置。剪辑、多片段、字幕烧录、轨道编号、裁剪边距及文件路径不保存到配置；这些编辑可保存到项目文件。",
        )}
      </p>
      <div className="action-row">
        <button
          className="button outlined"
          disabled={busy}
          onClick={() => void share(true)}
        >
          {t("导入配置")}
        </button>
        <button
          className="button text"
          disabled={busy || !selected}
          onClick={() => void share(false)}
        >
          {t("导出所选配置")}
        </button>
      </div>
      {message && (
        <p role="status" className="field-hint">
          {t(message)}
        </p>
      )}
    </section>
  );
}
