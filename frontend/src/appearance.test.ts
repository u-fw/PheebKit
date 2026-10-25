import { afterEach, describe, expect, it, vi } from "vitest";
import { argbFromHex, Contrast } from "@material/material-color-utilities";
import {
  appearanceKey,
  appearanceTokens,
  defaultAppearance,
  getAppearance,
  parseAppearance,
  readAppearance,
  setAppearance,
} from "./appearance";
import { legacyThemeKey } from "./legacy";

afterEach(() => {
  vi.stubGlobal("localStorage", { setItem: vi.fn() });
  setAppearance(defaultAppearance);
  vi.unstubAllGlobals();
});
describe("Material appearance contracts", () => {
  it("validates persisted values and retains legacy theme without writing", () => {
    const getItem = vi.fn((key: string) =>
      key === legacyThemeKey ? "dark" : null,
    );
    expect(readAppearance({ getItem }).preferences.mode).toBe("dark");
    expect(getItem).toHaveBeenCalledWith(appearanceKey);
    expect(
      parseAppearance(JSON.stringify({ ...defaultAppearance, seed: "red" })),
    ).toBeNull();
    expect(
      parseAppearance(
        JSON.stringify({ ...defaultAppearance, schemaVersion: 2 }),
      ),
    ).toBeNull();
    expect(
      parseAppearance(
        JSON.stringify({ ...defaultAppearance, mode: "constructor" }),
      ),
    ).toBeNull();
    expect(readAppearance({ getItem: () => "broken" }).recovered).toBe(true);
    expect(
      readAppearance({
        getItem: () => {
          throw new Error("denied");
        },
      }).persistent,
    ).toBe(false);
  });
  it("keeps in-session preferences when storage fails and rejects invalid mutation", () => {
    vi.stubGlobal("localStorage", {
      setItem: () => {
        throw new Error("denied");
      },
    });
    setAppearance({ seed: "#315DA8", mode: "dark", density: "compact" });
    expect(getAppearance()).toMatchObject({
      persistent: false,
      theme: "dark",
      preferences: { seed: "#315da8", density: "compact" },
    });
    const before = getAppearance();
    expect(() =>
      setAppearance({ seed: "url(https://example.invalid)" }),
    ).toThrow();
    expect(getAppearance()).toBe(before);
  });
  it("generates contrast-safe text roles for arbitrary valid theme seeds in both modes", () => {
    for (const seed of [
      "#236956",
      "#6750a4",
      "#315da8",
      "#8b4a68",
      "#805600",
      "#000000",
      "#ffffff",
      "#ff0000",
      "#00ff00",
      "#0000ff",
    ]) {
      for (const dark of [false, true]) {
        const tokens = appearanceTokens(seed, dark);
        for (const [foreground, background] of [
          ["--primary", "--on-primary"],
          ["--on-primary-container", "--primary-container"],
          ["--on-secondary-container", "--secondary-container"],
          ["--color", "--surface"],
          ["--muted", "--surface"],
          ["--subtle", "--surface-container-high"],
          ["--error", "--surface"],
        ]) {
          const luminance = (color: string) => {
            const value = argbFromHex(color);
            const channels = [
              (value >> 16) & 255,
              (value >> 8) & 255,
              value & 255,
            ].map((n) => {
              const s = n / 255;
              return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
            });
            return (
              (channels[0] * 0.2126 +
                channels[1] * 0.7152 +
                channels[2] * 0.0722) *
              100
            );
          };
          expect(
            Contrast.ratioOfYs(
              luminance(tokens[foreground]),
              luminance(tokens[background]),
            ),
            `${seed} ${dark} ${foreground}`,
          ).toBeGreaterThanOrEqual(4.5);
        }
      }
    }
  });
});
