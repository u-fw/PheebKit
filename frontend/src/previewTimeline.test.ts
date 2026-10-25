import { describe, expect, it } from "vitest";
import {
  boundedPosition,
  filmstripPositions,
  framePosition,
  selectionPlayback,
} from "./previewTimeline";

describe("source preview timing", () => {
  it("bounds seeking and steps without accumulating millisecond rounding", () => {
    expect(boundedPosition(NaN, 3)).toBe(0);
    expect(boundedPosition(10, 3)).toBe(3);
    expect(boundedPosition(-1, 3)).toBe(0);
    expect(framePosition(0, -1, 30, 8)).toBe(0);
    expect(framePosition(8, 1, 30, 8)).toBe(8);
    expect(framePosition(1, 1, 30000 / 1001, 8)).toBeCloseTo(1.0333666667);
    expect(framePosition(1, -1, 0, 8)).toBeCloseTo(0.9);
  });
  it("keeps the selected range exclusive and loops even at media end", () => {
    expect(selectionPlayback(2.5, 2, 4, false)).toBeNull();
    expect(selectionPlayback(4, 2, 4, false)).toEqual({ seek: 4, pause: true });
    expect(selectionPlayback(8, 2, 8, true)).toEqual({ seek: 2, pause: false });
    expect(selectionPlayback(0, 2, 4, true)).toEqual({ seek: 2, pause: false });
    expect(selectionPlayback(0, 4, 2, true)).toBeNull();
  });
  it("bounds filmstrip extraction independently of source length", () => {
    expect(filmstripPositions(Infinity)).toEqual([]);
    expect(filmstripPositions(0)).toEqual([]);
    expect(filmstripPositions(0.2)).toEqual([0]);
    expect(filmstripPositions(8)).toEqual([0, 1, 2, 3, 4, 5, 6, 7]);
    expect(filmstripPositions(86400)).toHaveLength(8);
    expect(filmstripPositions(86400).at(-1)).toBeLessThan(86400);
  });
});
