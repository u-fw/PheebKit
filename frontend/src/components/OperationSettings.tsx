import { t } from "../i18n";
import type { Media, Settings } from "../bridge";
import { operationLabels, operationOf, type Operation } from "../project";
import { MaterialSelect } from "./MaterialSelect";

export function OperationSettings({
  media,
  settings,
  onChange,
}: {
  media: Media;
  settings: Settings;
  onChange: (operation: Operation) => void;
}) {
  const operation = operationOf(settings);
  return (
    <section className="surface settings-card">
      <div className="section-title">
        <h2>{t("处理任务")}</h2>
      </div>
      <MaterialSelect
        label={t("你想处理什么")}
        value={operation}
        onChange={(value) => onChange(value as Operation)}
        options={[
          {
            value: "compress",
            label: operationLabels.compress,
            description: "重新编码视频，可剪辑并调整画面",
          },
          {
            value: "remux",
            label: operationLabels.remux,
            description: "保留编码，转换为兼容的媒体容器",
          },
          {
            value: "extract-audio",
            label: operationLabels["extract-audio"],
            description: "选择一条音轨，单独保存",
            disabled: !media.streams.some(
              (stream) => stream.codecType === "audio",
            ),
          },
          {
            value: "extract-subtitle",
            label: operationLabels["extract-subtitle"],
            description: "选择一条文字字幕，单独保存",
            disabled: !media.streams.some(
              (stream) => stream.codecType === "subtitle",
            ),
          },
        ]}
      />
      <p className="field-hint">
        {t(
          "切换任务会清除剪辑、多片段、字幕烧录、画面调整、音量调整和附加参数，并更新输出扩展名。源文件保持原样。",
        )}
      </p>
      {operation === "remux" && (
        <div className="field-notice">
          <p>
            {t(
              "转换封装不会重新编码。音频、视频和字幕必须与目标容器兼容；不兼容时会说明原因，请改选容器或使用压缩任务。",
            )}
          </p>
        </div>
      )}
      {operation === "extract-subtitle" && (
        <div className="field-notice">
          <p>
            {t(
              "支持文字字幕。图片字幕需要识别文字，当前不能转换；改变字幕格式可能影响样式。",
            )}
          </p>
        </div>
      )}
    </section>
  );
}
