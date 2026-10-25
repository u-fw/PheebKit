import { useSyncExternalStore } from "react";
import {
  argbFromHex,
  hexFromArgb,
  themeFromSourceColor,
} from "@material/material-color-utilities";
import { legacyThemeKey } from "./legacy";

export type Appearance = {
  schemaVersion: 1;
  mode: "system" | "light" | "dark";
  seed: string;
  density: "comfortable" | "compact";
};
export const defaultAppearance: Appearance = {
  schemaVersion: 1,
  mode: "system",
  seed: "#236956",
  density: "comfortable",
};
export const appearanceKey = "pheebkit-appearance";
export const seedPattern = /^#[0-9a-f]{6}$/i;

export function parseAppearance(value: string): Appearance | null {
  try {
    const item: unknown = JSON.parse(value);
    if (!item || typeof item !== "object") return null;
    const input = item as Record<string, unknown>;
    if (
      input.schemaVersion !== 1 ||
      !["system", "light", "dark"].includes(String(input.mode)) ||
      !["comfortable", "compact"].includes(String(input.density)) ||
      typeof input.seed !== "string" ||
      !seedPattern.test(input.seed)
    )
      return null;
    return {
      schemaVersion: 1,
      mode: input.mode as Appearance["mode"],
      seed: input.seed.toLowerCase(),
      density: input.density as Appearance["density"],
    };
  } catch {
    return null;
  }
}

export function readAppearance(storage: Pick<Storage, "getItem">): {
  preferences: Appearance;
  persistent: boolean;
  recovered: boolean;
} {
  try {
    const raw = storage.getItem(appearanceKey);
    if (raw !== null) {
      const preferences = parseAppearance(raw);
      return {
        preferences: preferences ?? { ...defaultAppearance },
        persistent: true,
        recovered: !preferences,
      };
    }
    const previous =
      storage.getItem("pheebkit-theme") ?? storage.getItem(legacyThemeKey);
    return {
      preferences: {
        ...defaultAppearance,
        mode: previous === "dark" || previous === "light" ? previous : "system",
      },
      persistent: true,
      recovered: false,
    };
  } catch {
    return {
      preferences: { ...defaultAppearance },
      persistent: false,
      recovered: false,
    };
  }
}

function systemDark() {
  return (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
  );
}
const initial =
  typeof window === "undefined"
    ? {
        preferences: { ...defaultAppearance },
        persistent: true,
        recovered: false,
      }
    : readAppearance({ getItem: (key) => window.localStorage.getItem(key) });
let snapshot = {
  ...initial,
  theme: (initial.preferences.mode === "system"
    ? systemDark()
      ? "dark"
      : "light"
    : initial.preferences.mode) as "light" | "dark",
};
const listeners = new Set<() => void>();
let schemeQuery: MediaQueryList | null = null;
export function getAppearance() {
  return snapshot;
}
function emit() {
  for (const listener of listeners) listener();
}
function refreshSystem() {
  const theme =
    snapshot.preferences.mode === "system"
      ? systemDark()
        ? "dark"
        : "light"
      : snapshot.preferences.mode;
  if (theme !== snapshot.theme) {
    snapshot = { ...snapshot, theme };
    emit();
  }
}
function storageChanged(event: StorageEvent) {
  if (event.key !== appearanceKey && event.key !== null) return;
  const next = readAppearance({
    getItem: (key) => window.localStorage.getItem(key),
  });
  snapshot = {
    ...next,
    theme:
      next.preferences.mode === "system"
        ? systemDark()
          ? "dark"
          : "light"
        : next.preferences.mode,
  };
  emit();
}
export function setAppearance(
  patch: Partial<Omit<Appearance, "schemaVersion">>,
) {
  const preferences = parseAppearance(
    JSON.stringify({ ...snapshot.preferences, ...patch }),
  );
  if (!preferences) throw new Error("Invalid appearance preference");
  let persistent = true;
  try {
    localStorage.setItem(appearanceKey, JSON.stringify(preferences));
  } catch {
    persistent = false;
  }
  snapshot = {
    preferences,
    persistent,
    recovered: false,
    theme:
      preferences.mode === "system"
        ? systemDark()
          ? "dark"
          : "light"
        : preferences.mode,
  };
  emit();
}
function subscribe(listener: () => void) {
  listeners.add(listener);
  if (listeners.size === 1 && typeof window !== "undefined") {
    schemeQuery =
      typeof window.matchMedia === "function"
        ? window.matchMedia("(prefers-color-scheme: dark)")
        : null;
    schemeQuery?.addEventListener("change", refreshSystem);
    window.addEventListener("storage", storageChanged);
    refreshSystem();
  }
  return () => {
    listeners.delete(listener);
    if (!listeners.size && typeof window !== "undefined") {
      schemeQuery?.removeEventListener("change", refreshSystem);
      schemeQuery = null;
      window.removeEventListener("storage", storageChanged);
    }
  };
}
export function useAppearance() {
  return {
    ...useSyncExternalStore(subscribe, getAppearance, getAppearance),
    setAppearance,
  };
}

