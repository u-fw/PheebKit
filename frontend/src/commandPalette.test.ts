import { describe, expect, it } from "vitest";
import {
  filterCommands,
  nextCommandIndex,
  type CommandAction,
} from "./commandPalette";

const command = (
  id: string,
  label: string,
  disabled = false,
): CommandAction => ({
  id,
  label,
  description: "",
  icon: "home",
  disabled,
  run: () => undefined,
});

describe("command palette navigation", () => {
  it("matches every search term across labels, descriptions and keywords", () => {
    const commands = [
      { ...command("media", "打开媒体"), keywords: ["import", "video"] },
      { ...command("settings", "外观设置"), keywords: ["theme", "color"] },
    ];
    expect(
      filterCommands(commands, "import video").map(({ id }) => id),
    ).toEqual(["media"]);
    expect(filterCommands(commands, "外观").map(({ id }) => id)).toEqual([
      "settings",
    ]);
    expect(filterCommands(commands, "missing")).toEqual([]);
  });

  it("matches text after localization, including English display labels", () => {
    const commands = [
      {
        ...command("settings", "打开设置"),
        description: "配置外观、FFmpeg 和 GPU。",
      },
    ];
    const translate = (value: string) =>
      ({
        打开设置: "Open settings",
        "配置外观、FFmpeg 和 GPU。": "Configure appearance, FFmpeg and GPU.",
      })[value] ?? value;
    expect(
      filterCommands(commands, "settings", translate).map(({ id }) => id),
    ).toEqual(["settings"]);
    expect(
      filterCommands(commands, "GPU", translate).map(({ id }) => id),
    ).toEqual(["settings"]);
  });

  it("keeps stable English action keywords searchable in localized UI", () => {
    const commands = [
      {
        ...command("settings", "打开设置"),
        keywords: ["settings", "preferences"],
      },
    ];
    const translate = (value: string) =>
      value === "打开设置" ? "Open settings" : value;
    expect(
      filterCommands(commands, "preferences", translate).map(({ id }) => id),
    ).toEqual(["settings"]);
  });

  it("wraps in both directions and skips disabled commands", () => {
    const commands = [
      command("one", "One"),
      command("two", "Two", true),
      command("three", "Three"),
    ];
    expect(nextCommandIndex(commands, -1, 1)).toBe(0);
    expect(nextCommandIndex(commands, 0, 1)).toBe(2);
    expect(nextCommandIndex(commands, 0, -1)).toBe(2);
    expect(nextCommandIndex([command("off", "Off", true)], -1, 1)).toBe(-1);
    expect(nextCommandIndex([], 0, 1)).toBe(-1);
  });
});
