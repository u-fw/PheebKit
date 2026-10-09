import { Events } from "@wailsio/runtime";
import * as Service from "../bindings/kompeg/internal/app/service";
import type * as Core from "../bindings/kompeg/internal/core/models";

export type Source = Core.Source;
export type Stream = Core.Stream;
export type Media = Omit<Core.Media, "streams"> & { streams: Stream[] };
export type Settings = Omit<Core.Settings, "rawArguments"> & {
  rawArguments: string[];
};
export type Project = Omit<Core.Project, "inputs" | "settings"> & {
  inputs: Source[];
  settings: Settings;
};
export type ExecutionPlan = Omit<Core.ExecutionPlan, "args"> & {
  args: string[];
};
export interface Capabilities {
  version: string;
  encoders: string[];
  codecs: Core.Codec[];
}
export interface Job {
  id: string;
  projectID: string;
  state: string;
  input: Source;
  output: Source;
  startedAt: string;
  completedAt: string;
  duration: number;
  progress: number;
  fps: number;
  speed: number;
  currentTime: number;
  totalTime: number;
  inputSize: number;
  outputSize: number;
  error: string;
  logs: string[];
  savedPercent?: number;
  averageSpeed?: number;
}

function jobView(job: Core.Job): Job {
  return {
    id: job.id,
    projectID: job.projectID,
    state: job.state,
    input: job.input,
    output: job.output,
    startedAt: job.startedAt,
    completedAt: job.completedAt,
    duration: job.result?.elapsedSeconds ?? 0,
    progress: job.progress.percent / 100,
    fps: job.progress.fps,
    speed: job.progress.speed,
    currentTime: job.progress.currentTimeSeconds,
    totalTime: job.progress.totalTimeSeconds,
    inputSize: job.result?.inputSizeBytes ?? 0,
    outputSize: job.result?.outputSizeBytes ?? job.progress.outputSizeBytes,
    error: job.error,
    logs: job.logs ?? [],
    savedPercent: job.result?.savedPercent,
    averageSpeed: job.result?.averageSpeed,
  };
}

function mediaView(media: Core.Media): Media {
  return { ...media, streams: media.streams ?? [] };
}

export const API = {
  getPlatform: async (): Promise<string> => Service.GetPlatform(),
  chooseMedia: async (): Promise<Media | null> => {
    const media = await Service.ChooseMedia();
    return media ? mediaView(media) : null;
  },
  probe: async (path: string): Promise<Media> =>
    mediaView(await Service.Probe(path)),
  capabilities: async (): Promise<Capabilities> => {
    const caps = await Service.Capabilities();
    return {
      version: caps.version,
      encoders: (caps.encoders ?? []).map((e) => e.name),
      codecs: caps.codecs ?? [],
    };
  },
  preview: async (project: Project): Promise<ExecutionPlan> => {
    const plan = await Service.Preview(project);
    return { ...plan, args: plan.args ?? [] };
  },
  start: async (project: Project): Promise<Job> =>
    jobView(await Service.Start(project)),
  cancel: async (id: string): Promise<void> => {
    await Service.Cancel(id);
  },
  jobs: async (): Promise<Job[]> => ((await Service.Jobs()) ?? []).map(jobView),
  chooseOutput: async (suggested: string): Promise<string> =>
    Service.ChooseOutput(suggested),
  subscribeJobs: (callback: (job: Job) => void): (() => void) =>
    Events.On("job:updated", (event) => callback(jobView(event.data))),
  subscribeMedia: (callback: (media: Media) => void): (() => void) =>
    Events.On("media:probed", (event) => callback(mediaView(event.data))),
  onDrop: (callback: (paths: string[]) => void): (() => void) =>
    Events.On("media:drop", (event) => callback(event.data)),
};
