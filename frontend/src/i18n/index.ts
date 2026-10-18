import { useSyncExternalStore } from "react";
import english from "../../../locales/en.json";

export type Language = "zh-CN" | "en-US";
const storageKey = "pheebkit-language";
const listeners = new Set<() => void>();
const catalog = new Map(Object.entries(english));
let persistent = true;

function initialLanguage(): Language {
  if (typeof window === "undefined") return "zh-CN";
  try {
    const saved = localStorage.getItem(storageKey);
    persistent = true;
    if (saved === "zh-CN" || saved === "en-US") return saved;
  } catch {
    persistent = false;
  }
  return navigator.language.toLowerCase().startsWith("zh") ? "zh-CN" : "en-US";
}

let language = initialLanguage();
let snapshot = { language, persistent };
export function getLanguage(): Language {
  return language;
}

function notify() {
  if (snapshot.language === language && snapshot.persistent === persistent)
    return;
  snapshot = { language, persistent };
  for (const listener of listeners) listener();
}
export function saveLocalPreference(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
    persistent = true;
  } catch {
    persistent = false;
  }
  notify();
}
export function setLanguage(next: Language): void {
  if (next !== "zh-CN" && next !== "en-US")
    throw new Error("Unsupported language");
  language = next;
  saveLocalPreference(storageKey, next);
  if (typeof document !== "undefined") document.documentElement.lang = next;
  notify();
}

function onStorage(event: StorageEvent) {
  if (event.key !== storageKey && event.key !== null) return;
  language = initialLanguage();
  if (typeof document !== "undefined") document.documentElement.lang = language;
  notify();
}
function subscribe(listener: () => void) {
  if (!listeners.size && typeof window !== "undefined")
    window.addEventListener("storage", onStorage);
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
    if (!listeners.size && typeof window !== "undefined")
      window.removeEventListener("storage", onStorage);
  };
}

function interpolate(message: string, values: readonly unknown[]): string {
  return message.replace(/\{\{(\d+)\}\}/g, (token, index: string) =>
    Number(index) < values.length ? String(values[Number(index)] ?? "") : token,
  );
}

// Patterns apply only to application-owned messages passed at render boundaries.
// Captured values stay literal: filenames, paths and user text are never keys.
const patterns = [...catalog]
  .flatMap(([source, target]) => {
    if (!source.includes("{{") || !/[\u3400-\u9fff]/u.test(source)) return [];
    const literals = source.split(/\{\{\d+\}\}/g);
    const markerOrder = [...source.matchAll(/\{\{(\d+)\}\}/g)].map((match) =>
      Number(match[1]),
    );
    const anchor = [...literals].sort((a, b) => b.length - a.length)[0];
    if (!anchor) return [];
    const expression = literals
      .map((literal) => literal.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
      .join("([\\s\\S]*?)");
    return [
      {
        source,
        target,
        anchor,
        markerOrder,
        expression: new RegExp(`^${expression}$`, "u"),
      },
    ];
  })
  .sort((a, b) => b.source.length - a.source.length);

const messagePrefixes = [...catalog]
  .filter(
    ([source]) => /[：:]$/.test(source) && /[\u3400-\u9fff]/u.test(source),
  )
  .sort(([a], [b]) => b.length - a.length);

function translateMessage(source: string): string {
  const exact = catalog.get(source);
  if (exact !== undefined) return exact;
  if (source.length > 8192 || !/[\u3400-\u9fff]/u.test(source)) return source;
  for (const pattern of patterns) {
    if (!source.includes(pattern.anchor)) continue;
    const match = pattern.expression.exec(source);
    if (!match) continue;
    const values: string[] = [];
    pattern.markerOrder.forEach((index, position) => {
      values[index] = match[position + 1];
    });
    return interpolate(pattern.target, values);
  }
  for (const [prefix, translated] of messagePrefixes) {
    if (source.startsWith(prefix))
      return translated + source.slice(prefix.length);
  }
  if (source.includes("\n"))
    return source
      .split("\n")
      .map((line) => translateMessage(line))
      .join("\n");
  return source;
}

export function t(
  message: string | undefined | null,
  values: readonly unknown[] = [],
): string {
  const source = message ?? "";
  if (language === "zh-CN") return interpolate(source, values);
  return values.length
    ? interpolate(catalog.get(source) ?? source, values)
    : translateMessage(source);
}

export function useI18n() {
  const current = useSyncExternalStore(
    subscribe,
    () => snapshot,
    () => snapshot,
  );
  return { ...current, setLanguage, t };
}
