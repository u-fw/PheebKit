import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import {
  CustomDownloadFields,
  customDownloadError,
} from "./CustomDownloadFields";

describe("custom ZIP download setup", () => {
  it("accepts a known mirror without asking for a hash", () => {
    expect(
      customDownloadError(
        "https://example.com/file.zip?signature=example",
        "",
        false,
        false,
      ),
    ).toBe("");
  });
  it("requires HTTPS, ZIP, a complete custom hash and explicit trust", () => {
    expect(
      customDownloadError(
        "https://example.com/file.zip",
        "a".repeat(64),
        true,
        true,
      ),
    ).toBe("");
    for (const link of [
      "example.com/file.zip",
      "http://example.com/file.zip",
      "https://user:password@example.com/file.zip",
      "https://example.com/file.7z",
    ])
      expect(customDownloadError(link, "", false, false)).not.toBe("");
    expect(
      customDownloadError("https://example.com/file.zip", "", true, true),
    ).toContain("SHA-256");
    expect(
      customDownloadError(
        "https://example.com/file.zip",
        "a".repeat(64),
        true,
        false,
      ),
    ).toContain("信任");
  });
  it("explains automatic package detection and labels sensitive fields", () => {
    const action = () => undefined;
    const html = renderToStaticMarkup(
      <CustomDownloadFields
        link=""
        sha256=""
        otherBuild={true}
        trusted={false}
        disabled={false}
        onLink={action}
        onSHA256={action}
        onOtherBuild={action}
        onTrusted={action}
      />,
    );
    expect(html).toContain("ZIP 下载链接");
    expect(html).toContain("SHA-256");
    expect(html).toContain("bin 子目录");
    expect(html).toContain("我信任此来源");
    expect(html).toContain('type="checkbox"');
    expect(html).toContain('aria-describedby="custom-hash-help"');
  });
});
