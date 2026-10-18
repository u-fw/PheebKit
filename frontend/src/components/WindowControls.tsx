import { t } from "../i18n";
import {
  WindowIsMaximised,
  WindowMinimise,
  WindowToggleMaximise,
  Quit,
} from "../../wailsjs/runtime/runtime";
import { useEffect, useState } from "react";
import { Icon } from "./Icon";

export function WindowControls({
  onError,
}: {
  onError: (message: string) => void;
}) {
  const [maximised, setMaximised] = useState(false);

  useEffect(() => {
    let alive = true;
    const refresh = () => {
      void WindowIsMaximised().then(
        (value) => {
          if (alive) setMaximised(value);
        },
        (cause: unknown) => {
          if (alive) onError(`无法读取窗口状态：${String(cause)}`);
        },
      );
    };
    refresh();
    window.addEventListener("resize", refresh);
    return () => {
      alive = false;
      window.removeEventListener("resize", refresh);
    };
  }, [onError]);

  async function control(action: "minimise" | "maximise" | "close") {
    try {
      if (action === "minimise") WindowMinimise();
      else if (action === "close") Quit();
      else {
        WindowToggleMaximise();
        setMaximised(await WindowIsMaximised());
      }
    } catch (cause) {
      onError(`窗口操作失败：${String(cause)}`);
    }
  }

  return (
    <div className="window-controls" role="group" aria-label={t("窗口控制")}>
      <button
        className="window-button"
        aria-label={t("最小化窗口")}
        title={t("最小化")}
        onClick={() => void control("minimise")}
      >
        <Icon name="minimise" />
      </button>
      <button
        className="window-button"
        aria-label={t(maximised ? "还原窗口" : "最大化窗口")}
        title={t(maximised ? "还原" : "最大化")}
        onClick={() => void control("maximise")}
      >
        <Icon name={maximised ? "restore" : "maximise"} />
      </button>
      <button
        className="window-button window-close"
        aria-label={t("关闭窗口")}
        title={t("关闭")}
        onClick={() => void control("close")}
      >
        <Icon name="close" />
      </button>
    </div>
  );
}
