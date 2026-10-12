import { useEffect, useId, useRef, useState } from "react";
import type { KeyboardEvent } from "react";
import { Icon } from "./Icon";
import {
  findOptionByPrefix,
  nextEnabled,
  numericOption,
} from "./selectNavigation";
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
  editable = false,
}: {
  label: string;
  value: string;
  options: readonly SelectOption[];
  onChange: (value: string) => void;
  disabled?: boolean;
  describedBy?: string;
  invalid?: boolean;
  accessibleLabel?: string;
  editable?: boolean;
}) {
  const id = useId();
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const input = useRef<HTMLInputElement>(null);
  const [draft, setDraft] = useState<{ base: string; text: string } | null>(
    null,
  );
  const search = useRef({ text: "", time: 0 });
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const [above, setAbove] = useState(false);
  const [menuHeight, setMenuHeight] = useState(320);
  const selected = options.findIndex((option) => option.value === value);
  const text =
    draft?.base === value ? draft.text : options[selected]?.label || value;
  const typedOption = numericOption(options, text);
  const inputError = editable && draft?.base === value && !typedOption;

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
    setDraft(null);
    setOpen(false);
    if (editable) input.current?.focus();
    else trigger.current?.focus();
  }

  function handleKey(event: KeyboardEvent<HTMLElement>) {
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
      {editable ? (
        <div
          className={`material-select-editable ${invalid || inputError ? "invalid" : ""}`}
        >
          <label className="material-select-copy" htmlFor={`${id}-input`}>
            <span className="material-field-label">{label}</span>
            <input
              ref={input}
              id={`${id}-input`}
              type="text"
              inputMode="numeric"
              autoComplete="off"
              spellCheck={false}
              value={text}
              disabled={disabled}
              aria-label={accessibleLabel || label}
              aria-invalid={invalid || inputError || undefined}
              aria-describedby={
                `${describedBy || ""}${inputError ? ` ${id}-error` : ""}`.trim() ||
                undefined
              }
              onFocus={(event) => {
                setOpen(false);
                event.currentTarget.select();
              }}
              onChange={(event) => {
                const raw = event.target.value;
                const option = numericOption(options, raw);
                setOpen(false);
                setDraft({ base: option?.value ?? value, text: raw });
                if (option) onChange(option.value);
              }}
              onBlur={() => {
                setDraft(null);
                setOpen(false);
              }}
              onKeyDown={(event) => {
                if (open || (event.altKey && event.key === "ArrowDown"))
                  handleKey(event);
                else if (event.key === "Enter" || event.key === "Escape") {
                  event.preventDefault();
                  setDraft(null);
                  setOpen(false);
                }
              }}
            />
          </label>
          <button
            ref={trigger}
            type="button"
            className="time-part-toggle"
            aria-label={`${accessibleLabel || label}候选列表`}
            aria-haspopup="listbox"
            aria-expanded={open}
            aria-controls={open ? `${id}-listbox` : undefined}
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
            <Icon name="chevron" />
          </button>
        </div>
      ) : (
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
            <span id={`${id}-value`}>
              {options[selected]?.label || "请选择"}
            </span>
          </span>
          <Icon name="chevron" />
        </button>
      )}
      {inputError && (
        <small id={`${id}-error`} className="inline-error" role="status">
          请输入 {options[0]?.label}–{options.at(-1)?.label} 的整数
        </small>
      )}
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
