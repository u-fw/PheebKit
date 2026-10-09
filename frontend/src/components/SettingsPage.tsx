import type { Capabilities } from "../bridge";
import { Icon } from "./Icon";

export function SettingsPage({
  platform,
  connected,
  capabilities,
  capabilityError,
  theme,
  toggleTheme,
}: {
  platform: string;
  connected: boolean;
  capabilities: Capabilities | null;
  capabilityError: string;
  theme: "light" | "dark";
  toggleTheme: () => void;
}) {
  return (
    <div className="page settings-page">
      <header className="page-heading">
        <div>
          <div className="section-eyebrow">SETTINGS & CAPABILITIES</div>
          <h1>你的工作台，按你的习惯。</h1>
          <p>外观、平台状态与当前 FFmpeg 构建能力。</p>
        </div>
      </header>
      <section className="surface preferences-card">
        <div className="section-title">
          <h2>外观</h2>
          <Icon name="sun" />
        </div>
        <div className="preference-row">
          <div>
            <h3>主题</h3>
            <p>柔和的明亮界面，或专注的深色工作台。</p>
          </div>
          <button className="button outlined" onClick={toggleTheme}>
            <Icon name={theme === "light" ? "moon" : "sun"} />
            {theme === "light" ? "切换深色" : "切换浅色"}
          </button>
        </div>
      </section>
      <section className="surface preferences-card">
        <div className="section-title">
          <h2>运行环境</h2>
          <span className={`status-pill ${connected ? "connected" : ""}`}>
            <span className="tiny-dot" />
            {connected ? "原生桥接已连接" : "原生桥接不可用"}
          </span>
        </div>
        <dl className="environment-grid">
          <div>
            <dt>GetPlatform()</dt>
            <dd>{platform}</dd>
          </div>
          <div>
            <dt>版本阶段</dt>
            <dd>Technical Spike</dd>
          </div>
          <div>
            <dt>媒体处理</dt>
            <dd>本地设备</dd>
          </div>
          <div>
            <dt>执行适配器</dt>
            <dd>
              {platform === "android"
                ? "Android MockRunner"
                : platform === "windows"
                  ? "Windows FFmpegRunner"
                  : "未连接"}
            </dd>
          </div>
        </dl>
        {platform === "android" && (
          <div className="notice">
            <Icon name="info" />
            <p>
              Android 共享 UI 和 Go binding 仍需在设备上验证。MockRunner
              会明确拒绝 native FFmpeg 操作；真实 probe、编码和 SAF 输出将在
              Android FFmpeg Spike 验证。
            </p>
          </div>
        )}
      </section>
      <section className="surface preferences-card">
        <div className="section-title">
          <h2>FFmpeg 能力</h2>
          <span className="subtle-label">来自实际构建</span>
        </div>
        {capabilities ? (
          <>
            <pre className="version-text">
              {capabilities.version || "未返回版本信息"}
            </pre>
            <h3>{capabilities.encoders.length} 个编码器</h3>
            {capabilities.encoders.length ? (
              <div className="encoder-chips">
                {capabilities.encoders.map((encoder) => (
                  <span key={encoder} className="chip">
                    {encoder}
                  </span>
                ))}
              </div>
            ) : (
              <p>此平台适配器尚未提供可用编码器。</p>
            )}
          </>
        ) : (
          <div className="notice">
            <Icon name="info" />
            <p>{capabilityError || "正在查询当前 FFmpeg 构建…"}</p>
          </div>
        )}
      </section>
    </div>
  );
}
