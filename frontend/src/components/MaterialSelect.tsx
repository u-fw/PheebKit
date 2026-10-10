import { useEffect, useId, useRef, useState } from "react";
import type { KeyboardEvent } from "react";
import { Icon } from "./Icon";
import { findOptionByPrefix, nextEnabled } from "./selectNavigation";
import type { SelectOption } from "./selectNavigation";

export function MaterialSelect({
  label,
  value,
  options,
  onChange,
  disabled = false,
  describedBy,
  invalid = false,
  accessibleLabel,
}: {
  label: string;
  value: string;
  options: readonly SelectOption[];
  onChange: (value: string) => void;
  disabled?: boolean;
  describedBy?: string;
  invalid?: boolean;
  accessibleLabel?: string;
}) {
  const id = useId();
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const search = useRef({ text: "", time: 0 });
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const [above, setAbove] = useState(false);
  const [menuHeight, setMenuHeight] = useState(320);
  const selected = options.findIndex((option) => option.value === value);

  useEffect(() => {
    if (!open) return;
    function outside(event: PointerEvent) {
      if (event.target && !root.current?.contains(event.target as Node))
        setOpen(false);
    }
    document.addEventListener("pointerdown", outside);
    return () => document.removeEventListener("pointerdown", outside);
  }, [open]);

  useEffect(() => {
    if (open && active >= 0)
      document
        .getElementById(`${id}-option-${active}`)
        ?.scrollIntoView({ block: "nearest" });
  }, [open, active, id]);

  function show() {
    if (disabled) return;
    const rect = trigger.current?.getBoundingClientRect();
    if (rect) {
      const topBar = document.querySelector(".topbar")?.getBoundingClientRect();
      const bottomBar = document
        .querySelector(".bottom-nav")
        ?.getBoundingClientRect();
      const viewportHeight = document.documentElement.clientHeight;
      const bottom = bottomBar?.height ? bottomBar.top : viewportHeight;
      const top = topBar?.bottom ?? 0;
      const belowSpace = Math.max(48, bottom - rect.bottom - 16);
      const aboveSpace = Math.max(48, rect.top - top - 16);
      const preferredHeight = Math.min(options.length * 72 + 16, 320);
      const opensAbove =
        belowSpace < preferredHeight && aboveSpace > belowSpace;
      setAbove(opensAbove);
      setMenuHeight(
        Math.min(preferredHeight, opensAbove ? aboveSpace : belowSpace),
      );
    }
    setActive(
      selected >= 0 && !options[selected].disabled
        ? selected
        : nextEnabled(options, -1, 1),
    );
    search.current = { text: "", time: 0 };
    setOpen(true);
  }

  function choose(index: number) {
    const option = options[index];
    if (!option || option.disabled) return;
    onChange(option.value);
    setOpen(false);
    trigger.current?.focus();
  }

  function handleKey(event: KeyboardEvent<HTMLButtonElement>) {
    if (event.key === "Tab") {
      setOpen(false);
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      setOpen(false);
      return;
    }
    if (
      ["ArrowDown", "ArrowUp", "Home", "End", "Enter", " "].includes(event.key)
    ) {
      event.preventDefault();
      if (!open) {
        show();
        if (event.key === "Home") setActive(nextEnabled(options, -1, 1));
        if (event.key === "End") setActive(nextEnabled(options, 0, -1));
        return;
      }
      if (event.key === "Enter" || event.key === " ") choose(active);
      else if (event.key === "Home") setActive(nextEnabled(options, -1, 1));
      else if (event.key === "End") setActive(nextEnabled(options, 0, -1));
      else
        setActive(
          nextEnabled(options, active, event.key === "ArrowDown" ? 1 : -1),
        );
      return;
    }
    if (
      event.key.length === 1 &&
      !event.ctrlKey &&
      !event.metaKey &&
      !event.altKey
    ) {
      event.preventDefault();
      const now = Date.now();
      const prefix =
        now - search.current.time < 700
          ? search.current.text + event.key
          : event.key;
      if (!open) show();
      search.current = { text: prefix, time: now };
      const match = findOptionByPrefix(options, prefix, active);
      if (match >= 0) setActive(match);
    }
  }

  return (
    <div className={`material-select ${open ? "is-open" : ""}`} ref={root}>
      <button
        ref={trigger}
        type="button"
        className="material-select-trigger"
        role="combobox"
        aria-haspopup="listbox"
        aria-describedby={describedBy}
        aria-invalid={invalid || undefined}
        aria-expanded={open}
        aria-controls={open ? `${id}-listbox` : undefined}
        aria-label={accessibleLabel}
        aria-labelledby={
          accessibleLabel ? undefined : `${id}-label ${id}-value`
        }
        aria-activedescendant={
          open && active >= 0 ? `${id}-option-${active}` : undefined
        }
        disabled={disabled}
        onClick={() => (open ? setOpen(false) : show())}
        onKeyDown={handleKey}
        onBlur={(event) => {
          if (!root.current?.contains(event.relatedTarget)) setOpen(false);
        }}
      >
        <span className="material-select-copy">
          <span className="material-field-label" id={`${id}-label`}>
            {label}
          </span>
          <span id={`${id}-value`}>{options[selected]?.label || "请选择"}</span>
        </span>
        <Icon name="chevron" />
      </button>
      {open && (
        <div
          role="listbox"
          id={`${id}-listbox`}
          aria-label={accessibleLabel || label}
          className={`material-select-menu ${above ? "above" : ""}`}
          style={{ maxHeight: menuHeight }}
        >
          {options.map((option, index) => (
            <button
              type="button"
              role="option"
              key={option.value}
              id={`${id}-option-${index}`}
              aria-selected={value === option.value}
              aria-disabled={option.disabled || undefined}
              tabIndex={-1}
              className={`material-select-option ${active === index ? "highlighted" : ""}`}
              onPointerDown={(event) => event.preventDefault()}
              onPointerMove={() => {
                if (!option.disabled) setActive(index);
              }}
              onClick={() => choose(index)}
            >
              <span>
                <strong>{option.label}</strong>
                {option.description && <small>{option.description}</small>}
              </span>
              {value === option.value && <Icon name="check" />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
