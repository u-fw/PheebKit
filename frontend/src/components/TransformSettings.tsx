import { t } from "../i18n";
import type { Media, Settings } from "../bridge";
import { operationOf, selectedTracks, videoDisplaySize } from "../project";
import type { UpdateSettings } from "./EncodingSettings";
import { MaterialSelect } from "./MaterialSelect";

const emptyTransform = {
  rotation: 0,
  flipHorizontal: false,
  flipVertical: false,
  crop: null,
};
const cropLabels = {
  top: "上边距",
  bottom: "下边距",
  left: "左边距",
  right: "右边距",
};
export function TransformSettings({
  media,
  settings,
  update,
}: {
  media: Media;
  settings: Settings;
  update: UpdateSettings;
}) {
  const compress = operationOf(settings) === "compress";
  const transform = settings.transform ?? emptyTransform;
  const video = media.streams.find(
    (stream) =>
      stream.codecType === "video" &&
      !stream.attachedPicture &&
      !stream.timedThumbnails,
  );
  const hasAudio = selectedTracks(media, "audio", settings).length > 0;
  return (
    <details className="surface advanced-card transform-settings">
      <summary>
        <span>{t(compress ? "画面与声音调整" : "声音调整")}</span>
        <span>{t("可选")}</span>
      </summary>
      <div className="advanced-content">
        {compress && (
          <>
            <MaterialSelect
              label={t("旋转画面")}
              value={String(transform.rotation)}
              options={[
                { value: "0", label: "保持方向" },
                { value: "90", label: "顺时针 90°" },
                { value: "180", label: "旋转 180°" },
                { value: "270", label: "逆时针 90°" },
              ]}
              onChange={(value) =>
                update("transform", { ...transform, rotation: Number(value) })
              }
            />
            <div className="track-list">
              {(["flipHorizontal", "flipVertical"] as const).map((key) => (
                <label className="track-choice" key={key}>
                  <input
                    type="checkbox"
                    checked={transform[key]}
                    onChange={(event) =>
                      update("transform", {
                        ...transform,
                        [key]: event.target.checked,
                      })
                    }
                  />
                  <span>
                    {t(key === "flipHorizontal" ? "左右镜像" : "上下翻转")}
                  </span>
                </label>
              ))}
            </div>
            <label className="track-choice">
              <input
                type="checkbox"
                checked={Boolean(transform.crop)}
                onChange={(event) =>
                  update("transform", {
                    ...transform,
                    crop: event.target.checked
                      ? { top: 0, bottom: 0, left: 0, right: 0 }
                      : null,
                  })
                }
              />
              <span>{t("裁剪画面边缘")}</span>
            </label>
            {transform.crop && (
              <>
                <div className="field-grid">
                  {(Object.keys(cropLabels) as (keyof typeof cropLabels)[]).map(
                    (key) => (
                      <label className="field-label" key={key}>
                        {t(cropLabels[key])} (px)
                        <input
                          type="number"
                          min="0"
                          max="16384"
                          step="2"
                          value={transform.crop?.[key] ?? 0}
                          onChange={(event) =>
                            update("transform", {
                              ...transform,
                              crop: {
                                ...(transform.crop ?? {
                                  top: 0,
                                  bottom: 0,
                                  left: 0,
                                  right: 0,
                                }),
                                [key]: Number(event.target.value),
                              },
                            })
                          }
                        />
                      </label>
                    ),
                  )}
                </div>
                <p className="field-hint">
                  {t("按原片显示方向裁剪，再旋转和缩放。边距使用偶数像素。")}{" "}
                  {t(
                    video
                      ? `源画面 ${videoDisplaySize(video).width} × ${videoDisplaySize(video).height}。`
                      : "",
                  )}{" "}
                  {t("预览仍展示原片，调整将在输出中生效。")}
                </p>
              </>
            )}
          </>
        )}
        {hasAudio && (
          <label className="field-label">
            {t("音量调整 (dB)")}
            <input
              type="number"
              min="-60"
              max="20"
              step="1"
              value={settings.volumeDB ?? 0}
              onChange={(event) =>
                update("volumeDB", Number(event.target.value))
              }
            />
            <span className="field-hint">
              {t("0 保持原音量；负值减小，正值增大。需要重新编码音频。")}
            </span>
          </label>
        )}
        {hasAudio &&
          settings.audio === "copy" &&
          (settings.volumeDB ?? 0) !== 0 && (
            <div className="field-notice error-notice" role="alert">
              <div>
                <p>
                  {t("保留原音频不能调整音量。改为 AAC，或把音量调整设回 0。")}
                </p>
                <button
                  type="button"
                  className="button outlined"
                  onClick={() => update("audio", "aac")}
                >
                  {t("切换为 AAC")}
                </button>
              </div>
            </div>
          )}
        {!compress && !hasAudio && (
          <p className="field-hint">{t("先选择一条音轨。")}</p>
        )}
      </div>
    </details>
  );
}
