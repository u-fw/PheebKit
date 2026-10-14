import { useState } from "react";
import type { EditorPreferences as Preferences } from "../bridge";
import { errorMessage } from "../format";
const fields: {
  key: Exclude<keyof Preferences, "autosave">;
  label: string;
  min: number;
  max: number;
}[] = [
  { key: "intervalSeconds", label: "草稿保存间隔（秒）", min: 5, max: 300 },
  { key: "draftLimit", label: "历史草稿数量", min: 1, max: 20 },
  { key: "retentionDays", label: "历史草稿保留天数", min: 1, max: 90 },
  {
    key: "recentLimit",
    label: "最近项目数量（0 为关闭记录）",
    min: 0,
    max: 50,
  },
  { key: "undoLimit", label: "当前文件撤销步数", min: 10, max: 200 },
];
export function EditorPreferences({
  value,
  save,
}: {
  value: Preferences;
  save: (value: Preferences) => Promise<void>;
}) {
  const [form, setForm] = useState(value);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const invalid = fields.some(
    (f) =>
      !Number.isInteger(form[f.key]) ||
      form[f.key] < f.min ||
      form[f.key] > f.max,
  );
  async function submit() {
    setBusy(true);
    setMessage("");
    try {
      await save(form);
      setMessage("编辑与恢复设置已保存。");
    } catch (cause) {
      setMessage(`保存失败：${errorMessage(cause)}`);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="surface preferences-card">
      <div className="section-title">
        <h2>编辑与恢复</h2>
        <span className="subtle-label">仅保存在本机</span>
      </div>
      <label className="toggle-row">
        <input
          type="checkbox"
          checked={form.autosave}
          onChange={(e) => setForm({ ...form, autosave: e.target.checked })}
        />
        自动保存当前项目草稿
      </label>
      <div className="field-grid">
        {fields.map((f) => (
          <label className="field-label" key={f.key}>
            {f.label}
            <input
              type="number"
              min={f.min}
              max={f.max}
              step="1"
              value={form[f.key]}
              onChange={(e) =>
                setForm({ ...form, [f.key]: Number(e.target.value) })
              }
            />
            <span className="field-hint">
              范围 {f.min}–{f.max}
            </span>
          </label>
        ))}
      </div>
      <p className="field-hint">
        草稿不会覆盖 .ffproject
        或自动开始任务；保留源文件路径和处理设置。关闭自动保存会保留已有草稿。缩短保留时间或数量会在保存设置时清理过期/较旧的历史草稿，活动窗口的草稿会保留。
      </p>
      <button
        className="button primary"
        disabled={busy || invalid}
        onClick={() => void submit()}
      >
        {busy ? "保存中…" : "保存编辑设置"}
      </button>
      {message && (
        <p className="field-hint" role="status">
          {message}
        </p>
      )}
    </section>
  );
}
