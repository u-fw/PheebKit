import { t } from "../i18n";
import { useEffect, useState } from "react";
import { API, type CatalogEntry } from "../bridge";
import { errorMessage } from "../format";
import { MaterialSelect } from "./MaterialSelect";

export const catalogCategories = [
  { value: "encoders", label: "编码器" },
  { value: "decoders", label: "解码器" },
  { value: "filters", label: "滤镜" },
  { value: "muxers", label: "封装器" },
  { value: "demuxers", label: "解封装器" },
  { value: "protocols", label: "协议" },
  { value: "bsfs", label: "码流滤镜" },
  { value: "pixel-formats", label: "像素格式" },
  { value: "sample-formats", label: "音频采样格式" },
  { value: "layouts", label: "声道布局" },
  { value: "colors", label: "颜色" },
  { value: "hardware", label: "硬件后端" },
  { value: "devices", label: "设备接口" },
];
const helpCategories = new Set([
  "encoders",
  "decoders",
  "filters",
  "muxers",
  "demuxers",
  "protocols",
  "bsfs",
]);
export function FFmpegExplorer({ connected }: { connected: boolean }) {
  const [open, setOpen] = useState(false);
  const [category, setCategory] = useState("filters");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(0);
  const [revision, setRevision] = useState(0);
  const [result, setResult] = useState<{
    key: string;
    entries: CatalogEntry[];
    error: string;
  }>({ key: "", entries: [], error: "" });
  const [help, setHelp] = useState<{ name: string; text: string } | null>(null);
  const [helpBusy, setHelpBusy] = useState(false);
  const [helpError, setHelpError] = useState("");
  const key = `${category}/${revision}`;
  useEffect(() => {
    if (!open || !connected) return;
    let alive = true;
    void API.getFFmpegCatalog(category)
      .then((entries) => {
        if (alive) setResult({ key, entries, error: "" });
      })
      .catch((cause: unknown) => {
        if (alive) setResult({ key, entries: [], error: errorMessage(cause) });
      });
    return () => {
      alive = false;
    };
  }, [open, connected, category, key]);
  const busy = open && connected && result.key !== key;
  const entries = result.key === key ? result.entries : [];
  const search = query.trim().toLowerCase();
  const found = entries.filter((entry) =>
    `${entry.name} ${entry.description}`.toLowerCase().includes(search),
  );
  const current = Math.min(page, Math.max(0, Math.ceil(found.length / 50) - 1));
  async function readHelp(name: string, kind = category) {
    setHelpBusy(true);
    setHelpError("");
    setHelp(null);
    try {
      const text = await API.getFFmpegOptionHelp(kind, name);
      setHelp({ name: name || "完整参数帮助", text });
    } catch (cause) {
      setHelpError(errorMessage(cause));
    } finally {
      setHelpBusy(false);
    }
  }
  return (
    <section className="surface preferences-card ffmpeg-explorer">
      <details onToggle={(event) => setOpen(event.currentTarget.open)}>
        <summary>{t("FFmpeg 能力与参数帮助")}</summary>
        <p className="field-hint">
          {t(
            "来自当前选定 FFmpeg 的实际目录。列出表示构建中包含此功能，不代表 GPU、设备或处理流程已经验证；也不代表界面已提供该功能的执行入口。",
          )}
        </p>
        <div className="field-grid">
          <MaterialSelect
            label={t("能力类别")}
            value={category}
            options={catalogCategories}
            disabled={helpBusy}
            onChange={(value) => {
              setCategory(value);
              setPage(0);
              setQuery("");
              setHelp(null);
              setHelpError("");
            }}
          />
          <label className="field-label">
            {t("搜索名称或说明")}
            <input
              type="search"
              value={query}
              onChange={(event) => {
                setQuery(event.target.value);
                setPage(0);
              }}
              placeholder={t("例如 subtitles、loudnorm、libx264")}
            />
          </label>
        </div>
        <div className="action-row">
          <button
            className="button outlined"
            disabled={!connected || busy || helpBusy}
            onClick={() => {
              setRevision((v) => v + 1);
              setHelp(null);
            }}
          >
            {t("重新读取目录")}
          </button>
          <button
            className="button text"
            disabled={!connected || helpBusy}
            onClick={() => void readHelp("", "full")}
          >
            {t("查看完整参数帮助")}
          </button>
        </div>
        {!connected ? (
          <p className="field-hint">{t("连接桌面服务后可读取本机能力。")}</p>
        ) : busy ? (
          <p role="status">{t("正在读取本机 FFmpeg…")}</p>
        ) : result.error ? (
          <p className="inline-error" role="alert">
            {t(result.error)}
          </p>
        ) : (
          <>
            <p className="field-hint" role="status">
              {t("匹配")} {found.length} / {entries.length}{" "}
              {t("项，每页最多显示 50 项。")}
            </p>
            <ul className="catalog-list">
              {found.slice(current * 50, (current + 1) * 50).map((entry) => (
                <li key={entry.name}>
                  <div>
                    <code>{entry.name}</code>
                    <p>{t(entry.description || "当前构建提供此项目")}</p>
                    {entry.flags && (
                      <small className="field-hint">
                        {t("构建标记：")} {t(entry.flags)}
                      </small>
                    )}
                  </div>
                  {helpCategories.has(category) && (
                    <button
                      className="button text"
                      disabled={helpBusy}
                      aria-label={t("查看 {{0}} 参数", [entry.name])}
                      onClick={() => void readHelp(entry.name)}
                    >
                      {t("参数帮助")}
                    </button>
                  )}
                </li>
              ))}
            </ul>
            <div className="action-row">
              <button
                className="button outlined"
                disabled={current === 0}
                onClick={() => setPage(current - 1)}
              >
                {t("上一页")}
              </button>
              <span>
                {t("第")} {current + 1} /{" "}
                {Math.max(1, Math.ceil(found.length / 50))}
                {t(" ")}
                {t("页")}
              </span>
              <button
                className="button outlined"
                disabled={(current + 1) * 50 >= found.length}
                onClick={() => setPage(current + 1)}
              >
                {t("下一页")}
              </button>
            </div>
          </>
        )}
        {helpBusy && <p role="status">{t("正在读取参数帮助…")}</p>}
        {helpError && (
          <p className="inline-error" role="alert">
            {t(helpError)}
          </p>
        )}
        {help && (
          <>
            <h3>{help.name}</h3>
            <p className="field-hint">
              {t(
                "以下为本机 FFmpeg 原始帮助，可能为英文。参数的实际使用仍需要对应流程和校验。",
              )}
            </p>
            <pre
              className="diagnostic-preview"
              tabIndex={0}
              aria-label={t("FFmpeg 参数帮助")}
            >
              {help.text}
            </pre>
          </>
        )}
      </details>
    </section>
  );
}
