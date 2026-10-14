import { describe, it, expect } from "vitest";
import { newEditHistory, reduceEditHistory } from "./editHistory";
import { defaultSettings } from "./project";
describe("project undo and redo", () => {
  it("restores a preset and its output extension together; branches discard redo", () => {
    const original = {
      settings: structuredClone(defaultSettings),
      output: "C:/out.mp4",
    };
    let state = newEditHistory(original);
    state = reduceEditHistory(state, {
      type: "edit",
      at: 1,
      limit: 50,
      value: (current) => ({
        ...current,
        settings: { ...current.settings, codec: "av1" },
        output: "C:/out.webm",
      }),
    });
    state = reduceEditHistory(state, { type: "undo", at: 2, limit: 50 });
    expect(state.present).toEqual(original);
    state = reduceEditHistory(state, { type: "redo", at: 3, limit: 50 });
    expect(state.present.output).toBe("C:/out.webm");
    state = reduceEditHistory(state, { type: "undo", at: 4, limit: 50 });
    state = reduceEditHistory(state, {
      type: "edit",
      at: 5,
      limit: 50,
      value: (current) => ({
        ...current,
        settings: { ...current.settings, quality: 19 },
      }),
    });
    expect(state.future).toHaveLength(0);
    expect(original.settings.quality).toBe(defaultSettings.quality);
  });
  it("coalesces continuous input while preserving separate actions and a bounded history", () => {
    let state = newEditHistory({
      settings: structuredClone(defaultSettings),
      output: "out.mp4",
    });
    for (let i = 0; i < 5; i++)
      state = reduceEditHistory(state, {
        type: "edit",
        at: i * 100,
        group: "quality",
        limit: 10,
        value: (current) => ({
          ...current,
          settings: { ...current.settings, quality: 18 + i },
        }),
      });
    expect(state.past).toHaveLength(1);
    state = reduceEditHistory(state, { type: "undo", at: 600, limit: 10 });
    expect(state.present.settings.quality).toBe(defaultSettings.quality);
    for (let i = 0; i < 30; i++)
      state = reduceEditHistory(state, {
        type: "edit",
        at: 1000 + i * 1000,
        limit: 10,
        value: (current) => ({ ...current, output: `out-${i}.mp4` }),
      });
    expect(state.past).toHaveLength(10);
  });
});
