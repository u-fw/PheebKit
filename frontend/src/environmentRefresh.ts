import type { Capabilities, EnvironmentStatus } from "./bridge";

interface EnvironmentAPI {
  refreshCapabilities: () => Promise<Capabilities>;
  getEnvironment: () => Promise<EnvironmentStatus>;
}

interface EnvironmentSnapshot {
  environment: EnvironmentStatus;
  capabilities: Capabilities | null;
}

export async function refreshEnvironmentSnapshot(
  api: EnvironmentAPI,
  isCurrent: () => boolean,
  publish: (snapshot: EnvironmentSnapshot) => void,
): Promise<void> {
  // Refresh also reloads shared configuration, so the prior ready status cannot
  // decide whether to scan. Read actual runner paths after success or failure.
  const [scan] = await Promise.allSettled([api.refreshCapabilities()]);
  const environment = await api.getEnvironment();
  if (isCurrent())
    publish({
      environment,
      capabilities:
        environment.ready && scan.status === "fulfilled" ? scan.value : null,
    });
  if (scan.status === "rejected") throw scan.reason;
  if (!environment.ready)
    throw new Error(environment.message || "FFmpeg 环境尚未就绪。请补全路径。");
}
