import type { Project } from "./bridge";
export function draftFingerprint(project: Project): string {
  return JSON.stringify(
    [project.inputs, project.output, project.settings],
    (_key, value: unknown) => {
      if (value && typeof value === "object" && !Array.isArray(value)) {
        return Object.fromEntries(
          Object.entries(value).sort(([a], [b]) => a.localeCompare(b)),
        );
      }
      return value;
    },
  );
}
export type Draft = {
  project: Project;
  dirty: boolean;
  markSaved: (project: Project) => void;
};
export function inOutputDirectory(path: string, directory?: string): string {
  return directory
    ? `${directory.replace(/[\\/]$/, "")}/${path.split(/[\\/]/).pop()}`
    : path;
}
