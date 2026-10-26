import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";
import {
  filterCommands,
  nextCommandIndex,
  type CommandAction,
} from "../commandPalette";
import { t } from "../i18n";
import { Icon } from "./Icon";

export function CommandPalette({
  commands,
  close,
}: {
  commands: readonly CommandAction[];
  close: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const input = useRef<HTMLInputElement>(null);
  const listID = useId();
  const titleID = useId();
  const [query, setQuery] = useState("");
  const filtered = useMemo(
    () => filterCommands(commands, query, t),
    [commands, query],
  );
  const [activeID, setActiveID] = useState<string | null>(
    () => commands.find((command) => !command.disabled)?.id ?? null,
  );
  const selectedIndex = filtered.findIndex(
    (command) => command.id === activeID,
  );
  const active =
    selectedIndex >= 0 && !filtered[selectedIndex]?.disabled
      ? selectedIndex
      : nextCommandIndex(filtered, -1, 1);

  useEffect(() => {
    const element = dialog.current;
    if (!element) return;
    element.showModal();
    const focusFrame = requestAnimationFrame(() => input.current?.focus());
    const overflow = document.documentElement.style.overflow;
    document.documentElement.style.overflow = "hidden";
    return () => {
      cancelAnimationFrame(focusFrame);
      document.documentElement.style.overflow = overflow;
      if (element.open) element.close();
    };
  }, []);

  function run(command: CommandAction | undefined) {
    if (!command || command.disabled) return;
    close();
    void command.run();
  }

  function onInputKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    function move(next: number) {
      if (next >= 0) setActiveID(filtered[next]?.id ?? null);
    }
    if (event.key === "ArrowDown") {
      event.preventDefault();
      move(nextCommandIndex(filtered, active, 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      move(nextCommandIndex(filtered, active, -1));
    } else if (event.key === "Home") {
      event.preventDefault();
      move(nextCommandIndex(filtered, -1, 1));
    } else if (event.key === "End") {
      event.preventDefault();
      const last = [...filtered]
        .map((command, index) => ({ command, index }))
        .reverse()
        .find((item) => !item.command.disabled);
      if (last) move(last.index);
    } else if (event.key === "Enter") {
      event.preventDefault();
      run(filtered[active]);
    }
  }

  return (
    <dialog
      ref={dialog}
      className="command-palette"
      aria-labelledby={titleID}
      onCancel={(event) => {
        event.preventDefault();
        close();
      }}
      onClick={(event) => {
        if (event.target === dialog.current) close();
      }}
    >
      <div className="command-palette-surface glass">
        <h2 id={titleID} className="visually-hidden">
          {t("快速操作")}
        </h2>
        <div className="command-palette-search">
          <Icon name="search" />
          <input
            ref={input}
            autoFocus
            value={query}
            placeholder={t("搜索操作…")}
            aria-label={t("搜索快速操作")}
            role="combobox"
            aria-expanded="true"
            aria-controls={listID}
            aria-autocomplete="list"
            aria-activedescendant={
              active >= 0 ? `${listID}-${filtered[active]?.id}` : undefined
            }
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={onInputKeyDown}
          />
          <kbd>Ctrl K</kbd>
        </div>
        <div
          id={listID}
          className="command-palette-list"
          role="listbox"
          aria-label={t("可用操作")}
        >
          {filtered.length ? (
            filtered.map((command, index) => (
              <button
                type="button"
                role="option"
                id={`${listID}-${command.id}`}
                key={command.id}
                aria-selected={index === active}
                aria-disabled={command.disabled || undefined}
                className={index === active ? "active" : ""}
                disabled={command.disabled}
                tabIndex={-1}
                onMouseMove={() => {
                  if (!command.disabled) setActiveID(command.id);
                }}
                onClick={() => run(command)}
              >
                <span className="command-icon">
                  <Icon name={command.icon} />
                </span>
                <span className="command-copy">
                  <strong>{t(command.label)}</strong>
                  <small>{t(command.description)}</small>
                </span>
                {command.shortcut && <kbd>{command.shortcut}</kbd>}
              </button>
            ))
          ) : (
            <p className="command-empty" role="status">
              {t("没有匹配的操作")}
            </p>
          )}
        </div>
        <footer className="command-palette-help">
          <span>{t("↑↓ 选择 · Enter 执行 · Esc 关闭")}</span>
          <span>{t("Material You 快速操作")}</span>
        </footer>
      </div>
    </dialog>
  );
}
