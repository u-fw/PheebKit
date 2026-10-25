import { t, useI18n } from "./i18n";
import { applyAppearance, useAppearance } from "./appearance";
import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { UnsavedDialog } from "./components/UnsavedDialog";
import type { Draft } from "./draft";
import { Home } from "./components/Home";
import { Icon } from "./components/Icon";
import type { IconName } from "./components/Icon";
import { WindowControls } from "./components/WindowControls";
import { useStudio } from "./hooks/useStudio";
import { BRAND_NAME, BRAND_ENGLISH } from "./brand";
import { BrandLogo } from "./components/BrandLogo";
import { API } from "./bridge";
import { ExitDialog } from "./components/ExitDialog";
import { BatchPanel } from "./components/BatchPanel";
import { ProjectActions } from "./components/ProjectActions";
import { ProjectLibrary } from "./components/ProjectLibrary";
import { EditorPreferences } from "./components/EditorPreferences";
import { useAutosaveDraft } from "./hooks/useAutosaveDraft";
import type { ProjectLibrary as Library, LoadedProject } from "./bridge";

const SettingsPage = lazy(() =>
  import("./components/SettingsPage").then((module) => ({
    default: module.SettingsPage,
  })),
);
const Compress = lazy(() =>
  import("./components/Compress").then((module) => ({
    default: module.Compress,
  })),
);
const Queue = lazy(() =>
  import("./components/Queue").then((module) => ({ default: module.Queue })),
);
const SampleComparison = lazy(() =>
  import("./components/SampleComparison").then((module) => ({
    default: module.SampleComparison,
  })),
);

type Page = "home" | "compress" | "queue" | "settings";
const navigation: {
  id: Page;
  label: string;
  english: string;
  icon: IconName;
}[] = [
  { id: "home", label: "工作台", english: "Workspace", icon: "home" },
  { id: "compress", label: "媒体处理", english: "Process", icon: "compress" },
  { id: "queue", label: "任务队列", english: "Queue", icon: "queue" },
  { id: "settings", label: "设置", english: "Settings", icon: "settings" },
];

