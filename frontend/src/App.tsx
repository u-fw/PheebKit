import { useEffect, useState } from "react";
import { Compress } from "./components/Compress";
import { Home } from "./components/Home";
import { Icon } from "./components/Icon";
import type { IconName } from "./components/Icon";
import { Queue } from "./components/Queue";
import { SettingsPage } from "./components/SettingsPage";
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
    <div className="app-shell">
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
            <small>MEDIA WORKSPACE</small>
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
            LOCAL FIRST
          </div>
          <p>
            媒体留在你的设备。
            <br />
            创作由你掌控。
          </p>
          <div className="build-tag">
            创作，从容一点 <span>0.1</span>
          </div>
        </div>
      </aside>
      <div className="workspace-shell">
        <header className="topbar glass">
          <div className="breadcrumb">
            <span>kompeg</span>
            <Icon name="chevron" />
            <strong>{currentNav?.label}</strong>
          </div>
          <div className="topbar-actions">
            <span
              className={`platform-badge ${studio.connected ? "connected" : ""}`}
            >
              <span className="tiny-dot" />
              {studio.platform}
            </span>
            <button
              className="icon-button"
              onClick={toggleTheme}
              aria-label={theme === "light" ? "切换深色主题" : "切换浅色主题"}
            >
              <Icon name={theme === "light" ? "moon" : "sun"} />
            </button>
          </div>
        </header>
        <main id="workspace" tabIndex={-1} data-file-drop-target>
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
                key={studio.media?.source.uri || "empty"}
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
              theme={theme}
              toggleTheme={toggleTheme}
            />
          )}
        </main>
        <footer className="workspace-footer">
          <span>kompeg</span>
          <span>让媒体处理，更简单。</span>
        </footer>
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
