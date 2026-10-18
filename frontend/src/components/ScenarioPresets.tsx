import { t } from "../i18n";
import { useState } from "react";
import type { Media, Settings } from "../bridge";
import { buildScenario, scenarioPresets } from "../scenarioPresets";
import { MaterialSelect } from "./MaterialSelect";
export function ScenarioPresets({
  media,
  apply,
}: {
  media: Media;
  apply: (settings: Settings) => void;
}) {
  const [selected, setSelected] = useState("compatible");
  return (
    <section className="surface settings-card">
      <div className="section-title">
        <h2>{t("场景起点")}</h2>
        <span className="subtle-label">{t("所有参数均可继续修改")}</span>
      </div>
      <div className="saved-settings-row">
        <MaterialSelect
          label={t("场景预设")}
          value={selected}
          options={scenarioPresets.map((p) => ({ value: p.id, label: p.name }))}
          onChange={setSelected}
        />
        <button
          className="button outlined"
          onClick={() => apply(buildScenario(selected, media))}
        >
          {t("应用场景")}
        </button>
      </div>
      <p className="field-hint">
        {t(scenarioPresets.find((p) => p.id === selected)?.description)}
      </p>
      <p className="field-hint">
        {t(
          "预设不会开始处理。应用后可修改质量、码率、分辨率、音频等参数，并保存为自己的配置。",
        )}
      </p>
    </section>
  );
}
