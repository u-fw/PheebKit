import { Events } from "@wailsio/runtime";
import * as Service from "../bindings/kompeg/internal/app/service";
import type * as Core from "../bindings/kompeg/internal/core/models";
import type { FFmpegSetupStatus } from "../bindings/kompeg/internal/app/models";
export type { FFmpegSetupStatus };

export type Source = Core.Source;
export type Stream = Core.Stream;
export type Media = Omit<Core.Media, "streams"> & { streams: Stream[] };
export type Settings = Omit<Core.Settings, "rawArguments"> & {
  rawArguments: string[];
  encoder: string;
};
export type Project = Omit<Core.Project, "inputs" | "settings"> & {
  inputs: Source[];
  settings: Settings;
};
export type ExecutionPlan = Omit<Core.ExecutionPlan, "args"> & {
  args: string[];
  encoder: string;
  encoderKind: string;
  encoderLabel: string;
  encoderReason: string;
};
export interface VideoEncoder {
  name: string;
  codec: string;
  label: string;
  kind: string;
  available: boolean;
  reason: string;
  pixelFormats: string[];
}
export interface Capabilities {
  version: string;
  encoders: string[];
  codecs: Core.Codec[];
  videoEncoders: VideoEncoder[];
  gpus: { name: string; vendor: string }[];
}
export interface BinaryStatus {
  path: string;
  version: string;
  source: string;
  available: boolean;
}
export interface EnvironmentStatus {
  platform: string;
  ready: boolean;
  ffmpeg: BinaryStatus;
  ffprobe: BinaryStatus;
  message: string;
  configurable: boolean;
}
export interface Thumbnail {
  dataUrl: string;
  source: string;
  width: number;
  height: number;
}
export interface Job {
  id: string;
  projectID: string;
  state: string;
  input: Source;
  output: Source;
  queuedAt?: string;
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
    queuedAt: job.queuedAt,
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

function capabilityView(caps: Core.Capabilities): Capabilities {
  return {
    version: caps.version,
    encoders: (caps.encoders ?? []).map((encoder) => encoder.name),
    codecs: caps.codecs ?? [],
    videoEncoders: (caps.videoEncoders ?? []).map((encoder) => ({
      ...encoder,
      pixelFormats: encoder.pixelFormats ?? [],
    })),
    gpus: caps.gpus ?? [],
  };
}

export const API = {
  getFFmpegSetupStatus: async (): Promise<FFmpegSetupStatus> =>
    Service.GetFFmpegSetupStatus(),
  startFFmpegSetup: async (source: string): Promise<FFmpegSetupStatus> =>
    Service.StartFFmpegSetup(source),
  startCustomFFmpegSetup: async (
    link: string,
    sha256: string,
    trustBuild: boolean,
  ): Promise<FFmpegSetupStatus> =>
    Service.StartCustomFFmpegSetup(link, sha256, trustBuild),
  cancelFFmpegSetup: async (): Promise<void> => Service.CancelFFmpegSetup(),
  getPlatform: async (): Promise<string> => Service.GetPlatform(),
  getEnvironment: async (): Promise<EnvironmentStatus> =>
    Service.GetEnvironment(),
  configureFFmpeg: async (
    ffmpegPath: string,
    ffprobePath: string,
  ): Promise<EnvironmentStatus> =>
    Service.ConfigureFFmpeg(ffmpegPath, ffprobePath),
  chooseFFmpeg: async (): Promise<EnvironmentStatus | null> =>
    Service.ChooseFFmpeg(),
  getThumbnail: async (uri: string): Promise<Thumbnail> =>
    Service.GetThumbnail(uri),
  chooseMedia: async (): Promise<Media | null> => {
    const media = await Service.ChooseMedia();
    return media ? mediaView(media) : null;
  },
  probe: async (path: string): Promise<Media> =>
    mediaView(await Service.Probe(path)),
  capabilities: async (): Promise<Capabilities> =>
    capabilityView(await Service.Capabilities()),
  refreshCapabilities: async (): Promise<Capabilities> =>
    capabilityView(await Service.RefreshCapabilities()),
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
