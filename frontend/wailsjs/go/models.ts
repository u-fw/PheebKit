export namespace app {
	
	export interface BinaryStatus {
	    path: string;
	    version: string;
	    source: string;
	    available: boolean;
	}
	export interface DiagnosticOptions {
	    includeRecentJobs: boolean;
	    jobLimit: number;
	}
	export interface DiagnosticPreview {
	    token: string;
	    json: string;
	}
	export interface DraftSummary {
	    id: string;
	    source: string;
	    updatedAt: string;
	    active: boolean;
	}
	export interface EditorPreferences {
	    autosave: boolean;
	    intervalSeconds: number;
	    draftLimit: number;
	    retentionDays: number;
	    recentLimit: number;
	    undoLimit: number;
	}
	export interface Environment {
	    platform: string;
	    ready: boolean;
	    ffmpeg: BinaryStatus;
	    ffprobe: BinaryStatus;
	    message: string;
	    configurable: boolean;
	}
	export interface FFmpegSetupStatus {
	    stage: string;
	    package: string;
	    source: string;
	    file: string;
	    received: number;
	    total: number;
	    speed: number;
	    active: boolean;
	    error: string;
	    version: string;
	}
	export interface JobReuse {
	    project: core.Project;
	    media: core.Media;
	    notice?: string;
	    needsSave?: boolean;
	}
	export interface ProjectLibrary {
	    preferences: EditorPreferences;
	    drafts: DraftSummary[] | null;
	    recent: RecentProject[] | null;
	    error: string;
	}
	export interface QueueStatus {
	    paused: boolean;
	    pendingIDs: string[];
	    revision: number;
	}
	export interface RecentProject {
	    id: string;
	    path: string;
	    updatedAt: string;
	}
	export interface SampleComparison {
	    original: core.Media;
	    output: core.Media;
	    startSeconds: number;
	    durationSeconds: number;
	}
	export interface Thumbnail {
	    dataUrl: string;
	    source: string;
	    width: number;
	    height: number;
	}
	export interface WorkspacePreferences {
	    version: number;
	    lastSettings?: core.Settings | null;
	    presets: core.Preset[] | null;
	    lastOutputDirectory?: string;
	}
	export interface WorkspaceStatus {
	    error: string;
	    preferences: WorkspacePreferences;
	}

}

export namespace core {
	
