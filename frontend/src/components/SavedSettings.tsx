import { useEffect, useState } from "react";
import { API } from "../bridge";
import type { Settings, WorkspacePreferences } from "../bridge";
import { errorMessage } from "../format";
import { MaterialSelect } from "./MaterialSelect";

export function SavedSettings({
  settings,
  apply,
  onSaved,
}: {
  settings: Settings;
  apply: (value: Settings) => void;
  onSaved?: (value: Settings) => void;
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
  }, []);
  async function save(named: boolean) {
    setBusy(true);
    setMessage("");
    try {
      const next = await API.saveSettings(settings, named ? name.trim() : "");
      setPreferences(next);
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
      setPreferences(await API.deletePreset(selected));
      setSelected("");
      setMessage("常用配置已删除。");
    } catch (cause) {
      setMessage(`删除失败：${errorMessage(cause)}`);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="surface settings-card saved-settings">
      <div className="section-title">
        <h2>常用设置</h2>
        <span className="subtle-label">保存在本机</span>
      </div>
      <div className="action-row">
        <button
          className="button outlined"
          onClick={() => void save(false)}
          disabled={busy}
        >
          将当前设置设为默认
        </button>
        <button
          className="button text"
          disabled={!preferences.lastSettings}
          onClick={() => {
            if (preferences.lastSettings) apply(preferences.lastSettings);
          }}
        >
          应用默认设置
        </button>
      </div>
      <div className="saved-settings-row">
        <MaterialSelect
          label="已保存配置"
          value={selected}
          options={[
            { value: "", label: "选择常用配置" },
            ...preferences.presets.map((p) => ({ value: p.id, label: p.name })),
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
          应用
        </button>
        <button
          className="button text"
          disabled={!selected || busy}
          onClick={() => void remove()}
        >
          删除配置
        </button>
      </div>
      <label className="field-label" htmlFor="preset-name">
        配置名称
      </label>
      <div className="saved-settings-row">
        <input
          id="preset-name"
          value={name}
          maxLength={40}
          onChange={(event) => setName(event.target.value)}
          placeholder="例如：聊天分享 · 1080p"
        />
        <button
          className="button outlined"
          disabled={busy || !name.trim()}
          onClick={() => void save(true)}
        >
          保存配置
        </button>
      </div>
      <p className="field-hint">
        保存编码、质量、音频和高级参数。剪辑范围、源文件和输出路径不会保存到常用配置。
      </p>
      {message && (
        <p role="status" className="field-hint">
          {message}
        </p>
      )}
    </section>
  );
}
