import { describe, expect, it } from "vitest";
import { chooseTimePart, timeOptions, timeParts } from "./timePicker";
import { numericOption } from "./components/selectNavigation";
import { renderToStaticMarkup } from "react-dom/server";
import { createElement } from "react";
import { TimePicker } from "./components/TimePicker";

describe("duration selector", () => {
  it("accepts direct digits without a menu confirmation and preserves milliseconds", () => {
    const options = timeOptions(24.378, "minutes", 120.378);
    expect(numericOption(options, "1")?.value).toBe("1");
    expect(numericOption(options, "01")?.value).toBe("1");
    expect(chooseTimePart(24.378, "minutes", 1, 120.378)).toBe(84.378);
    for (const text of ["", "3", "1.5", "-1", "1e0", "abc"])
      expect(numericOption(options, text)).toBeUndefined();
    expect(
      numericOption(timeOptions(120.378, "seconds", 120.378), "1"),
    ).toBeUndefined();
  });
  it("renders separate editable numbers and list buttons", () => {
    const html = renderToStaticMarkup(
      createElement(TimePicker, {
        label: "开始时间",
        value: 24.378,
        max: 120.378,
        onChange: () => undefined,
        invalid: false,
        describedBy: "trim-help",
      }),
    );
    expect(html.match(/inputMode="numeric"/g)).toHaveLength(3);
    expect(html).toContain('aria-label="开始时间 · 分"');
    expect(html).toContain('aria-label="开始时间 · 分候选列表"');
    expect(html).not.toContain('role="combobox"');
  });
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
