import { t } from "../i18n";
import { useId, useRef, useState } from "react";
import type { Media, Settings, TimeRange } from "../bridge";
import { formatTimecode, selectedDuration, trimError } from "../trim";
import { moveSegment } from "../segments";
import { selectedTracks } from "../project";
import { TimePicker } from "./TimePicker";
import { ClipPreview } from "./ClipPreview";

export function SegmentsEditor({
  media,
  settings,
  onChange,
  visible,
  exportParts,
  canExport,
}: {
  media: Media;
  settings: Settings;
  onChange: (settings: Settings) => void;
  visible: boolean;
  exportParts?: () => void;
  canExport: boolean;
}) {
  const ranges = settings.segments ?? [];
  const helpID = useId();
  const [active, setActive] = useState(0);
  const list = useRef<HTMLOListElement>(null);
  const index = Math.min(active, Math.max(0, ranges.length - 1));
  const selected = ranges[index];
  const max = Math.floor(media.durationSeconds * 1000) / 1000;
  const error = trimError(
    settings,
    media.durationSeconds,
    media.streams.find(
      (s) =>
        s.codecType === "video" && !s.attachedPicture && !s.timedThumbnails,
    )?.frameRate,
  );
  function change(next: TimeRange[]) {
    onChange({ ...settings, trim: null, segments: next });
  }
  function boundary(key: "startSeconds" | "endSeconds", value: number) {
    change(
      ranges.map((range, i) =>
        i === index ? { ...range, [key]: value } : range,
      ),
    );
  }
  function reorder(index: number, direction: -1 | 1) {
    const next = index + direction;
    if (next < 0 || next >= ranges.length) return;
    change(moveSegment(ranges, index, direction));
    setActive(next);
    // Keep keyboard focus with the moved segment after React commits its list.
    requestAnimationFrame(() =>
      list.current
        ?.querySelectorAll<HTMLButtonElement>(".segment-select")
        .item(next)
        ?.focus(),
    );
  }
  return (
    <div className="segments-editor">
      <p className="field-hint" id={helpID}>
        {t(
          "按列表顺序拼接并重新编码；重复或重叠区间会重复保留。最多 16 段、4 条音轨。软字幕请移除或改用烧录。",
        )}
      </p>
      {selectedTracks(media, "subtitle", settings).length > 0 && (
        <button
          className="button outlined"
          onClick={() =>
            onChange({
              ...settings,
              subtitleSelection: { mode: "none", indices: [] },
            })
          }
        >
          {t("移除软字幕轨道以启用拼接")}
        </button>
      )}
      <p className="field-hint">
        {t("聚焦片段后，按 Alt + ↑ / ↓ 调整拼接顺序。")}
      </p>
      <ol className="segment-list" ref={list}>
        {ranges.map((range, i) => (
          <li key={i} className={index === i ? "selected" : ""}>
            <button
              className="button text segment-select"
              aria-pressed={index === i}
              aria-keyshortcuts="Alt+ArrowUp Alt+ArrowDown"
              onKeyDown={(event) => {
                if (
                  !event.altKey ||
                  (event.key !== "ArrowUp" && event.key !== "ArrowDown")
                )
                  return;
                event.preventDefault();
                reorder(i, event.key === "ArrowUp" ? -1 : 1);
              }}
              onClick={() => setActive(i)}
            >
              {t("片段")} {i + 1} · {t(formatTimecode(range.startSeconds))} →
              {t(" ")}
              {t(formatTimecode(range.endSeconds))}
            </button>
            <div className="action-row">
              <button
                className="button text"
                aria-label={t("片段 {{0}} 上移", [i + 1])}
                disabled={i === 0}
                onClick={() => reorder(i, -1)}
              >
                {t("上移")}
              </button>
              <button
                className="button text"
                aria-label={t("片段 {{0}} 下移", [i + 1])}
                disabled={i === ranges.length - 1}
                onClick={() => reorder(i, 1)}
              >
                {t("下移")}
              </button>
              <button
                className="button text"
                aria-label={t("删除片段 {{0}}", [i + 1])}
                disabled={ranges.length === 1}
                onClick={() => {
                  change(ranges.filter((_, j) => i !== j));
                  setActive(Math.max(0, i - 1));
                }}
              >
                {t("删除")}
              </button>
            </div>
          </li>
        ))}
      </ol>
      <button
        className="button outlined"
        disabled={ranges.length >= 16}
        onClick={() => {
          const start =
            (ranges.at(-1)?.endSeconds ?? 0) < max
              ? (ranges.at(-1)?.endSeconds ?? 0)
              : 0;
          change([
            ...ranges,
            { startSeconds: start, endSeconds: Math.min(max, start + 10) },
          ]);
          setActive(ranges.length);
        }}
      >
        {t("添加片段")}
      </button>
      {selected && (
        <>
          <h3>
            {t("编辑片段")} {index + 1}
          </h3>
          <div className="field-grid trim-fields">
            <TimePicker
              label={t("片段开始时间")}
              value={selected.startSeconds}
              max={max}
              onChange={(v) => boundary("startSeconds", v)}
              invalid={Boolean(error)}
              describedBy={helpID}
            />
            <TimePicker
              label={t("片段结束时间")}
              value={selected.endSeconds}
              max={max}
              onChange={(v) => boundary("endSeconds", v)}
              invalid={Boolean(error)}
              describedBy={helpID}
            />
          </div>
          <ClipPreview
            key={media.source.uri}
            media={media}
            start={selected.startSeconds}
            end={selected.endSeconds}
            visible={visible}
            onBoundary={boundary}
          />
        </>
      )}
      {error ? (
        <p className="inline-error" role="alert">
          {t(error)}
        </p>
      ) : (
        <p className="field-hint">
          {t("合计保留")}
          {t(" ")}
          {t(
            formatTimecode(selectedDuration(settings, media.durationSeconds)),
          )}{" "}
          {t("。右侧开始按钮将输出一个拼接文件。")}
        </p>
      )}
      <button
        className="button outlined"
        disabled={!canExport || !exportParts || Boolean(error)}
        onClick={exportParts}
      >
        {t("分别导出")} {ranges.length} {t("个片段")}
      </button>
      <p className="field-hint">
        {t(
          "分别导出时使用 -part-01 等文件名；全部预检通过后加入队列。目标体积按各段时长分摊。",
        )}
      </p>
    </div>
  );
}
