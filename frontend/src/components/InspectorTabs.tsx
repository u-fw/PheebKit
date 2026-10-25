import { useId, type KeyboardEvent } from "react";
import { t } from "../i18n";

export const inspectorTabs = [
  { id: "output", label: "输出" },
  { id: "tracks", label: "轨道" },
  { id: "adjust", label: "调整" },
  { id: "tools", label: "工具" },
] as const;
export type InspectorTab = (typeof inspectorTabs)[number]["id"];
export function InspectorTabs({
  value,
  onChange,
  prefix,
}: {
  value: InspectorTab;
  onChange: (value: InspectorTab) => void;
  prefix: string;
}) {
  const id = useId();
  function keydown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    let next: number;
    if (event.key === "ArrowRight") next = (index + 1) % inspectorTabs.length;
    else if (event.key === "ArrowLeft")
      next = (index + inspectorTabs.length - 1) % inspectorTabs.length;
    else if (event.key === "Home") next = 0;
    else if (event.key === "End") next = inspectorTabs.length - 1;
    else return;
    event.preventDefault();
    onChange(inspectorTabs[next].id);
    document.getElementById(`${prefix}-tab-${inspectorTabs[next].id}`)?.focus();
  }
  return (
    <div
      className="inspector-tabs"
      role="tablist"
      aria-label={t("处理参数分类")}
      id={id}
    >
      {inspectorTabs.map((tab, index) => (
        <button
          key={tab.id}
          id={`${prefix}-tab-${tab.id}`}
          type="button"
          role="tab"
          aria-selected={value === tab.id}
          tabIndex={value === tab.id ? 0 : -1}
          aria-controls={`${prefix}-panel-${tab.id}`}
          onKeyDown={(event) => keydown(event, index)}
          onClick={() => onChange(tab.id)}
        >
          {t(tab.label)}
        </button>
      ))}
    </div>
  );
}
