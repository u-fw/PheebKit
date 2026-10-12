import type { Project } from "./bridge";
export function draftFingerprint(project: Project): string {
  return JSON.stringify([project.inputs, project.output, project.settings]);
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
