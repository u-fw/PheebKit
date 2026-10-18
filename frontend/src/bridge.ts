import {
  EventsOn,
  OnFileDrop,
  OnFileDropOff,
} from "../wailsjs/runtime/runtime";
import * as Service from "../wailsjs/go/app/Desktop";
import { getLanguage } from "./i18n";
import type { core as Core, app as Models } from "../wailsjs/go/models";
type FFmpegSetupStatus = Models.FFmpegSetupStatus;
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
export type TimeRange = Core.TimeRange;
export type SubtitleBurn = Core.SubtitleBurn;
export type CatalogEntry = Core.CatalogEntry;
export type MediaHealth = Core.MediaHealth;
export type AudioProcessing = Core.AudioProcessing;
export type EqualizerBand = Core.EqualizerBand;
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
  filters?: string[];
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
  verification?: Core.OutputVerification | null;
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
  finishing?: boolean;
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
  notice?: string;
  needsSave?: boolean;
  media: Media;
  project: Project;
}

export interface EditorPreferences {
  autosave: boolean;
  intervalSeconds: number;
  draftLimit: number;
  retentionDays: number;
  recentLimit: number;
  undoLimit: number;
}
export interface ProjectLibrary {
  preferences: EditorPreferences;
  drafts: { id: string; source: string; updatedAt: string; active: boolean }[];
  recent: { id: string; path: string; updatedAt: string }[];
  error: string;
}

function loadedProjectView(
  value: Awaited<ReturnType<typeof Service.LoadProject>>,
): LoadedProject | null {
  return value
    ? {
        media: mediaView(value.media),
        project: projectView(value.project),
        notice: value.notice,
        needsSave: value.needsSave,
      }
    : null;
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
    verification: job.result?.verification,
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
    finishing: job.state === "running" && job.progress.finished,
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
    filters: caps.filters ?? [],
    encoders: (caps.encoders ?? []).map((encoder) => encoder.name),
    codecs: caps.codecs ?? [],
    videoEncoders: (caps.videoEncoders ?? []).map((encoder) => ({
      ...encoder,
      pixelFormats: encoder.pixelFormats ?? [],
    })),
    gpus: caps.gpus ?? [],
  };
}

async function localizedDialog<T>(action: () => Promise<T>): Promise<T> {
  await Service.SetLocale(getLanguage());
  return action();
}

export const API = {
  setLocale: (language: "zh-CN" | "en-US"): Promise<void> =>
    Service.SetLocale(language),
  startMediaHealthScan: (
    uri: string,
    windowSeconds: number,
  ): Promise<MediaHealth> => Service.StartMediaHealthScan(uri, windowSeconds),
  getMediaHealthScan: (): Promise<MediaHealth> => Service.GetMediaHealthScan(),
  cancelMediaHealthScan: (id: string): Promise<void> =>
    Service.CancelMediaHealthScan(id),
  getFFmpegCatalog: async (category: string): Promise<CatalogEntry[]> =>
    (await Service.GetFFmpegCatalog(category)) ?? [],
  getFFmpegOptionHelp: (category: string, name: string): Promise<string> =>
    Service.GetFFmpegOptionHelp(category, name),
  chooseSubtitleFile: (): Promise<string> =>
    localizedDialog(Service.ChooseSubtitleFile),
  getDiagnosticPreview: (
    includeRecentJobs: boolean,
    jobLimit: number,
  ): Promise<{ token: string; json: string }> =>
    Service.GetDiagnosticPreview({ includeRecentJobs, jobLimit }),
  exportDiagnosticReport: (token: string): Promise<string> =>
    localizedDialog(() => Service.ExportDiagnosticReport(token)),
  getProjectLibrary: async (): Promise<ProjectLibrary> => {
    const value = await Service.GetProjectLibrary();
    return { ...value, drafts: value.drafts ?? [], recent: value.recent ?? [] };
  },
  saveEditorPreferences: (preferences: EditorPreferences): Promise<void> =>
    Service.SaveEditorPreferences(preferences),
  saveRecoveryDraft: (project: Project, revision: number): Promise<void> =>
    Service.SaveRecoveryDraft(project, revision),
  clearRecoveryDraft: (revision: number): Promise<void> =>
    Service.ClearRecoveryDraft(revision),
  removeRecoveryDraft: (id: string): Promise<void> =>
    Service.RemoveRecoveryDraft(id),
  clearRecentProjects: (): Promise<void> => Service.ClearRecentProjects(),
  openRecentProject: async (id: string): Promise<LoadedProject | null> =>
    loadedProjectView(
      await localizedDialog(() => Service.OpenRecentProject(id)),
    ),
  restoreRecoveryDraft: async (id: string): Promise<LoadedProject | null> =>
    loadedProjectView(await Service.RestoreRecoveryDraft(id)),
  setActiveMedia: (uris: string[], revision: number): Promise<void> =>
    Service.SetActiveMedia(uris, revision),
  setUnsavedEdits: (value: boolean, revision: number): Promise<void> =>
    Service.SetUnsavedEdits(value, revision),
  rememberOutputDirectory: (path: string): Promise<void> =>
    Service.RememberOutputDirectory(path),
  suggestOutputPath: (path: string): Promise<string> =>
    Service.SuggestOutputPath(path),
  chooseMediaPaths: async (): Promise<string[]> =>
    (await localizedDialog(Service.ChooseMediaPaths)) ?? [],
  chooseOutputDirectory: async (suggested: string): Promise<string> =>
    localizedDialog(() => Service.ChooseOutputDirectory(suggested)),
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
    EventsOn("queue:updated", (status: QueueStatus) =>
      callback({ ...status, pendingIDs: status.pendingIDs ?? [] }),
    ),
  saveProject: async (project: Project): Promise<string> =>
    localizedDialog(() => Service.SaveProject(project)),
  loadProject: async (): Promise<LoadedProject | null> => {
    return loadedProjectView(await localizedDialog(Service.LoadProject));
  },
  importPreset: async (): Promise<WorkspacePreferences | null> => {
    const value = await localizedDialog(Service.ImportPreset);
    return value ? preferencesView(value) : null;
  },
  exportPreset: async (id: string): Promise<string> =>
    localizedDialog(() => Service.ExportPreset(id)),
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
    EventsOn("app:close-requested", (request: Parameters<typeof callback>[0]) =>
      callback(request),
    ),
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
    localizedDialog(Service.ChooseFFmpeg),
  getThumbnail: async (uri: string): Promise<Thumbnail> =>
    Service.GetThumbnail(uri),
  chooseMedia: async (): Promise<Media | null> => {
    const media = await localizedDialog(Service.ChooseMedia);
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
    localizedDialog(() => Service.ChooseOutput(suggested)),
  subscribeJobs: (callback: (job: Job) => void): (() => void) =>
    EventsOn("job:updated", (job: Core.Job) => callback(jobView(job))),
  subscribeMedia: (callback: (media: Media) => void): (() => void) =>
    EventsOn("media:probed", (media: Core.Media) => callback(mediaView(media))),
  onDrop: (callback: (paths: string[]) => void): (() => void) => {
    OnFileDrop((_x, _y, paths) => callback(paths), false);
    return OnFileDropOff;
  },
};
