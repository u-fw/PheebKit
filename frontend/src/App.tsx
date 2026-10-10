import { useEffect, useState } from "react";
import { Compress } from "./components/Compress";
import { Home } from "./components/Home";
import { Icon } from "./components/Icon";
import type { IconName } from "./components/Icon";
import { Queue } from "./components/Queue";
import { SettingsPage } from "./components/SettingsPage";
import { WindowControls } from "./components/WindowControls";
import { useStudio } from "./hooks/useStudio";

type Page = "home" | "compress" | "queue" | "settings";
const navigation: {
  id: Page;
  label: string;
  english: string;
  icon: IconName;
}[] = [
  { id: "home", label: "工作台", english: "Workspace", icon: "home" },
  { id: "compress", label: "压缩", english: "Compress", icon: "compress" },
  { id: "queue", label: "任务队列", english: "Queue", icon: "queue" },
  { id: "settings", label: "设置", english: "Settings", icon: "settings" },
];

function initialTheme(): "light" | "dark" {
  const stored = localStorage.getItem("kompeg-theme");
  if (stored === "light" || stored === "dark") return stored;
  return matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export default function App() {
  const [page, setPage] = useState<Page>("home");
  const [compressVisited, setCompressVisited] = useState(false);
  const [theme, setTheme] = useState<"light" | "dark">(initialTheme);
  const studio = useStudio();
  const activeCount = studio.jobs.filter((job) =>
    ["queued", "preparing", "running", "cancelling"].includes(
      job.state.toLowerCase(),
    ),
  ).length;
  const currentNav = navigation.find((item) => item.id === page);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("kompeg-theme", theme);
  }, [theme]);
  const toggleTheme = () =>
    setTheme((value) => (value === "light" ? "dark" : "light"));
  function navigate(nextPage: Page) {
    setPage(nextPage);
    if (nextPage === "compress") setCompressVisited(true);
    document.documentElement.scrollTop = 0;
  }

  return (
    <div
      className={`app-shell ${studio.connected && studio.platform === "windows" ? "desktop-native" : ""}`}
    >
      <a className="skip-link" href="#workspace">
        跳到工作区
      </a>
      <aside className="sidebar glass" aria-label="工作空间侧栏">
        <a
          className="brand"
          href="#home"
          onClick={(event) => {
            event.preventDefault();
            navigate("home");
          }}
          aria-label="kompeg 工作台"
        >
          <span className="brand-mark">
            <Icon name="studio" />
          </span>
          <span>
            <strong>kompeg</strong>
            <small>本地媒体工作台</small>
          </span>
        </a>
        <div className="sidebar-label">工作空间</div>
        <nav aria-label="主导航">
          {navigation.map((item) => (
            <button
              key={item.id}
              className={`nav-item ${page === item.id ? "active" : ""} ${item.id === "settings" ? "settings-nav" : ""}`}
              aria-current={page === item.id ? "page" : undefined}
              onClick={() => navigate(item.id)}
            >
              <Icon name={item.icon} />
              <span>{item.label}</span>
              {item.id === "queue" && activeCount > 0 && (
                <span
                  className="nav-count"
                  aria-label={`${activeCount} 个进行中的任务`}
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
            离线工作空间
          </div>
          <div className="build-tag">
            kompeg <span>0.1</span>
          </div>
        </div>
      </aside>
      <div className="workspace-shell">
        <header className="topbar glass">
          <div className="workspace-title">
            <Icon name={currentNav?.icon || "home"} />
            <strong>{currentNav?.label}</strong>
          </div>
          <div className="topbar-actions">
            <span
              className={`platform-badge ${studio.connected ? "connected" : ""}`}
            >
              <span className="tiny-dot" />
              {studio.connected ? "本地处理" : "界面预览"}
            </span>
            <button
              className="icon-button"
              onClick={toggleTheme}
              aria-label={theme === "light" ? "切换深色主题" : "切换浅色主题"}
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
                  <strong>需要配置 FFmpeg</strong>
                  <p>{studio.environment.message}</p>
                </div>
                <button
                  className="button outlined"
                  onClick={() => navigate("settings")}
                >
                  前往设置
                </button>
              </div>
            )}
          {studio.error && (
            <div className="app-alert" role="alert">
              <Icon name="info" />
              <span>{studio.error}</span>
              <button
                className="icon-button"
                aria-label="关闭提示"
                onClick={() => studio.setError("")}
              >
                <Icon name="close" />
              </button>
            </div>
          )}
          {page === "home" && (
            <Home
              media={studio.media}
              busy={studio.busy}
              connected={studio.connected}
              chooseMedia={() => void studio.chooseMedia()}
              compress={() => navigate("compress")}
            />
          )}
          {compressVisited && (
            <div hidden={page !== "compress"}>
              <Compress
                key={
                  studio.media
                    ? `${studio.media.source.uri}:${studio.media.sourceFingerprint || ""}`
                    : "empty"
                }
                media={studio.media}
                capabilities={studio.capabilities}
                connected={studio.connected}
                chooseMedia={() => void studio.chooseMedia()}
                onJob={studio.addJob}
                showQueue={() => navigate("queue")}
              />
            </div>
          )}
          {page === "queue" && (
            <Queue
              jobs={studio.jobs}
              cancel={(id) => void studio.cancelJob(id)}
              refresh={() => void studio.refreshJobs()}
              compress={() => navigate("compress")}
            />
          )}
          {page === "settings" && (
            <SettingsPage
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
              theme={theme}
              toggleTheme={toggleTheme}
            />
          )}
        </main>
      </div>
      <nav className="bottom-nav glass" aria-label="移动端主导航">
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
            <span>{item.label}</span>
          </button>
        ))}
      </nav>
    </div>
  );
}
