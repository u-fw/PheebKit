package app

import "pheebkit/internal/core"

// NativeHost keeps framework APIs out of the media service and its tests.
type NativeHost interface {
	Emit(string, ...interface{})
	Quit()
	OpenFile(FileDialog) (string, error)
	OpenFiles(FileDialog) ([]string, error)
	OpenDirectory(FileDialog) (string, error)
	SaveFile(FileDialog) (string, error)
}

type FileDialog struct {
	Title, Directory, Filename, FilterName, Pattern string
}

// Desktop binds only the reviewed application API. Lifecycle hooks and the
// media asset handler must never be callable through renderer IPC.
type Desktop struct{ DesktopAPI }

type DesktopAPI interface {
	Cancel(id string) error
	CancelFFmpegSetup() error
	CancelMediaHealthScan(id string) error
	Capabilities() (core.Capabilities, error)
	ChooseFFmpeg() (*Environment, error)
	ChooseMedia() (*core.Media, error)
	ChooseMediaPaths() ([]string, error)
	ChooseOutput(suggested string) (string, error)
	ChooseOutputDirectory(suggested string) (string, error)
	ChooseSubtitleFile() (string, error)
	ClearRecentProjects() error
	ClearRecoveryDraft(revision uint64) error
	ConfigureFFmpeg(ffmpegPath, ffprobePath string) (Environment, error)
	ConfirmQuit()
	DeletePreset(id string) (WorkspacePreferences, error)
	ExportDiagnosticReport(token string) (string, error)
	ExportPreset(id string) (string, error)
	GetDiagnosticPreview(options DiagnosticOptions) (DiagnosticPreview, error)
	GetEnvironment() Environment
	GetFFmpegCatalog(category string) ([]core.CatalogEntry, error)
	GetFFmpegOptionHelp(category, name string) (string, error)
	GetFFmpegSetupStatus() FFmpegSetupStatus
	GetMediaHealthScan() core.MediaHealth
	GetMediaPreviewURL(uri string) (string, error)
	GetPlatform() string
	GetProjectLibrary() ProjectLibrary
	GetSampleComparison(id string) (SampleComparison, error)
	GetThumbnail(uri string) (Thumbnail, error)
	GetVideoFrame(uri string, seconds float64) (Thumbnail, error)
	GetWorkspace() WorkspaceStatus
	ImportPreset() (*WorkspacePreferences, error)
	Jobs() []core.Job
	LoadProject() (*JobReuse, error)
	MoveQueued(id string, direction int) (QueueStatus, error)
	OpenJobOutput(id string, folder bool) error
	OpenRecentProject(id string) (*JobReuse, error)
	Preview(p core.Project) (core.ExecutionPlan, error)
	Probe(uri string) (core.Media, error)
	QueueStatus() QueueStatus
	RefreshCapabilities() (core.Capabilities, error)
	RememberOutputDirectory(output string) error
	RemoveRecoveryDraft(id string) error
	RestoreRecoveryDraft(id string) (*JobReuse, error)
	ReuseJob(id string) (JobReuse, error)
	SaveEditorPreferences(p EditorPreferences) error
	SaveProject(p core.Project) (string, error)
	SaveRecoveryDraft(p core.Project, revision uint64) error
	SaveSettings(settings core.Settings, name string) (WorkspacePreferences, error)
	SetActiveMedia(uris []string, revision uint64) error
	SetQueuePaused(paused bool) (QueueStatus, error)
	SetLocale(language string) error
	SetUnsavedEdits(unsaved bool, revision uint64)
	Start(p core.Project) (core.Job, error)
	StartBatch(projects []core.Project) ([]core.Job, error)
	StartCustomFFmpegSetup(link, sha256 string, trustBuild bool) (FFmpegSetupStatus, error)
	StartFFmpegSetup(source string) (FFmpegSetupStatus, error)
	StartMediaHealthScan(uri string, windowSeconds int) (core.MediaHealth, error)
	StartSample(p core.Project, seconds float64) (core.Job, error)
	SuggestOutputPath(output string) (string, error)
}

var _ DesktopAPI = (*Service)(nil)
