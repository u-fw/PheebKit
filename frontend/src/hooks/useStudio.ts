import { useCallback, useEffect, useRef, useState } from "react";
import { API } from "../bridge";
import type { Capabilities, Job, Media } from "../bridge";
import { errorMessage } from "../format";

export function useStudio() {
  const [platform, setPlatform] = useState("连接中");
  const [connected, setConnected] = useState(false);
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null);
  const [capabilityError, setCapabilityError] = useState("");
  const [media, setMedia] = useState<Media | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const importBusy = useRef(false);

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
    void Promise.allSettled([
      API.getPlatform(),
      API.capabilities(),
      API.jobs(),
    ]).then((results) => {
      if (!alive) return;
      const [platformResult, capabilityResult, jobResult] = results;
      if (platformResult.status === "fulfilled") {
        setPlatform(platformResult.value);
        setConnected(true);
      } else {
        setPlatform("浏览器预览");
        setError(
          `原生桥接不可用：${errorMessage(platformResult.reason)}。请在 Wails 应用中导入和处理媒体。`,
        );
      }
      if (capabilityResult.status === "fulfilled")
        setCapabilities(capabilityResult.value);
      else setCapabilityError(errorMessage(capabilityResult.reason));
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
