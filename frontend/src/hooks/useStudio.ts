import { useCallback, useEffect, useRef, useState } from "react";
import { API } from "../bridge";
import type {
  Capabilities,
  EnvironmentStatus,
  FFmpegSetupStatus,
  Job,
  Media,
  Settings,
  QueueStatus,
} from "../bridge";
import { errorMessage } from "../format";
import { JobHistory } from "../jobHistory";
import { refreshEnvironmentSnapshot } from "../environmentRefresh";
import { latestFFmpegSetup } from "../ffmpegSetup";
import { latestQueueStatus } from "../queueSchedule";

export function useStudio(protect?: (action: () => Promise<void>) => void) {
  const mediaRevision = useRef(0);
  const [outputDirectory, setOutputDirectory] = useState<string>();
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
  const ffmpegSetupRef = useRef<FFmpegSetupStatus | null>(null);
  const [media, setMedia] = useState<Media | null>(null);
  const [mediaList, setMediaList] = useState<Media[]>([]);
  const mediaListRef = useRef<Media[]>([]);
  const [importProgress, setImportProgress] = useState({
    completed: 0,
    total: 0,
  });
  const stopImport = useRef(false);
  const [queueStatus, setQueueStatus] = useState<QueueStatus | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (!connected) return;
    mediaRevision.current = Math.max(Date.now(), mediaRevision.current + 1);
    void API.setActiveMedia(
      mediaList.map((item) => item.source.uri),
      mediaRevision.current,
    ).catch((cause: unknown) =>
      setError(`媒体集合保护失败：${errorMessage(cause)}`),
    );
  }, [connected, mediaList]);
  const [savedSettings, setSavedSettings] = useState<Settings | undefined>();
  const [reuse, setReuse] = useState<{
    settings: Settings;
    nonce: string;
    output?: string;
    needsSave?: boolean;
  } | null>(null);
  const importBusy = useRef(false);
  const environmentGeneration = useRef(0);
  const jobHistory = useRef(new JobHistory());
  const savedSettingsRevision = useRef(0);
  const updateSavedSettings = useCallback((settings: Settings) => {
    savedSettingsRevision.current++;
    setSavedSettings(settings);
  }, []);

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

  const acceptFFmpegSetup = useCallback(
    (next: FFmpegSetupStatus) => {
      const current = ffmpegSetupRef.current;
      const ordered = latestFFmpegSetup(current, next);
      if (current === ordered) return;
      const completed =
        ordered.stage === "completed" && current?.stage !== "completed";
      ffmpegSetupRef.current = ordered;
      setFFmpegSetup(ordered);
      if (!completed) return;
      setEnvironmentBusy(true);
      void API.getEnvironment()
        .then(updateEnvironment)
        .catch((cause: unknown) =>
          setError(`下载完成后环境刷新失败：${errorMessage(cause)}`),
        )
        .finally(() => setEnvironmentBusy(false));
    },
    [updateEnvironment],
  );

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
    const apply = (status: FFmpegSetupStatus) => {
      if (alive) acceptFFmpegSetup(status);
    };
    const unsubscribe = API.subscribeFFmpegSetup(apply);
    void API.getFFmpegSetupStatus().then(apply, (cause: unknown) => {
      if (alive) setError(`下载状态读取失败：${errorMessage(cause)}`);
    });
    return () => {
      alive = false;
      unsubscribe();
    };
  }, [acceptFFmpegSetup, connected, platform]);

  const startFFmpegSetup = useCallback(
    async (source: string) => {
      acceptFFmpegSetup(await API.startFFmpegSetup(source));
    },
    [acceptFFmpegSetup],
  );
  const startCustomFFmpegSetup = useCallback(
    async (link: string, sha256: string, trustBuild: boolean) => {
      acceptFFmpegSetup(
        await API.startCustomFFmpegSetup(link, sha256, trustBuild),
      );
    },
    [acceptFFmpegSetup],
  );
  const cancelFFmpegSetup = useCallback(async () => {
    await API.cancelFFmpegSetup();
  }, []);

  const processImport = useCallback(async (paths: string[], append = false) => {
    const unique = [
      ...new Map(paths.map((path) => [path.toLowerCase(), path])).values(),
    ];
    const existing = append ? mediaListRef.current : [];
    const newPaths = unique.filter(
      (path) =>
        !existing.some(
          (item) => item.source.uri.toLowerCase() === path.toLowerCase(),
        ),
    );
    if (existing.length + newPaths.length > 32)
      throw new Error("每批最多导入 32 个文件，请分批选择。");
    stopImport.current = false;
    setImportProgress({ completed: 0, total: unique.length });
    const imported: Media[] = [];
    const failed: string[] = [];
    for (let i = 0; i < unique.length; i++) {
      if (stopImport.current) break;
      try {
        imported.push(await API.probe(unique[i]));
      } catch (cause) {
        failed.push(
          `${unique[i].split(/[\\/]/).pop()}：${errorMessage(cause)}`,
        );
      }
      setImportProgress({ completed: i + 1, total: unique.length });
    }
    if (imported.length) {
      setNotice("");
      const all = append ? [...existing, ...imported] : imported;
      const collection = [
        ...new Map(
          all.map((item) => [item.source.uri.toLowerCase(), item]),
        ).values(),
      ];
      mediaListRef.current = collection;
      setMediaList(collection);
      if (!append) {
        setMedia(imported[0]);
        setReuse(null);
      }
    }
    if (failed.length) setError(`以下文件未导入：${failed.join("；")}`);
    if (stopImport.current)
      setError(`已停止后续分析，保留 ${imported.length} 个已成功导入的文件。`);
  }, []);

  const importPaths = useCallback(
    async (paths: string[]) => {
      if (!paths.length || importBusy.current) return;
      importBusy.current = true;
      setBusy(true);
      setError("");
      try {
        await processImport(paths);
      } catch (cause) {
        setError(`媒体分析失败：${errorMessage(cause)}`);
      } finally {
        importBusy.current = false;
        setBusy(false);
      }
    },
    [processImport],
  );

  const chooseMedia = useCallback(
    async (append = false) => {
      if (importBusy.current) return;
      importBusy.current = true;
      setBusy(true);
      setImportProgress({ completed: 0, total: 0 });
      setError("");
      try {
        const paths = await API.chooseMediaPaths();
        if (paths.length) {
          const workspace = await API.getWorkspace();
          setOutputDirectory(
            workspace.preferences.lastOutputDirectory || undefined,
          );
          await processImport(paths, append);
        }
      } catch (cause) {
        setError(`导入失败：${errorMessage(cause)}`);
      } finally {
        importBusy.current = false;
        setBusy(false);
      }
    },
    [processImport],
  );

  const selectMedia = useCallback((value: Media) => {
    if (importBusy.current) return;
    setNotice("");
    setMedia(value);
    setReuse(null);
  }, []);

  const loadProject = useCallback(async (loader = API.loadProject) => {
    if (importBusy.current) return false;
    importBusy.current = true;
    setBusy(true);
    setError("");
    setImportProgress({ completed: 0, total: 0 });
    try {
      const value = await loader();
      if (!value) return false;
      setNotice(value.notice || "");
      setMedia(value.media);
      setMediaList([value.media]);
      mediaListRef.current = [value.media];
      setReuse({
        settings: value.project.settings,
        output: value.project.output.uri || undefined,
        needsSave: value.needsSave,
        nonce: crypto.randomUUID(),
      });
      return true;
    } catch (cause) {
      setError(`打开项目失败：${errorMessage(cause)}`);
      return false;
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
    const workspaceRequest = API.getWorkspace();
    void API.queueStatus().then(
      (value) => {
        if (alive)
          setQueueStatus((current) => latestQueueStatus(current, value));
      },
      (cause) => {
        if (alive) setError(`队列状态读取失败：${errorMessage(cause)}`);
      },
    );
    const initialSavedRevision = savedSettingsRevision.current;
    // Defaults need not wait for the comparatively slow GPU scan. Later user
    // saves take precedence over an older startup reply; errors are handled below.
    void Promise.allSettled([workspaceRequest]).then(([result]) => {
      if (
        alive &&
        result.status === "fulfilled" &&
        initialSavedRevision === savedSettingsRevision.current
      ) {
        setSavedSettings(result.value.preferences.lastSettings ?? undefined);
        setOutputDirectory(
          result.value.preferences.lastOutputDirectory || undefined,
        );
      }
    });
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
      workspaceRequest,
    ]).then((results) => {
      if (!alive) return;
      const [
        platformResult,
        capabilityResult,
        jobResult,
        environmentResult,
        workspaceResult,
      ] = results;
      if (workspaceResult.status === "fulfilled") {
        if (workspaceResult.value.error) setError(workspaceResult.value.error);
      } else if (platformResult.status === "fulfilled")
        setError(`本地设置读取失败：${errorMessage(workspaceResult.reason)}`);
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
    let sequenceReconcileTimer: ReturnType<typeof setTimeout> | undefined;
    const reconcileSequenceGap = () => {
      if (sequenceReconcileTimer) return;
      sequenceReconcileTimer = setTimeout(async () => {
        sequenceReconcileTimer = undefined;
        const requestedAt = jobHistory.current.revision;
        try {
          const [snapshot, status] = await Promise.all([
            API.jobs(),
            API.queueStatus(),
          ]);
          if (!alive) return;
          setJobs(jobHistory.current.reconcile(snapshot, requestedAt));
          setQueueStatus((current) => latestQueueStatus(current, status));
        } catch (cause) {
          if (alive) setError(`任务事件缺口同步失败：${errorMessage(cause)}`);
        }
      }, 0);
    };
    const unsubscribe: (() => void)[] = [];
    try {
      unsubscribe.push(
        API.subscribeJobs((job) => {
          if (!alive) return;
          const update = jobHistory.current.updateWithSequence([job]);
          if (update.sequenceGap) reconcileSequenceGap();
          if (timer) return;
          timer = setTimeout(() => {
            timer = undefined;
            if (alive) setJobs(jobHistory.current.list());
          }, 150);
        }),
      );
      unsubscribe.push(
        API.subscribeQueue((value) => {
          if (alive)
            setQueueStatus((current) => latestQueueStatus(current, value));
        }),
      );
      unsubscribe.push(
        API.onDrop((paths) => {
          if (alive) {
            if (protect) protect(() => importPaths(paths));
            else void importPaths(paths);
          }
        }),
      );
    } catch (cause) {
      subscriptionError = `原生事件桥接不可用：${errorMessage(cause)}`;
    }
    return () => {
      alive = false;
      generationRef.current++;
      if (timer) clearTimeout(timer);
      if (sequenceReconcileTimer) clearTimeout(sequenceReconcileTimer);
      unsubscribe.forEach((off) => off());
    };
  }, [importPaths, protect]);

  const addJob = useCallback((job: Job) => {
    setJobs(jobHistory.current.update([job]));
  }, []);
  const addJobs = useCallback((values: Job[]) => {
    setJobs(jobHistory.current.update(values));
  }, []);

  const reuseJob = useCallback(async (id: string) => {
    if (importBusy.current) return false;
    importBusy.current = true;
    setBusy(true);
    setImportProgress({ completed: 0, total: 0 });
    setError("");
    try {
      const next = await API.reuseJob(id);
      setNotice(
        "已恢复任务设置。请检查源文件、轨道和输出位置，再点击开始处理；这会创建新任务。",
      );
      setMedia(next.media);
      setMediaList([next.media]);
      mediaListRef.current = [next.media];
      setReuse({ settings: next.settings, nonce: crypto.randomUUID() });
      return true;
    } catch (cause) {
      setError(`无法复用任务：${errorMessage(cause)}`);
      return false;
    } finally {
      importBusy.current = false;
      setBusy(false);
    }
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
      const [nextJobs, nextStatus] = await Promise.all([
        API.jobs(),
        API.queueStatus(),
      ]);
      setJobs(jobHistory.current.reconcile(nextJobs, requestedAt));
      setQueueStatus((current) => latestQueueStatus(current, nextStatus));
    } catch (cause) {
      setError(`刷新失败：${errorMessage(cause)}`);
    }
  }, []);

  const pauseQueue = useCallback(async (paused: boolean) => {
    const next = await API.setQueuePaused(paused);
    setQueueStatus((current) => latestQueueStatus(current, next));
  }, []);
  const moveQueued = useCallback(async (id: string, direction: number) => {
    const next = await API.moveQueued(id, direction);
    setQueueStatus((current) => latestQueueStatus(current, next));
  }, []);

  return {
    platform,
    savedSettings,
    outputDirectory,
    setOutputDirectory,
    setSavedSettings: updateSavedSettings,
    reuse,
    reuseJob,
    connected,
    capabilities,
    capabilityError,
    environment,
    environmentBusy,
    ffmpegSetup,
    startFFmpegSetup,
    startCustomFFmpegSetup,
    cancelFFmpegSetup,
    refreshEnvironment,
    chooseFFmpeg,
    configureFFmpeg,
    media,
    mediaList,
    selectMedia,
    importProgress,
    stopImport: () => {
      stopImport.current = true;
    },
    loadProject,
    queueStatus,
    pauseQueue,
    moveQueued,
    addJobs,
    jobs,
    error,
    notice,
    setNotice,
    busy,
    chooseMedia,
    addJob,
    cancelJob,
    refreshJobs,
    setError,
  };
}
