import { describe, expect, it } from "vitest";
import { chooseTimePart, timeOptions, timeParts } from "./timePicker";

describe("duration selector", () => {
  it("retains fractional precision and supports long media", () => {
    expect(timeParts(11.779)).toEqual({
      hours: 0,
      minutes: 0,
      seconds: 11,
      milliseconds: 779,
    });
    expect(chooseTimePart(11.779, "seconds", 20, 143.125)).toBe(20.779);
    expect(timeParts(90000)).toEqual({
      hours: 25,
      minutes: 0,
      seconds: 0,
      milliseconds: 0,
    });
  });
  it("bounds selectable units by the real source duration", () => {
    expect(
      timeOptions(143.125, "seconds", 143.125).find(
        (option) => option.value === "24",
      )?.disabled,
    ).toBeUndefined();
    expect(
      timeOptions(0, "minutes", 143.125).find((option) => option.value === "3")
        ?.disabled,
    ).toBeUndefined();
    expect(timeOptions(0, "hours", 143.125)).toHaveLength(1);
    expect(chooseTimePart(59.779, "minutes", 2, 143.125)).toBe(143.125);
  });
});
