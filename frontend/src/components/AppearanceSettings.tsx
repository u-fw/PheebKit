import { useState } from "react";
import { defaultAppearance, seedPattern, useAppearance } from "../appearance";
import { t } from "../i18n";
import { Icon } from "./Icon";

const palettes = [
  { seed: "#236956", name: "菲比绿" },
  { seed: "#6750a4", name: "鸢尾紫" },
  { seed: "#315da8", name: "晴空蓝" },
  { seed: "#8b4a68", name: "蔷薇粉" },
  { seed: "#805600", name: "琥珀金" },
] as const;

export function AppearanceSettings({
  systemAccent,
  systemAccentError,
  systemAccentSupported,
  refreshSystemAccent,
}: {
  systemAccent: string;
  systemAccentError: "" | "invalid" | "read";
  systemAccentSupported: boolean;
  refreshSystemAccent: () => Promise<void>;
}) {
  const { preferences, persistent, recovered, setAppearance } = useAppearance();
  const [seedDraft, setSeedDraft] = useState<string | null>(null);
  const [invalid, setInvalid] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const draft = seedDraft ?? preferences.seed;
  function applySeed() {
    if (!seedPattern.test(draft)) {
      setInvalid(true);
      return;
    }
    setAppearance({ seed: draft, seedMode: "custom" });
    setSeedDraft(null);
    setInvalid(false);
  }
  async function refreshAccent() {
    setRefreshing(true);
    try {
      await refreshSystemAccent();
    } finally {
      setRefreshing(false);
    }
  }
  return (
    <div className="appearance-controls">
      <fieldset>
        <legend>{t("明暗模式")}</legend>
        <div className="segmented">
          {(
            [
              ["system", "跟随系统"],
              ["light", "浅色"],
              ["dark", "深色"],
            ] as const
          ).map(([mode, label]) => (
            <button
              key={mode}
              type="button"
              aria-pressed={preferences.mode === mode}
              className={preferences.mode === mode ? "selected" : ""}
              onClick={() => setAppearance({ mode })}
            >
              {t(label)}
            </button>
          ))}
        </div>
      </fieldset>
      <fieldset>
        <legend>{t("主题色")}</legend>
        <p className="field-hint">
          {t("从主题色生成 Material 明暗配色，编辑内容保持不变。")}
        </p>
        <div className="palette-options">
          <button
            className={`palette-option system-palette ${preferences.seedMode === "system" ? "selected" : ""}`}
            type="button"
            aria-pressed={preferences.seedMode === "system"}
            disabled={!systemAccentSupported}
            onClick={() => {
              setAppearance({ seedMode: "system" });
              setSeedDraft(null);
              setInvalid(false);
            }}
          >
            <span
              className="palette-dot system-palette-dot"
              style={{ backgroundColor: systemAccent }}
              aria-hidden="true"
            />
            <span>
              <strong>{t("跟随 Windows")}</strong>
              <small>{t("Material You 动态取色")}</small>
            </span>
            <span className="palette-selected" aria-hidden="true">
              {preferences.seedMode === "system" && <Icon name="check" />}
            </span>
          </button>
          {palettes.map((palette) => (
            <button
              className={`palette-option ${preferences.seedMode === "custom" && preferences.seed === palette.seed ? "selected" : ""}`}
              type="button"
              key={palette.seed}
              aria-pressed={
                preferences.seedMode === "custom" &&
                preferences.seed === palette.seed
              }
              onClick={() => {
                setAppearance({ seed: palette.seed, seedMode: "custom" });
                setSeedDraft(null);
                setInvalid(false);
              }}
            >
              <span
                className="palette-dot"
                style={{ backgroundColor: palette.seed }}
                aria-hidden="true"
              />
              {t(palette.name)}
              <span className="palette-selected" aria-hidden="true">
                {preferences.seedMode === "custom" &&
                  preferences.seed === palette.seed && <Icon name="check" />}
              </span>
            </button>
          ))}
        </div>
        <div className="system-accent-row">
          <div>
            <p className="field-hint">
              {systemAccentSupported
                ? t(
                    "当前 Windows 强调色 {{0}}。切换系统主题后，可重新读取并立即更新整套 Material 配色。",
                    [systemAccent],
                  )
                : t(
                    "Windows 强调色仅在原生桌面应用中可用；当前继续使用自定义色。",
                  )}
            </p>
            {systemAccentError && (
              <p className="inline-error" role="alert">
                {t(
                  systemAccentError === "invalid"
                    ? "Windows 返回了无效的强调色。"
                    : "无法读取 Windows 强调色。",
                )}
              </p>
            )}
          </div>
          <button
            className="button text"
            type="button"
            disabled={!systemAccentSupported || refreshing}
            onClick={() => void refreshAccent()}
          >
            <Icon name="refresh" />
            {t(refreshing ? "正在读取…" : "重新读取系统色")}
          </button>
        </div>
        <label className="field-label" htmlFor="theme-seed">
          {t("自定义主题色")}
        </label>
        <div className="seed-field">
          <input
            id="theme-seed"
            value={draft}
            maxLength={7}
            aria-invalid={invalid || undefined}
            aria-describedby="theme-seed-help"
            spellCheck={false}
            onChange={(event) => {
              setSeedDraft(event.target.value);
              setInvalid(false);
            }}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault();
                applySeed();
              }
            }}
          />
          <button className="button tonal" onClick={applySeed}>
            {t("应用颜色")}
          </button>
        </div>
        <p
          id="theme-seed-help"
          className={invalid ? "inline-error" : "field-hint"}
          role={invalid ? "alert" : undefined}
        >
          {t(
            invalid
              ? "请输入六位十六进制颜色，例如 #236956。"
              : "六位十六进制颜色，例如 #236956。",
          )}
        </p>
      </fieldset>
      <fieldset>
        <legend>{t("界面密度")}</legend>
        <div className="segmented">
          {(
            [
              ["comfortable", "舒适"],
              ["compact", "紧凑"],
            ] as const
          ).map(([density, label]) => (
            <button
              key={density}
              type="button"
              aria-pressed={preferences.density === density}
              className={preferences.density === density ? "selected" : ""}
              onClick={() => setAppearance({ density })}
            >
              {t(label)}
            </button>
          ))}
        </div>
        <p className="field-hint">
          {t("紧凑模式减少留白，保留清晰的文字、焦点和操作区域。")}
        </p>
      </fieldset>
      {!persistent && (
        <p className="inline-error" role="status">
          {t("无法保存外观偏好，本次修改仍然有效。")}
        </p>
      )}
      {recovered && (
        <p className="notice" role="status">
          {t("原外观偏好无法读取，已使用默认值；原记录尚未覆盖。")}
        </p>
      )}
      <button
        className="button text"
        onClick={() => {
          setAppearance(defaultAppearance);
          setSeedDraft(null);
          setInvalid(false);
        }}
      >
        {t("恢复默认外观")}
      </button>
    </div>
  );
}
