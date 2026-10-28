import type { FFmpegSetupStatus } from "./bridge";

export function latestFFmpegSetup(
  current: FFmpegSetupStatus | null,
  incoming: FFmpegSetupStatus,
): FFmpegSetupStatus {
  if (!current || incoming.sequence >= current.sequence) return incoming;
  return current;
}
