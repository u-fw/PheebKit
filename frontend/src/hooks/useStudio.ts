import { useCallback, useEffect, useRef, useState } from "react";
import { API } from "../bridge";
import type {
  Capabilities,
  EnvironmentStatus,
  FFmpegSetupStatus,
  Job,
  Media,
} from "../bridge";
import { errorMessage } from "../format";
import { JobHistory } from "../jobHistory";
import { refreshEnvironmentSnapshot } from "../environmentRefresh";

export function useStudio() {
  const [platform, setPlatform] = useState("连接中");
  const [connected, setConnected] = useState(false);
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null);
  const [capabilityError, setCapabilityError] = useState("");
  const [environment, setEnvironment] = useState<EnvironmentStatus | null>(
    null,
  );
  const [environmentBusy, setEnvironmentBusy] = useState(false);
  const [ffmpegSetup, setFFmpegSetup] = useState<FFmpegSetupStatus | null>(
    null,
  );
  const [media, setMedia] = useState<Media | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const importBusy = useRef(false);
  const environmentGeneration = useRef(0);
  const jobHistory = useRef(new JobHistory());

  const updateEnvironment = useCallback(async (status: EnvironmentStatus) => {
    const generation = ++environmentGeneration.current;
    setEnvironment(status);
    setCapabilities(null);
    setCapabilityError("");
    if (!status.ready) {
      setCapabilityError(status.message);
      return;
    }
    try {
      const next = await API.capabilities();
      if (generation === environmentGeneration.current) setCapabilities(next);
    } catch (cause) {
      if (generation === environmentGeneration.current)
        setCapabilityError(errorMessage(cause));
    }
  }, []);

  const refreshEnvironment = useCallback(async () => {
    const generation = ++environmentGeneration.current;
    setEnvironmentBusy(true);
    setCapabilities(null);
    setCapabilityError("");
    try {
      await refreshEnvironmentSnapshot(
        API,
        () => generation === environmentGeneration.current,
        (snapshot) => {
          setEnvironment(snapshot.environment);
          setCapabilities(snapshot.capabilities);
        },
      );
    } catch (cause) {
      if (generation === environmentGeneration.current)
        setCapabilityError(errorMessage(cause));
      throw cause;
    } finally {
      if (generation === environmentGeneration.current)
        setEnvironmentBusy(false);
    }
  }, []);

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

  useEffect(() => {
    if (!connected || platform !== "windows") return;
    let alive = true;
    let lastStage = "";
    let timer: ReturnType<typeof setTimeout> | undefined;
    async function poll() {
      try {
        const next = await API.getFFmpegSetupStatus();
        if (!alive) return;
        setFFmpegSetup(next);
        const completed =
          next.stage === "completed" && lastStage !== "completed";
        lastStage = next.stage;
        if (completed) {
          setEnvironmentBusy(true);
          try {
            const installedEnvironment = await API.getEnvironment();
            if (alive) await updateEnvironment(installedEnvironment);
          } finally {
            if (alive) setEnvironmentBusy(false);
          }
        }
      } catch (cause) {
        if (alive) setError(`下载状态检测失败：${errorMessage(cause)}`);
      }
      if (alive) timer = setTimeout(() => void poll(), 750);
    }
    void poll();
    return () => {
      alive = false;
      if (timer) clearTimeout(timer);
    };
  }, [connected, platform, updateEnvironment]);

  const startFFmpegSetup = useCallback(async (source: string) => {
    setFFmpegSetup(await API.startFFmpegSetup(source));
  }, []);
  const cancelFFmpegSetup = useCallback(async () => {
    await API.cancelFFmpegSetup();
  }, []);

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
    const jobsRequestedAt = jobHistory.current.revision;
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
        platformResult.status === "fulfilled" &&
        initialGeneration === environmentGeneration.current
      )
        setCapabilityError(
          `FFmpeg 环境检测失败：${errorMessage(environmentResult.reason)}`,
        );
      if (jobResult.status === "fulfilled")
        setJobs(jobHistory.current.reconcile(jobResult.value, jobsRequestedAt));
      else if (platformResult.status === "fulfilled")
        setError(`任务队列加载失败：${errorMessage(jobResult.reason)}`);
      if (subscriptionError && platformResult.status === "fulfilled")
        setError(subscriptionError);
    });

    let timer: ReturnType<typeof setTimeout> | undefined;
    const unsubscribe: (() => void)[] = [];
    try {
      unsubscribe.push(
        API.subscribeJobs((job) => {
          if (!alive) return;
          jobHistory.current.update([job]);
          if (timer) return;
          timer = setTimeout(() => {
            timer = undefined;
            if (alive) setJobs(jobHistory.current.list());
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
    setJobs(jobHistory.current.update([job]));
  }, []);

  const cancelJob = useCallback(async (id: string) => {
    try {
      await API.cancel(id);
    } catch (cause) {
      setError(`取消失败：${errorMessage(cause)}`);
    }
  }, []);

  const refreshJobs = useCallback(async () => {
    const requestedAt = jobHistory.current.revision;
    try {
      setJobs(jobHistory.current.reconcile(await API.jobs(), requestedAt));
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
    ffmpegSetup,
    startFFmpegSetup,
    cancelFFmpegSetup,
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
