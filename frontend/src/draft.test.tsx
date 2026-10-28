// @vitest-environment happy-dom

import { afterEach, beforeEach, describe, it, expect, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { draftFingerprint, inOutputDirectory } from "./draft";
import { makeProject, defaultSettings } from "./project";
import { UnsavedDialog } from "./components/UnsavedDialog";
import { ExitDialog } from "./components/ExitDialog";
import { ProjectLibrary } from "./components/ProjectLibrary";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { setLanguage } from "./i18n";
vi.mock("./bridge", () => ({ API: {} }));
beforeEach(() => setLanguage("zh-CN"));
afterEach(cleanup);
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
    const reordered = Object.fromEntries(
      Object.entries(project.settings).reverse(),
    ) as typeof project.settings;
    expect(draftFingerprint({ ...project, settings: reordered })).toBe(
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
  it("restores an inactive recovery draft through the visible action", () => {
    const restore = vi.fn();
    render(
      <ProjectLibrary
        library={{
          preferences: {
            autosave: true,
            intervalSeconds: 15,
            draftLimit: 5,
            retentionDays: 7,
            recentLimit: 10,
            undoLimit: 50,
          },
          drafts: [
            {
              id: "draft-1",
              source: "C:/media/example.mp4",
              updatedAt: "2026-10-28T08:00:00Z",
              active: false,
            },
          ],
          recent: [],
          error: "",
        }}
        connected
        refresh={async () => {}}
        openRecent={() => {}}
        restore={restore}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "恢复草稿" }));
    expect(restore).toHaveBeenCalledWith("draft-1");
  });
});
