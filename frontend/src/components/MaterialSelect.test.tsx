import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { MaterialSelect } from "./MaterialSelect";
import { findOptionByPrefix, nextEnabled } from "./selectNavigation";

const options = [
  { value: "original", label: "保持原始分辨率" },
  { value: "2160", label: "2160p · 4K", disabled: true },
  { value: "1440", label: "1440p · QHD" },
  { value: "1080", label: "1080p · Full HD" },
];

describe("MaterialSelect accessibility and keyboard navigation", () => {
  it("renders a named collapsed combobox with the actual selected value", () => {
    const markup = renderToStaticMarkup(
      <MaterialSelect
        label="分辨率"
        value="1440"
        options={options}
        onChange={() => undefined}
      />,
    );
    expect(markup).toContain('role="combobox"');
    expect(markup).toContain('aria-expanded="false"');
    expect(markup).toContain('aria-haspopup="listbox"');
    expect(markup).toContain("aria-labelledby=");
    expect(markup).toContain("1440p · QHD");
    expect(markup).not.toContain('role="listbox"');
  });
  it("honors disabled controls and never picks disabled options", () => {
    expect(
      renderToStaticMarkup(
        <MaterialSelect
          label="分辨率"
          value="original"
          options={options}
          onChange={() => undefined}
          disabled
        />,
      ),
    ).toContain('disabled=""');
    expect(nextEnabled(options, 0, 1)).toBe(2);
    expect(nextEnabled(options, 2, -1)).toBe(0);
  });
  it("wraps arrow navigation and resolves Home/End without out-of-range indices", () => {
    expect(nextEnabled(options, 3, 1)).toBe(0);
    expect(nextEnabled(options, 0, -1)).toBe(3);
    expect(nextEnabled(options, -1, 1)).toBe(0);
    expect(nextEnabled([], -1, 1)).toBe(-1);
    expect(
      nextEnabled(
        options.map((option) => ({ ...option, disabled: true })),
        0,
        1,
      ),
    ).toBe(-1);
  });
  it("supports typeahead in Chinese/numeric labels and skips disabled matches", () => {
    expect(findOptionByPrefix(options, "保持", 2)).toBe(0);
    expect(findOptionByPrefix(options, "14", 0)).toBe(2);
    expect(findOptionByPrefix(options, "21", 0)).toBe(-1);
    expect(findOptionByPrefix(options, "missing", 0)).toBe(-1);
  });
});
