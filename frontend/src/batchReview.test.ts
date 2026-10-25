import { describe, expect, it, vi } from "vitest";
import type { ExecutionPlan, Media, Project } from "./bridge";
import {
  batchReviewKey,
  reviewBatch,
  type BatchReviewRow,
} from "./batchReview";
import { makeBatchProjects } from "./batch";
import { defaultSettings } from "./project";

function media(index: number): Media {
  return {
    source: {
      kind: "local-file",
      uri: `C:/test/${index}.mp4`,
      displayName: `${index}.mp4`,
    },
    name: `${index}.mp4`,
    container: "mp4",
    sizeBytes: 1000,
    durationSeconds: 10,
    bitrate: 800,
    sourceFingerprint: `source-${index}`,
    streams: [
      {
        index: 0,
        codecType: "video",
        codecName: "h264",
        width: 160,
        height: 90,
        frameRate: 30,
        pixelFormat: "yuv420p",
        bitDepth: 8,
        colorSpace: "",
        colorPrimaries: "",
        colorTransfer: "",
        hdr: "",
        title: "",
        language: "",
        attachedPicture: false,
        timedThumbnails: false,
      },
    ],
  };
}
function plan(project: Project): ExecutionPlan {
  return {
    projectID: project.id,
    jobID: "review",
    input: project.inputs[0],
    output: project.output,
    args: [],
    preview: "test plan",
    durationSeconds: 10,
    inputSizeBytes: 1000,
    encoder: "libx265",
    encoderKind: "cpu",
    encoderLabel: "CPU",
    encoderReason: "",
  };
}
describe("batch preflight admission boundary", () => {
  it("invalidates review for intent or source changes, independent of generated IDs", () => {
    const files = [media(1), media(2)];
    const projects = makeBatchProjects(
      files,
      defaultSettings,
      "C:/out",
      "result",
    );
    const key = batchReviewKey(projects, files);
    const copy = structuredClone(projects);
    copy[0].id = "other-id";
    expect(batchReviewKey(copy, files)).toBe(key);
    copy[0].settings.quality++;
    expect(batchReviewKey(copy, files)).not.toBe(key);
    const changed = structuredClone(files);
    changed[1].sourceFingerprint = "replaced";
    expect(batchReviewKey(projects, changed)).not.toBe(key);
  });
  it("reviews all files without queue admission and attributes frontend/backend errors per file", async () => {
    const files = [media(1), media(2), { ...media(3), streams: [] }];
    const projects = makeBatchProjects(
      files,
      defaultSettings,
      "C:/out",
      "result",
    );
    const rows: BatchReviewRow[] = [];
    const preview = vi.fn(async (p: Project) => {
      if (p.id === projects[1].id) throw new Error("Output already exists");
      return plan(p);
    });
    expect(
      await reviewBatch(
        projects,
        files,
        preview,
        () => true,
        (row) => rows.push(row),
      ),
    ).toBe(true);
    expect(preview).toHaveBeenCalledTimes(2);
    expect(rows.map((row) => row.projectID)).toEqual(projects.map((p) => p.id));
    expect(rows[0].plan?.encoderKind).toBe("cpu");
    expect(rows[1].errors).toEqual(["Output already exists"]);
    expect(rows[2].errors.length).toBeGreaterThan(0);
    expect(rows[2].plan).toBeNull();
  });
  it("cancels a stale generation before publishing or checking the next file", async () => {
    const files = [media(1), media(2)];
    const projects = makeBatchProjects(
      files,
      defaultSettings,
      "C:/out",
      "result",
    );
    let release: (value: ExecutionPlan) => void = () => {
      throw new Error("Preview was not started");
    };
    let active = true;
    const preview = vi.fn(
      () =>
        new Promise<ExecutionPlan>((resolve) => {
          release = resolve;
        }),
    );
    const result = vi.fn();
    const pending = reviewBatch(projects, files, preview, () => active, result);
    expect(preview).toHaveBeenCalledOnce();
    active = false;
    release(plan(projects[0]));
    expect(await pending).toBe(false);
    expect(result).not.toHaveBeenCalled();
    expect(preview).toHaveBeenCalledOnce();
  });
  it("freezes the reviewed intent and bounds the number of files", async () => {
    const files = [media(1), media(2)];
    const projects = makeBatchProjects(
      files,
      defaultSettings,
      "C:/out",
      "result",
    );
    const observed: number[] = [];
    const preview = vi.fn(async (p: Project) => {
      observed.push(p.settings.quality);
      if (observed.length === 1) projects[1].settings.quality = 99;
      return plan(p);
    });
    await reviewBatch(
      projects,
      files,
      preview,
      () => true,
      () => undefined,
    );
    expect(observed).toEqual([
      defaultSettings.quality,
      defaultSettings.quality,
    ]);
    await expect(
      reviewBatch(
        Array.from({ length: 33 }, () => projects[0]),
        files,
        preview,
        () => true,
        () => undefined,
      ),
    ).rejects.toThrow("1–32");
    expect(() =>
      makeBatchProjects(
        Array.from({ length: 33 }, () => files[0]),
        defaultSettings,
        "C:/out",
        "result",
      ),
    ).toThrow("32");
  });
});