	export interface AudioProcessing {
	    normalize?: Loudness | null;
	    denoiseReductionDB: number;
	    noiseFloorDB: number;
	    equalizer: EqualizerBand[] | null;
	    fadeInSeconds: number;
	    fadeOutSeconds: number;
	    sampleRate: number;
	    channels: string;
	}
	export interface Capabilities {
	    version: string;
	    buildConfiguration: string;
	    encoders: Encoder[] | null;
	    codecs: Codec[] | null;
	    hardware: string[] | null;
	    platform: string;
	    videoEncoders: VideoEncoder[] | null;
	    gpus: GPU[] | null;
	    filters?: string[] | null;
	}
	export interface CatalogEntry {
	    name: string;
	    description: string;
	    flags: string;
	}
	export interface Codec {
	    id: string;
	    label: string;
	    encoder: string;
	    available: boolean;
	}
	export interface CropMargins {
	    top: number;
	    bottom: number;
	    left: number;
	    right: number;
	}
	export interface Encoder {
	    name: string;
	    description: string;
	    mediaType: string;
	}
	export interface EqualizerBand {
	    frequencyHz: number;
	    q: number;
	    gainDB: number;
	}
	export interface Source {
	    uri: string;
	    displayName: string;
	    kind: string;
	}
	export interface ExecutionPlan {
	    jobID: string;
	    projectID: string;
	    input: Source;
	    sourceFingerprint?: string;
	    output: Source;
	    args: string[] | null;
	    preview: string;
	    durationSeconds: number;
	    inputSizeBytes: number;
	    encoder: string;
	    encoderKind: string;
	    encoderLabel: string;
	    encoderReason: string;
	    trim?: TimeRange | null;
	    subtitleAsset?: SubtitleAsset | null;
	    fullDecodeVerification?: boolean;
	    expectedOutput?: OutputExpectation | null;
	}
	export interface GPU {
	    name: string;
	    vendor: string;
	}
	export interface Progress {
	    percent: number;
	    fps: number;
	    speed: number;
	    currentTimeSeconds: number;
	    totalTimeSeconds: number;
	    outputSizeBytes: number;
	    frame: number;
	    finished: boolean;
	}
	export interface Job {
	    id: string;
	    projectID: string;
	    input: Source;
	    sourceFingerprint?: string;
	    output: Source;
	    state: string;
	    queuedAt: string;
	    startedAt: string;
	    completedAt: string;
	    progress: Progress;
	    result?: Result | null;
	    error: string;
	    logs: string[] | null;
	    trim?: TimeRange | null;
	    project?: Project | null;
	}
	export interface Loudness {
	    integratedLUFS: number;
	    truePeakDBTP: number;
	    rangeLU: number;
	}
	export interface Media {
	    source: Source;
	    sourceFingerprint?: string;
	    name: string;
	    container: string;
	    sizeBytes: number;
	    durationSeconds: number;
	    bitrate: number;
	    streams: Stream[] | null;
	}
	export interface MediaHealth {
	    id: string;
	    uri: string;
	    sourceFingerprint: string;
	    state: string;
	    percent: number;
	    durationSeconds: number;
	    windowSeconds: number;
	    damagedRanges: TimeRange[] | null;
	    error: string;
	}
	export interface OutputExpectation {
	    audioSampleRate?: number;
	    audioChannels?: number;
	    audioSourceChannels?: number[] | null;
	    videoStreams: number;
	    audioStreams: number;
	    subtitleStreams: number;
	    videoCodec?: string;
	}
	export interface OutputVerification {
	    fullDecodePassed?: boolean;
	    durationSeconds: number;
	    videoStreams: number;
	    audioStreams: number;
	    subtitleStreams: number;
	    warnings: string[] | null;
	}
	export interface Settings {
	    audioProcessing?: AudioProcessing | null;
	    recovery?: Recovery | null;
	    operation?: string;
	    codec: string;
	    encoder: string;
	    quality: number;
	    preset: string;
	    resolution: number;
	    audio: string;
	    audioBitrateKbps?: number;
	    mode: string;
	    bitrateKbps: number;
	    targetSizeMB: number;
	    rawArguments: string[] | null;
	    trim?: TimeRange | null;
	    segments?: TimeRange[] | null;
	    subtitleBurn?: SubtitleBurn | null;
	    audioSelection?: TrackSelection | null;
	    subtitleSelection?: TrackSelection | null;
	    transform?: VideoTransform | null;
	    volumeDB?: number;
	}
	export interface Preset {
	    schemaVersion: number;
	    id: string;
	    name: string;
	    settings: Settings;
	}
	
	export interface Project {
	    schemaVersion: number;
	    id: string;
	    inputs: Source[] | null;
	    output: Source;
	    settings: Settings;
	}
	export interface Recovery {
	    mode: string;
	    sourceFingerprint: string;
	    damagedRanges: TimeRange[] | null;
	}
	export interface Result {
	    inputSizeBytes: number;
	    outputSizeBytes: number;
	    savedPercent: number;
	    elapsedSeconds: number;
	    averageSpeed: number;
	    output: Source;
	    verification?: OutputVerification | null;
	}
	
	
	export interface Stream {
	    sampleRate?: number;
	    channels?: number;
	    channelLayout?: string;
	    index: number;
	    codecType: string;
	    codecName: string;
	    width: number;
	    height: number;
	    displayRotation?: number;
	    frameRate: number;
	    pixelFormat: string;
	    bitDepth: number;
	    colorSpace: string;
	    colorTransfer: string;
	    colorPrimaries: string;
	    hdr: string;
	    language: string;
	    title: string;
	    attachedPicture: boolean;
	    timedThumbnails: boolean;
	}
	export interface SubtitleAsset {
	    source: Source;
	    fingerprint: string;
	    streamIndex: number;
	    format: string;
	}
	export interface SubtitleBurn {
	    source: string;
	    path?: string;
	    streamIndex: number;
	    offsetSeconds: number;
	    font: string;
	    fontSize: number;
	    color: string;
	    outlineColor: string;
	    outline: number;
	    shadow: number;
	    alignment: number;
	    margin: number;
	    bold: boolean;
	}
	export interface TimeRange {
	    startSeconds: number;
	    endSeconds: number;
	}
	export interface TrackSelection {
	    mode: string;
	    indices?: number[] | null;
	}
	export interface VideoEncoder {
	    name: string;
	    codec: string;
	    label: string;
	    kind: string;
	    available: boolean;
	    reason: string;
	    pixelFormats: string[] | null;
	}
	export interface VideoTransform {
	    rotation: number;
	    flipHorizontal: boolean;
	    flipVertical: boolean;
	    crop?: CropMargins | null;
	}

}