export default function App() {
  const { language } = useI18n();
  const [page, setPage] = useState<Page>("home");
  const [compressVisited, setCompressVisited] = useState(false);
  const appearance = useAppearance();
  const { theme, preferences: appearancePreferences } = appearance;
  const draft = useRef<Draft | null>(null);
  const [editingDraft, setEditingDraft] = useState<Draft | null>(null);
  const [library, setLibrary] = useState<Library | null>(null);
  const draftRevision = useRef(0);
  const [guardError, setGuardError] = useState("");
  const [dirty, setDirty] = useState(false);
  const [pending, setPending] = useState<{
    action: () => Promise<void>;
  } | null>(null);
  const protect = useCallback((action: () => Promise<void>) => {
    if (draft.current?.dirty) setPending((current) => current ?? { action });
    else void action().catch((cause: unknown) => setGuardError(String(cause)));
  }, []);
  const studio = useStudio(protect);
  const { connected, setError } = studio;
  useEffect(() => {
    document.documentElement.lang = language;
    document.title =
      language === "en-US" ? "PheebKit" : "菲比工具箱 · PheebKit";
    if (!connected) return;
    void API.setLocale(language).catch((cause: unknown) =>
      setError(`语言设置同步失败：${String(cause)}`),
    );
  }, [language, connected, setError]);
  const refreshLibrary = useCallback(async () => {
    const value = await API.getProjectLibrary();
    setLibrary(value);
    if (value.error) setError(value.error);
  }, [setError]);
  useEffect(() => {
    if (!connected) return;
    let alive = true;
    void API.getProjectLibrary().then(
      (value) => {
        if (alive) {
          setLibrary(value);
          if (value.error) setError(value.error);
        }
      },
      (cause) => {
        if (alive) setError(`恢复记录读取失败：${String(cause)}`);
      },
    );
    return () => {
      alive = false;
    };
  }, [connected, setError]);
  const autosave = useAutosaveDraft(
    editingDraft,
    connected && !library?.error,
    library?.preferences,
  );
  const onDraft = useCallback((value: Draft) => {
    draft.current = value;
    setEditingDraft(value);
    setDirty(value.dirty);
  }, []);
  useEffect(() => {
    if (!connected) return;
    draftRevision.current = Math.max(Date.now(), draftRevision.current + 1);
    void API.setUnsavedEdits(dirty, draftRevision.current).catch(
      (cause: unknown) => setError(`未保存保护同步失败：${String(cause)}`),
    );
  }, [dirty, connected, setError]);
  async function saveDraft(): Promise<boolean> {
    const value = draft.current;
    if (!value) return false;
    const path = await API.saveProject(value.project);
    if (!path) return false;
    value.markSaved(value.project);
    studio.setNotice("");
    void refreshLibrary().catch((cause) =>
      setError(`最近项目刷新失败：${String(cause)}`),
    );
    return true;
  }
  const [comparisonID, setComparisonID] = useState<string | null>(null);
  const [closeRequest, setCloseRequest] = useState<{
    jobs: number;
    downloading: boolean;
    unsaved?: boolean;
  } | null>(null);
  useEffect(() => {
    if (!studio.connected) return;
    return API.onCloseRequested(setCloseRequest);
  }, [studio.connected]);
  const activeCount = studio.jobs.filter((job) =>
    ["queued", "preparing", "running", "cancelling"].includes(
      job.state.toLowerCase(),
    ),
  ).length;
  const currentNav = navigation.find((item) => item.id === page);

  useEffect(() => {
    applyAppearance(document.documentElement, {
      preferences: appearancePreferences,
      theme,
    });
  }, [appearancePreferences, theme]);
  const toggleTheme = () =>
    appearance.setAppearance({ mode: theme === "light" ? "dark" : "light" });
  function navigate(nextPage: Page) {
    setPage(nextPage);
    if (nextPage === "compress") setCompressVisited(true);
    document.documentElement.scrollTop = 0;
  }
  async function loadProject() {
    openStoredProject(API.loadProject);
  }
  function openStoredProject(loader: () => Promise<LoadedProject | null>) {
    protect(async () => {
      if (await studio.loadProject(loader)) {
        navigate("compress");
        void refreshLibrary().catch((cause) =>
          setError(`最近项目刷新失败：${String(cause)}`),
        );
      }
    });
  }

  return (
    <div
      className={`app-shell ${studio.connected && studio.platform === "windows" ? "desktop-native" : ""}`}
      data-page={page}
    >
      <a className="skip-link" href="#workspace">
        {t("跳到工作区")}
      </a>
      <aside className="sidebar glass" aria-label={t("工作空间侧栏")}>
        <a
          className="brand"
          href="#home"
          onClick={(event) => {
            event.preventDefault();
            navigate("home");
          }}
          aria-label={t("{{0}} 工作台", [t(BRAND_NAME)])}
        >
          <span className="brand-mark">
            <BrandLogo />
          </span>
          <span>
            <strong>{t(BRAND_NAME)}</strong>
            <small>{t(BRAND_ENGLISH)}</small>
          </span>
        </a>
        <div className="sidebar-label">{t("工作空间")}</div>
        <nav aria-label={t("主导航")}>
          {navigation.map((item) => (
            <button
              key={item.id}
              className={`nav-item ${page === item.id ? "active" : ""} ${item.id === "settings" ? "settings-nav" : ""}`}
              aria-current={page === item.id ? "page" : undefined}
              onClick={() => navigate(item.id)}
            >
              <Icon name={item.icon} />
              <span>{t(item.label)}</span>
              {item.id === "queue" && activeCount > 0 && (
                <span
                  className="nav-count"
                  aria-label={t("{{0}} 个进行中的任务", [activeCount])}
                >
                  {activeCount}
                </span>
              )}
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <div className="local-badge">
            <span className="tiny-dot" />
            {t("离线工作空间")}
          </div>
          <div className="build-tag">
            {t(BRAND_ENGLISH)} <span>{t(__PHEEBKIT_VERSION__)}</span>
          </div>
        </div>
      </aside>
      <div className="workspace-shell">
        <header className="topbar glass">
          <div className="workspace-title">
            <BrandLogo />
            <strong>{t(currentNav?.label)}</strong>
          </div>
          <div className="topbar-actions">
            <span
              className={`platform-badge ${studio.connected ? "connected" : ""}`}
            >
              <span className="tiny-dot" />
              {t(studio.connected ? "本地处理" : "界面预览")}
            </span>
            <button
              className="icon-button"
              onClick={toggleTheme}
              aria-label={t(
                theme === "light" ? "切换深色主题" : "切换浅色主题",
              )}
            >
              <Icon name={theme === "light" ? "moon" : "sun"} />
            </button>
            {studio.connected && studio.platform === "windows" && (
              <WindowControls onError={studio.setError} />
            )}
          </div>
        </header>
        <main id="workspace" tabIndex={-1} data-file-drop-target>
          {studio.environment &&
            studio.environment.configurable &&
            !studio.environment.ready &&
            page !== "settings" && (
              <div className="environment-alert" role="alert">
                <Icon name="info" />
                <div>
                  <strong>{t("需要配置 FFmpeg")}</strong>
                  <p>{t(studio.environment.message)}</p>
                </div>
                <button
                  className="button outlined"
                  onClick={() => navigate("settings")}
                >
                  {t("前往设置")}
                </button>
              </div>
            )}
          {(studio.error || guardError) && (
            <div className="app-alert" role="alert">
              <Icon name="info" />
              <span>{t(studio.error || guardError)}</span>
              <button
                className="icon-button"
                aria-label={t("关闭提示")}
                onClick={() => {
                  studio.setError("");
                  setGuardError("");
                }}
              >
                <Icon name="close" />
              </button>
            </div>
          )}
          {studio.notice && (
            <div className="surface import-progress" role="status">
              <Icon name="info" />
              <span>{t(studio.notice)}</span>
              <button
                className="icon-button"
                aria-label={t("关闭操作提示")}
                onClick={() => studio.setNotice("")}
              >
                <Icon name="close" />
              </button>
            </div>
          )}
          {studio.busy && studio.importProgress.total > 0 && (
            <div className="surface import-progress" role="status">
              <span>
                {t("分析媒体")} {studio.importProgress.completed} /{t(" ")}
                {studio.importProgress.total}
              </span>
              <progress
                value={studio.importProgress.completed}
                max={studio.importProgress.total}
                aria-label={t("媒体分析进度")}
              />
              <button className="button text" onClick={studio.stopImport}>
                {t("停止后续分析")}
              </button>
            </div>
          )}
          {(page === "home" || page === "compress") && (
            <section
              className={`surface media-collection ${page === "compress" ? "workbench-media-library" : ""}`}
              aria-label={t("已导入媒体与项目")}
            >
              <div className="section-title">
                <h2>
                  {t(
                    studio.mediaList.length > 1
                      ? `已导入 ${studio.mediaList.length} 个文件`
                      : "媒体与项目",
                  )}
                </h2>
                <div className="action-row">
                  <button
                    className="button outlined"
                    disabled={studio.busy || !studio.connected}
                    onClick={() =>
                      studio.mediaList.length > 0
                        ? void studio.chooseMedia(true)
                        : protect(() => studio.chooseMedia())
                    }
                  >
                    {t("添加媒体")}
                  </button>
                  <button
                    className="button text"
                    disabled={studio.busy || !studio.connected}
                    onClick={() => void loadProject()}
                  >
                    {t("打开项目")}
                  </button>
                </div>
              </div>
              {studio.mediaList.length > 1 && (
                <div className="media-collection-list">
                  {studio.mediaList.map((item) => (
                    <button
                      className={`button ${studio.media?.source.uri === item.source.uri ? "tonal" : "text"}`}
                      key={item.source.uri}
                      disabled={studio.busy}
                      aria-pressed={
                        studio.media?.source.uri === item.source.uri
                      }
                      onClick={() => {
                        if (studio.media?.source.uri !== item.source.uri)
                          protect(async () => {
                            studio.selectMedia(item);
                          });
                      }}
                    >
                      {item.name}
                    </button>
                  ))}
                </div>
              )}
              {studio.mediaList.length > 1 && (
                <p className="field-hint">
                  {t(
                    "选择一个文件查看和单独处理；媒体处理页下方可整批提交，最多 32 个文件。",
                  )}
                </p>
              )}
            </section>
          )}
          {page === "home" && (
            <Home
              media={studio.media}
              busy={studio.busy}
              connected={studio.connected}
              chooseMedia={() => protect(() => studio.chooseMedia())}
              compress={() => navigate("compress")}
            />
          )}
          {page === "home" && (
            <ProjectLibrary
              library={library}
              connected={connected}
              refresh={refreshLibrary}
              openRecent={(id) =>
                openStoredProject(() => API.openRecentProject(id))
              }
              restore={(id) =>
                openStoredProject(() => API.restoreRecoveryDraft(id))
              }
            />
          )}
          {compressVisited && (
            <div hidden={page !== "compress"}>
              <Suspense
                fallback={
                  <div className="page" role="status">
                    {t("正在加载工作区…")}
                  </div>
                }
              >
                <Compress
                  key={
                    studio.media
                      ? `${studio.media.source.uri}:${studio.media.sourceFingerprint || ""}:${studio.reuse?.nonce ?? ""}`
                      : "empty"
                  }
                  media={studio.media}
                  initialSettings={
                    studio.reuse?.settings ?? studio.savedSettings
                  }
                  initialOutput={studio.reuse?.output}
                  needsSave={studio.reuse?.needsSave}
                  undoLimit={library?.preferences.undoLimit ?? 50}
                  renderBatch={(settings) =>
                    studio.mediaList.length > 1 ? (
                      <BatchPanel
                        key={settings.operation || "compress"}
                        files={studio.mediaList}
                        settings={settings}
                        connected={studio.connected}
                        outputDirectory={studio.outputDirectory}
                        onOutputDirectory={studio.setOutputDirectory}
                        onNotice={studio.setError}
                        onJobs={studio.addJobs}
                        showQueue={() => navigate("queue")}
                      />
                    ) : null
                  }
                  onDraft={onDraft}
                  outputDirectory={studio.outputDirectory}
                  onOutputDirectory={studio.setOutputDirectory}
                  onNotice={studio.setError}
                  projectActions={(project, markSaved) => (
                    <ProjectActions
                      project={project}
                      onSaved={(project) => {
                        markSaved(project);
                        studio.setNotice("");
                        void refreshLibrary().catch((cause) =>
                          setError(`最近项目刷新失败：${String(cause)}`),
                        );
                      }}
                      dirty={dirty}
                      autosaveMessage={autosave.error || autosave.message}
                      autosaveError={Boolean(autosave.error)}
                      connected={studio.connected}
                      load={loadProject}
                      onJob={studio.addJob}
                      showQueue={() => navigate("queue")}
                    />
                  )}
                  onSaved={studio.setSavedSettings}
                  reused={Boolean(studio.reuse)}
                  visible={page === "compress"}
                  capabilities={studio.capabilities}
                  connected={studio.connected}
                  chooseMedia={() => protect(() => studio.chooseMedia())}
                  onJob={studio.addJob}
                  showQueue={() => navigate("queue")}
                />
              </Suspense>
            </div>
          )}
          {page === "queue" && (
            <Suspense
              fallback={
                <div className="page" role="status">
                  {t("正在加载任务队列…")}
                </div>
              }
            >
              <Queue
                jobs={studio.jobs}
                cancel={(id) => void studio.cancelJob(id)}
                refresh={() => void studio.refreshJobs()}
                compress={() => navigate("compress")}
                queueStatus={studio.queueStatus}
                onPauseQueue={studio.pauseQueue}
                onMoveQueued={studio.moveQueued}
                compareSample={(id) => {
                  setComparisonID(id);
                  requestAnimationFrame(() =>
                    document
                      .getElementById("sample-comparison")
                      ?.scrollIntoView({ block: "start" }),
                  );
                }}
                reuse={async (id) =>
                  protect(async () => {
                    const success = await studio.reuseJob(id);
                    if (success) navigate("compress");
                  })
                }
              />
              {comparisonID && (
                <div id="sample-comparison" className="page">
                  <SampleComparison
                    key={comparisonID}
                    id={comparisonID}
                    close={() => setComparisonID(null)}
                  />
                </div>
              )}
            </Suspense>
          )}
          {page === "settings" && (
            <Suspense
              fallback={
                <div className="page" role="status">
                  {t("正在加载设置…")}
                </div>
              }
            >
              <SettingsPage
                editorSettings={
                  library && (
                    <EditorPreferences
                      key={JSON.stringify(library.preferences)}
                      value={library.preferences}
                      save={async (value) => {
                        await API.saveEditorPreferences(value);
                        await refreshLibrary();
                        studio.setNotice("编辑与恢复设置已保存并生效。");
                      }}
                    />
                  )
                }
                platform={studio.platform}
                connected={studio.connected}
                capabilities={studio.capabilities}
                capabilityError={studio.capabilityError}
                environment={studio.environment}
                environmentBusy={
                  studio.environmentBusy || Boolean(studio.ffmpegSetup?.active)
                }
                ffmpegSetup={studio.ffmpegSetup}
                startFFmpegSetup={studio.startFFmpegSetup}
                startCustomFFmpegSetup={studio.startCustomFFmpegSetup}
                cancelFFmpegSetup={studio.cancelFFmpegSetup}
                refreshEnvironment={studio.refreshEnvironment}
                chooseFFmpeg={studio.chooseFFmpeg}
                configureFFmpeg={studio.configureFFmpeg}
              />
            </Suspense>
          )}
        </main>
      </div>
      <nav className="bottom-nav glass" aria-label={t("移动端主导航")}>
        {navigation.map((item) => (
          <button
            key={item.id}
            className={page === item.id ? "active" : ""}
            aria-current={page === item.id ? "page" : undefined}
            onClick={() => navigate(item.id)}
          >
            <span className="bottom-nav-icon">
              <Icon name={item.icon} />
              {item.id === "queue" && activeCount > 0 && (
                <span className="mobile-count">{activeCount}</span>
              )}
            </span>
            <span>{t(item.label)}</span>
          </button>
        ))}
      </nav>
      {closeRequest && (
        <ExitDialog
          request={{ ...closeRequest, unsaved: closeRequest.unsaved || dirty }}
          dismiss={() => setCloseRequest(null)}
          save={saveDraft}
        />
      )}
      {pending && (
        <UnsavedDialog
          save={saveDraft}
          discard={pending.action}
          dismiss={() => setPending(null)}
        />
      )}
    </div>
  );
}
