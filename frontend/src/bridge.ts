import { Events } from "@wailsio/runtime";
import * as Service from "../bindings/kompeg/internal/app/service";
import type * as Core from "../bindings/kompeg/internal/core/models";
import type { FFmpegSetupStatus } from "../bindings/kompeg/internal/app/models";
export type { FFmpegSetupStatus };

export type Source = Core.Source;
export type Stream = Core.Stream;
export type Media = Omit<Core.Media, "streams"> & { streams: Stream[] };
export interface TrackSelection {
  mode: string;
  indices?: number[];
}
export interface VideoTransform {
  rotation: number;
  flipHorizontal: boolean;
  flipVertical: boolean;
  crop?: { top: number; bottom: number; left: number; right: number } | null;
}
export type Settings = Omit<
  Core.Settings,
  "rawArguments" | "audioSelection" | "subtitleSelection" | "transform"
> & {
  rawArguments: string[];
  encoder: string;
  operation?: string;
  audioSelection?: TrackSelection | null;
  subtitleSelection?: TrackSelection | null;
  transform?: VideoTransform | null;
  volumeDB?: number;
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
  project?: Project | null;
  trim?: Core.TimeRange | null;
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

export interface QueueStatus {
  paused: boolean;
  pendingIDs: string[];
  revision: number;
}
export interface SampleComparison {
  original: Media;
  output: Media;
  startSeconds: number;
  durationSeconds: number;
}
export interface LoadedProject {
  media: Media;
  project: Project;
}
function projectView(project: Core.Project): Project {
  return {
    ...project,
    inputs: project.inputs ?? [],
    settings: settingsView(project.settings),
  };
}

function jobView(job: Core.Job): Job {
  return {
    project: job.project
      ? {
          ...job.project,
          inputs: job.project.inputs ?? [],
          settings: settingsView(job.project.settings),
        }
      : null,
    trim: job.trim,
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

function settingsView(settings: Core.Settings): Settings {
  return {
    ...settings,
    rawArguments: settings.rawArguments ?? [],
    encoder: settings.encoder || "auto",
    audioSelection: settings.audioSelection
      ? {
          ...settings.audioSelection,
          indices: settings.audioSelection.indices ?? [],
        }
      : null,
    subtitleSelection: settings.subtitleSelection
      ? {
          ...settings.subtitleSelection,
          indices: settings.subtitleSelection.indices ?? [],
        }
      : null,
  };
}
export interface WorkspacePreferences {
  lastOutputDirectory?: string;
  version: number;
  lastSettings?: Settings | null;
  presets: { id: string; name: string; settings: Settings }[];
}
function preferencesView(
  value: Awaited<ReturnType<typeof Service.SaveSettings>>,
): WorkspacePreferences {
  return {
    ...value,
    lastSettings: value.lastSettings ? settingsView(value.lastSettings) : null,
    presets: (value.presets ?? []).map((p) => ({
      ...p,
      settings: settingsView(p.settings),
    })),
  };
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
  setUnsavedEdits: (value: boolean, revision: number): Promise<void> =>
    Service.SetUnsavedEdits(value, revision),
  rememberOutputDirectory: (path: string): Promise<void> =>
    Service.RememberOutputDirectory(path),
  suggestOutputPath: (path: string): Promise<string> =>
    Service.SuggestOutputPath(path),
  chooseMediaPaths: async (): Promise<string[]> =>
    (await Service.ChooseMediaPaths()) ?? [],
  chooseOutputDirectory: async (suggested: string): Promise<string> =>
    Service.ChooseOutputDirectory(suggested),
  startBatch: async (projects: Project[]): Promise<Job[]> =>
    ((await Service.StartBatch(projects)) ?? []).map(jobView),
  queueStatus: async (): Promise<QueueStatus> => {
    const status = await Service.QueueStatus();
    return { ...status, pendingIDs: status.pendingIDs ?? [] };
  },
  setQueuePaused: async (paused: boolean): Promise<QueueStatus> => {
    const status = await Service.SetQueuePaused(paused);
    return { ...status, pendingIDs: status.pendingIDs ?? [] };
  },
  moveQueued: async (id: string, direction: number): Promise<QueueStatus> => {
    const status = await Service.MoveQueued(id, direction);
    return { ...status, pendingIDs: status.pendingIDs ?? [] };
  },
  subscribeQueue: (callback: (status: QueueStatus) => void): (() => void) =>
    Events.On("queue:updated", (event) =>
      callback({ ...event.data, pendingIDs: event.data.pendingIDs ?? [] }),
    ),
  saveProject: async (project: Project): Promise<string> =>
    Service.SaveProject(project),
  loadProject: async (): Promise<LoadedProject | null> => {
    const value = await Service.LoadProject();
    return value
      ? { media: mediaView(value.media), project: projectView(value.project) }
      : null;
  },
  importPreset: async (): Promise<WorkspacePreferences | null> => {
    const value = await Service.ImportPreset();
    return value ? preferencesView(value) : null;
  },
  exportPreset: async (id: string): Promise<string> => Service.ExportPreset(id),
  startSample: async (project: Project, seconds: number): Promise<Job> =>
    jobView(await Service.StartSample(project, seconds)),
  getSampleComparison: async (id: string): Promise<SampleComparison> => {
    const value = await Service.GetSampleComparison(id);
    return {
      ...value,
      original: mediaView(value.original),
      output: mediaView(value.output),
    };
  },
  deletePreset: async (id: string): Promise<WorkspacePreferences> =>
    preferencesView(await Service.DeletePreset(id)),
  confirmQuit: async (): Promise<void> => Service.ConfirmQuit(),
  onCloseRequested: (
    callback: (request: {
      jobs: number;
      downloading: boolean;
      unsaved?: boolean;
    }) => void,
  ): (() => void) =>
    Events.On("app:close-requested", (event) => callback(event.data)),
  openJobOutput: async (id: string, folder: boolean): Promise<void> =>
    Service.OpenJobOutput(id, folder),
  reuseJob: async (
    id: string,
  ): Promise<{ media: Media; settings: Settings }> => {
    const value = await Service.ReuseJob(id);
    return {
      media: mediaView(value.media),
      settings: settingsView(value.project.settings),
    };
  },
  getWorkspace: async (): Promise<{
    error: string;
    preferences: WorkspacePreferences;
  }> => {
    const value = await Service.GetWorkspace();
    return { ...value, preferences: preferencesView(value.preferences) };
  },
  saveSettings: async (
    settings: Settings,
    name: string,
  ): Promise<WorkspacePreferences> =>
    preferencesView(await Service.SaveSettings(settings, name)),
  getMediaPreviewURL: async (uri: string): Promise<string> =>
    Service.GetMediaPreviewURL(uri),
  getVideoFrame: async (uri: string, seconds: number): Promise<Thumbnail> =>
    Service.GetVideoFrame(uri, seconds),
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
