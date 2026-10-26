import type { Capabilities, EnvironmentStatus } from "./bridge";

export type ReadinessTone = "checking" | "ready" | "fallback" | "attention";

export type ReadinessSignal =
  | {
      kind: "ffmpeg";
      tone: "checking" | "ready" | "attention";
      label: string;
      detail: string;
      detailValues?: string[];
    }
  | {
      kind: "acceleration";
      tone: Exclude<ReadinessTone, "ready"> | "ready";
      label: string;
      detail: string;
      detailValues?: string[];
    }
  | {
      kind: "storage";
      tone: "ready" | "attention";
      label: string;
      detail: string;
      detailValues?: string[];
    };

export interface ProcessingReadiness {
  canChooseMedia: boolean;
  ffmpeg: ReadinessSignal;
  acceleration: ReadinessSignal;
  storage: ReadinessSignal;
}

function compactFFmpegVersion(version: string): string {
  const token = version.trim().match(/^ffmpeg version\s+(\S+)/i)?.[1];
  return (token || version.trim()).slice(0, 64);
}

export function summarizeProcessingReadiness({
  connected,
  environment,
  capabilities,
  capabilityError = "",
}: {
  connected: boolean;
  environment: EnvironmentStatus | null;
  capabilities: Capabilities | null;
  capabilityError?: string;
}): ProcessingReadiness {
  const ffmpeg: ReadinessSignal = !environment
    ? {
        kind: "ffmpeg",
        tone: "checking",
        label: "正在检测 FFmpeg",
        detail: "正在验证 ffmpeg 与 ffprobe。",
      }
    : environment.ready
      ? {
          kind: "ffmpeg",
          tone: "ready",
          label: "FFmpeg 已就绪",
          detail:
            compactFFmpegVersion(environment.ffmpeg.version) ||
            "可执行文件已通过验证。",
        }
      : {
          kind: "ffmpeg",
          tone: "attention",
          label: "需要 FFmpeg",
          detail: "设置后可导入并处理媒体。",
        };

  const gpu = capabilities?.videoEncoders.find(
    (encoder) => encoder.kind === "gpu" && encoder.available,
  );
  const acceleration: ReadinessSignal =
    !environment || !environment.ready
      ? {
          kind: "acceleration",
          tone: "checking",
          label: "等待环境检查",
          detail: "FFmpeg 就绪后会检测可用编码器。",
        }
      : capabilityError
        ? {
            kind: "acceleration",
            tone: "attention",
            label: "硬件检测失败",
            detail: "仍可使用自动编码设置；可在设置中重新检测。",
          }
        : !capabilities
          ? {
              kind: "acceleration",
              tone: "checking",
              label: "正在检测硬件",
              detail: "正在验证 CPU 与 GPU 编码路径。",
            }
          : gpu
            ? {
                kind: "acceleration",
                tone: "ready",
                label: "GPU 加速可用",
                detail: "{{0}} · 实际编码测试通过",
                detailValues: [gpu.label],
              }
            : {
                kind: "acceleration",
                tone: "fallback",
                label: "CPU 编码可用",
                detail: "未检测到已验证的 GPU 编码器。",
              };

  const storage: ReadinessSignal = connected
    ? {
        kind: "storage",
        tone: "ready",
        label: "本机处理已启用",
        detail: "媒体与项目仅保存在本机。",
      }
    : {
        kind: "storage",
        tone: "attention",
        label: "当前为界面预览",
        detail: "媒体操作需要桌面应用。",
      };

  return {
    canChooseMedia: connected && environment?.ready === true,
    ffmpeg,
    acceleration,
    storage,
  };
}
