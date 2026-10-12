import { describe, it, expect, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { draftFingerprint, inOutputDirectory } from "./draft";
import { makeProject, defaultSettings } from "./project";
import { UnsavedDialog } from "./components/UnsavedDialog";
import { ExitDialog } from "./components/ExitDialog";
vi.mock("./bridge", () => ({ API: {} }));
describe("editable project protection", () => {
  it("tracks media/output/settings but ignores regenerated task identities", () => {
    const media = {
      source: {
        uri: "C:/in.mp4",
        displayName: "in.mp4",
        kind: "local-file" as const,
      },
    };
    const project = makeProject(
      media as Parameters<typeof makeProject>[0],
      defaultSettings,
      "C:/out.mp4",
      "a",
    );
    expect(draftFingerprint({ ...project, id: "b" })).toBe(
      draftFingerprint(project),
    );
    expect(
      draftFingerprint({
        ...project,
        settings: {
          ...project.settings,
          trim: { startSeconds: 1, endSeconds: 4 },
        },
      }),
    ).not.toBe(draftFingerprint(project));
    expect(
      draftFingerprint({
        ...project,
        output: { ...project.output, uri: "C:/out-2.mp4" },
      }),
    ).not.toBe(draftFingerprint(project));
    expect(inOutputDirectory("C:/in-pheeb.mp4", "D:/results")).toBe(
      "D:/results/in-pheeb.mp4",
    );
  });
  it("provides safe return/save/discard for replacement and idle exit", () => {
    const html = renderToStaticMarkup(
      <UnsavedDialog
        save={async () => false}
        discard={async () => {}}
        dismiss={() => {}}
      />,
    );
    expect(html).toContain("继续编辑");
    expect(html).toContain("保存项目并继续");
    const exit = renderToStaticMarkup(
      <ExitDialog
        request={{ jobs: 0, downloading: false, unsaved: true }}
        save={async () => false}
        dismiss={() => {}}
      />,
    );
    expect(exit).toContain("保存项目并退出");
    expect(exit).toContain("放弃修改并退出");
    expect(exit).not.toContain("退出会取消这些操作");
  });
});
