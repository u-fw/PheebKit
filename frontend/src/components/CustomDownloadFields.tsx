import { MaterialSelect } from "./MaterialSelect";

export function customDownloadError(
  link: string,
  sha256: string,
  otherBuild: boolean,
  trusted: boolean,
): string {
  if (!link.trim()) return "请填写 ZIP 文件的完整下载链接。";
  try {
    const url = new URL(link.trim());
    if (
      url.protocol !== "https:" ||
      url.username ||
      url.password ||
      url.hash ||
      /\s/.test(link.trim())
    )
      return "请使用不含账号密码或 URL 片段的完整 HTTPS 链接。";
    if (/\.(7z|rar|tar|gz|xz|bz2)$/i.test(url.pathname))
      return "目前仅支持 ZIP，请选择 Windows x64 ZIP 包。";
  } catch {
    return "请填写有效的 HTTPS 文件下载链接。";
  }
  if (otherBuild && !/^[a-f\d]{64}$/i.test(sha256.trim()))
    return "其他 ZIP 构建需要提供完整的 64 位 SHA-256。";
  if (otherBuild && !trusted)
    return "请确认信任此来源，并允许运行包中的 FFmpeg 和 ffprobe。";
  return "";
}

export function CustomDownloadFields({
  link,
  sha256,
  otherBuild,
  trusted,
  disabled,
  onLink,
  onSHA256,
  onOtherBuild,
  onTrusted,
}: {
  link: string;
  sha256: string;
  otherBuild: boolean;
  trusted: boolean;
  disabled: boolean;
  onLink: (value: string) => void;
  onSHA256: (value: string) => void;
  onOtherBuild: (value: boolean) => void;
  onTrusted: (value: boolean) => void;
}) {
  return (
    <div className="custom-download-fields">
      <MaterialSelect
        label="自定义包类型"
        value={otherBuild ? "other" : "mirror"}
        disabled={disabled}
        onChange={(value) => onOtherBuild(value === "other")}
        options={[
          { value: "mirror", label: "内置 Gyan 9.0.2 完整版的 ZIP 镜像" },
          { value: "other", label: "其他 Windows x64 ZIP 构建" },
        ]}
      />
      <label className="field-label">
        ZIP 下载链接
        <input
          type="url"
          value={link}
          onChange={(event) => onLink(event.target.value)}
          disabled={disabled}
          placeholder="https://a.com/ffmpeg/version.zip"
          spellCheck={false}
          autoComplete="off"
          aria-describedby="custom-download-help"
        />
      </label>
      <p id="custom-download-help" className="field-hint">
        填写文件直链。ZIP 内需有同目录的 ffmpeg.exe 和 ffprobe.exe；顶层文件夹和
        bin 子目录会自动识别。7z、RAR、源码包和下载页面暂不支持。
      </p>
      {otherBuild ? (
        <>
          <label className="field-label">
            SHA-256
            <input
              value={sha256}
              onChange={(event) => onSHA256(event.target.value)}
              disabled={disabled}
              placeholder="发布者提供的 64 位十六进制校验值"
              spellCheck={false}
              autoComplete="off"
              maxLength={64}
              aria-describedby="custom-hash-help"
            />
          </label>
          <p id="custom-hash-help" className="field-hint">
            从发布者处获取校验值。SHA-256
            检查文件一致性，不证明来源可信；验证时会运行包中的程序。
          </p>
          <label className="custom-download-trust">
            <input
              type="checkbox"
              checked={trusted}
              disabled={disabled}
              onChange={(event) => onTrusted(event.target.checked)}
            />
            <span>我信任此来源，允许运行下载包中的 FFmpeg 和 ffprobe。</span>
          </label>
        </>
      ) : (
        <p className="field-hint">
          无需填写校验值，但文件必须与内置 Gyan 9.0.2 完整版 ZIP
          完全一致。其他版本或构建请选择“其他 Windows x64 ZIP 构建”。
        </p>
      )}
    </div>
  );
}
