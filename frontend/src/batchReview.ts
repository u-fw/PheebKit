import type { ExecutionPlan, Media, Project } from "./bridge";
import { errorMessage } from "./format";
import { operationOf, validateSettings } from "./project";

export interface BatchReviewRow {
  projectID: string;
  errors: string[];
  plan: ExecutionPlan | null;
}

export function batchFileErrors(project: Project, media: Media): string[] {
  const errors = validateSettings(
    project.settings,
    project.output.uri,
    media.source.uri,
    media,
  );
  if (
    operationOf(project.settings) === "compress" &&
    !media.streams.some(
      (stream) =>
        stream.codecType === "video" &&
        !stream.attachedPicture &&
        !stream.timedThumbnails,
    )
  )
    errors.unshift("源媒体没有视频轨，请选择视频文件。");
  return errors;
}

export function batchReviewKey(
  projects: readonly Project[],
  files: readonly Media[],
): string {
  return JSON.stringify({
    intent: projects.map((project) => ({
      inputs: project.inputs,
      output: project.output,
      settings: project.settings,
    })),
    sources: files.map((file) => ({
      uri: file.source.uri,
      fingerprint: file.sourceFingerprint,
      size: file.sizeBytes,
      duration: file.durationSeconds,
    })),
  });
}

// A review never admits jobs. Run sequentially, publish only the current
// generation and continue past per-file failures so all issues are reviewable.
export async function reviewBatch(
  projects: readonly Project[],
  files: readonly Media[],
  preview: (project: Project) => Promise<ExecutionPlan>,
  current: () => boolean,
  result: (row: BatchReviewRow) => void,
): Promise<boolean> {
  if (projects.length < 1 || projects.length > 32)
    throw new Error("批量预检需要 1–32 个文件。");
  const snapshot = structuredClone(projects);
  const media = new Map(
    files.map((file) => [file.source.uri, structuredClone(file)]),
  );
  for (const project of snapshot) {
    if (!current()) return false;
    const file = media.get(project.inputs[0]?.uri);
    const row: BatchReviewRow = {
      projectID: project.id,
      errors: file
        ? batchFileErrors(project, file)
        : ["源媒体不在当前批次中，请重新导入。"],
      plan: null,
    };
    if (!row.errors.length) {
      try {
        row.plan = await preview(project);
      } catch (cause) {
        row.errors = [errorMessage(cause)];
      }
    }
    if (!current()) return false;
    result(row);
  }
  return true;
}