// HCT palettes from the official Material Color Utilities; no arbitrary HSL
// substitutions. Text/surface pairs use deliberate Material tonal roles.
export function appearanceTokens(
  seed: string,
  dark: boolean,
): Record<string, string> {
  if (!seedPattern.test(seed)) throw new Error("Invalid theme seed");
  const theme = themeFromSourceColor(argbFromHex(seed));
  const colors = (dark ? theme.schemes.dark : theme.schemes.light).toJSON();
  const neutral = (tone: number) =>
    hexFromArgb(theme.palettes.neutral.tone(tone));
  const variant = (tone: number) =>
    hexFromArgb(theme.palettes.neutralVariant.tone(tone));
  const role = (name: keyof typeof colors) => hexFromArgb(colors[name]);
  return {
    "--primary": role("primary"),
    "--on-primary": role("onPrimary"),
    "--primary-container": role("primaryContainer"),
    "--on-primary-container": role("onPrimaryContainer"),
    "--secondary": role("secondary"),
    "--secondary-container": role("secondaryContainer"),
    "--on-secondary-container": role("onSecondaryContainer"),
    "--tertiary": role("tertiary"),
    "--color": role("onSurface"),
    "--muted": variant(dark ? 80 : 30),
    "--subtle": variant(dark ? 80 : 30),
    "--background": neutral(dark ? 6 : 98),
    "--surface": neutral(dark ? 12 : 99),
    "--surface-container-low": neutral(dark ? 10 : 96),
    "--surface-container": neutral(dark ? 12 : 94),
    "--surface-container-high": neutral(dark ? 17 : 92),
    "--surface-container-highest": neutral(dark ? 22 : 90),
    "--surface-tonal": neutral(dark ? 10 : 96),
    "--surface-accent": role("secondaryContainer"),
    "--outline": role("outline"),
    "--outline-variant": role("outlineVariant"),
    "--error": role("error"),
    "--error-container": role("errorContainer"),
    "--on-error-container": role("onErrorContainer"),
    "--success": dark ? "#91d5ac" : "#17613b",
    "--success-container": dark ? "#143d27" : "#d6f4de",
    "--code-background": neutral(dark ? 6 : 96),
    "--accent": role("tertiaryContainer"),
    "--canvas-mint": role("primaryContainer"),
    "--canvas-lilac": role("tertiaryContainer"),
    "--glass": neutral(dark ? 12 : 98) + "ed",
    "--glass-chrome": neutral(dark ? 12 : 98) + "e6",
    "--glass-navigation": neutral(dark ? 10 : 96) + "eb",
    "--glass-edge": role("outlineVariant"),
  };
}
export function applyAppearance(
  target: HTMLElement,
  value: Pick<ReturnType<typeof getAppearance>, "preferences" | "theme">,
) {
  target.dataset.theme = value.theme;
  target.dataset.density = value.preferences.density;
  for (const [name, color] of Object.entries(
    appearanceTokens(value.preferences.seed, value.theme === "dark"),
  ))
    target.style.setProperty(name, color);
}
