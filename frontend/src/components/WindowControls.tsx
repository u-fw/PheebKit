import { Events, Window } from "@wailsio/runtime";
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
      void Window.IsMaximised().then(
        (value) => {
          if (alive) setMaximised(value);
        },
        (cause: unknown) => {
          if (alive) onError(`无法读取窗口状态：${String(cause)}`);
        },
      );
    };
    refresh();
    const unsubscribe = Events.On("window:maximised", (event) => {
      if (alive && typeof event.data === "boolean") setMaximised(event.data);
    });
    return () => {
      alive = false;
      unsubscribe();
    };
  }, [onError]);

  async function control(action: "minimise" | "maximise" | "close") {
    try {
      if (action === "minimise") await Window.Minimise();
      else if (action === "close") await Window.Close();
      else {
        await Window.ToggleMaximise();
        setMaximised(await Window.IsMaximised());
      }
    } catch (cause) {
      onError(`窗口操作失败：${String(cause)}`);
    }
  }

  return (
    <div className="window-controls" role="group" aria-label="窗口控制">
      <button
        className="window-button"
        aria-label="最小化窗口"
        title="最小化"
        onClick={() => void control("minimise")}
      >
        <Icon name="minimise" />
      </button>
      <button
        className="window-button"
        aria-label={maximised ? "还原窗口" : "最大化窗口"}
        title={maximised ? "还原" : "最大化"}
        onClick={() => void control("maximise")}
      >
        <Icon name={maximised ? "restore" : "maximise"} />
      </button>
      <button
        className="window-button window-close"
        aria-label="关闭窗口"
        title="关闭"
        onClick={() => void control("close")}
      >
        <Icon name="close" />
      </button>
    </div>
  );
}
