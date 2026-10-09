import { useCallback, useEffect, useRef, useState } from "react";
import { API } from "../bridge";
import type { Capabilities, EnvironmentStatus, Job, Media } from "../bridge";
import { errorMessage } from "../format";

export function useStudio() {
  const [platform, setPlatform] = useState("连接中");
  const [connected, setConnected] = useState(false);
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null);
  const [capabilityError, setCapabilityError] = useState("");
  const [environment, setEnvironment] = useState<EnvironmentStatus | null>(
    null,
  );
  const [environmentBusy, setEnvironmentBusy] = useState(false);
  const [media, setMedia] = useState<Media | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const importBusy = useRef(false);
  const environmentGeneration = useRef(0);

  const updateEnvironment = useCallback(
    async (status: EnvironmentStatus, rescan = false) => {
      const generation = ++environmentGeneration.current;
      setEnvironment(status);
      setCapabilities(null);
      setCapabilityError("");
      if (!status.ready) {
        setCapabilityError(status.message);
        return;
      }
      try {
        const next = await (rescan
          ? API.refreshCapabilities()
          : API.capabilities());
        if (generation === environmentGeneration.current) setCapabilities(next);
      } catch (cause) {
        if (generation === environmentGeneration.current)
          setCapabilityError(errorMessage(cause));
      }
    },
    [],
  );

  const refreshEnvironment = useCallback(async () => {
    setEnvironmentBusy(true);
    try {
      await updateEnvironment(await API.getEnvironment(), true);
    } finally {
      setEnvironmentBusy(false);
    }
  }, [updateEnvironment]);

  const chooseFFmpeg = useCallback(async () => {
    setEnvironmentBusy(true);
    try {
      const status = await API.chooseFFmpeg();
      if (status) await updateEnvironment(status);
    } finally {
      setEnvironmentBusy(false);
    }
  }, [updateEnvironment]);

  const configureFFmpeg = useCallback(
    async (ffmpegPath: string, ffprobePath: string) => {
      setEnvironmentBusy(true);
      try {
        await updateEnvironment(
          await API.configureFFmpeg(ffmpegPath, ffprobePath),
        );
      } finally {
        setEnvironmentBusy(false);
      }
    },
    [updateEnvironment],
  );

  const importPaths = useCallback(async (paths: string[]) => {
    if (!paths.length || importBusy.current) return;
    importBusy.current = true;
    setBusy(true);
    setError("");
    try {
      setMedia(await API.probe(paths[0]));
      if (paths.length > 1)
        setError("本次 Spike 每次分析一个源文件，已导入第一个文件。");
    } catch (cause) {
      setError(`媒体分析失败：${errorMessage(cause)}`);
    } finally {
      importBusy.current = false;
      setBusy(false);
    }
  }, []);

  const chooseMedia = useCallback(async () => {
    if (importBusy.current) return;
    importBusy.current = true;
    setBusy(true);
    setError("");
    try {
      const selected = await API.chooseMedia();
      if (selected) setMedia(selected);
    } catch (cause) {
      setError(`导入失败：${errorMessage(cause)}`);
    } finally {
      importBusy.current = false;
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    let alive = true;
    let subscriptionError = "";
    const generationRef = environmentGeneration;
    const initialGeneration = environmentGeneration.current;
    const platformRequest = API.getPlatform();
    const environmentRequest = API.getEnvironment();
    void environmentRequest
      .then((status) => {
        if (alive && initialGeneration === environmentGeneration.current)
          setEnvironment(status);
      })
      .catch((cause: unknown) => {
        if (alive && initialGeneration === environmentGeneration.current)
          setCapabilityError(`FFmpeg 环境检测失败：${errorMessage(cause)}`);
      });
    void platformRequest
      .then((value) => {
        if (!alive) return;
        setPlatform(value);
        setConnected(true);
      })
      .catch((cause: unknown) => {
        if (!alive) return;
        setPlatform("浏览器预览");
        setError(
          `原生桥接不可用：${errorMessage(cause)}。请在 Wails 应用中导入和处理媒体。`,
        );
      });
    void Promise.allSettled([
      platformRequest,
      API.capabilities(),
      API.jobs(),
      environmentRequest,
    ]).then((results) => {
      if (!alive) return;
      const [platformResult, capabilityResult, jobResult, environmentResult] =
        results;
      if (
        capabilityResult.status === "fulfilled" &&
        initialGeneration === environmentGeneration.current
      )
        setCapabilities(capabilityResult.value);
      else if (
        capabilityResult.status === "rejected" &&
        initialGeneration === environmentGeneration.current
      )
        setCapabilityError(errorMessage(capabilityResult.reason));
      if (
        environmentResult.status === "fulfilled" &&
        initialGeneration === environmentGeneration.current
      )
        setEnvironment(environmentResult.value);
      else if (
        environmentResult.status === "rejected" &&
        platformResult.status === "fulfilled"
      )
        setCapabilityError(
          `FFmpeg 环境检测失败：${errorMessage(environmentResult.reason)}`,
        );
      if (jobResult.status === "fulfilled") setJobs(jobResult.value);
      else if (platformResult.status === "fulfilled")
        setError(`任务队列加载失败：${errorMessage(jobResult.reason)}`);
      if (subscriptionError && platformResult.status === "fulfilled")
        setError(subscriptionError);
    });

    const pending = new Map<string, Job>();
    let timer: ReturnType<typeof setTimeout> | undefined;
    const unsubscribe: (() => void)[] = [];
    try {
      unsubscribe.push(
        API.subscribeJobs((job) => {
          pending.set(job.id, job);
          if (timer) return;
          timer = setTimeout(() => {
            const updates = [...pending.values()];
            pending.clear();
            timer = undefined;
            if (alive)
              setJobs((current) => {
                const merged = new Map(
                  current.map((entry) => [entry.id, entry]),
                );
                updates.forEach((entry) => merged.set(entry.id, entry));
                return [...merged.values()];
              });
          }, 150);
        }),
      );
      unsubscribe.push(
        API.subscribeMedia((value) => {
          if (alive) setMedia(value);
        }),
      );
      unsubscribe.push(
        API.onDrop((paths) => {
          if (alive) void importPaths(paths);
        }),
      );
    } catch (cause) {
      subscriptionError = `原生事件桥接不可用：${errorMessage(cause)}`;
    }
    return () => {
      alive = false;
      generationRef.current++;
      if (timer) clearTimeout(timer);
      unsubscribe.forEach((off) => off());
    };
  }, [importPaths]);

  const addJob = useCallback((job: Job) => {
    setJobs((current) =>
      current.some((entry) => entry.id === job.id)
        ? current
        : [...current, job],
    );
  }, []);

  const cancelJob = useCallback(async (id: string) => {
    try {
      await API.cancel(id);
    } catch (cause) {
      setError(`取消失败：${errorMessage(cause)}`);
    }
  }, []);

  const refreshJobs = useCallback(async () => {
    try {
      setJobs(await API.jobs());
    } catch (cause) {
      setError(`刷新失败：${errorMessage(cause)}`);
    }
  }, []);

  return {
    platform,
    connected,
    capabilities,
    capabilityError,
    environment,
    environmentBusy,
    refreshEnvironment,
    chooseFFmpeg,
    configureFFmpeg,
    media,
    jobs,
    error,
    busy,
    chooseMedia,
    addJob,
    cancelJob,
    refreshJobs,
    setError,
  };
}
