import { useEffect, useState } from "react";
import { API } from "../bridge";
import type { ExecutionPlan, Media, Settings } from "../bridge";
import { errorMessage } from "../format";
import { makeProject, validateSettings } from "../project";

type PreviewState = {
  key: string;
  plan: ExecutionPlan | null;
  error: string;
  loading: boolean;
};

export function useExecutionPlan(
  media: Media | null,
  settings: Settings,
  output: string,
  connected: boolean,
  projectID: string,
) {
  const [preview, setPreview] = useState<PreviewState>({
    key: "",
    plan: null,
    error: "",
    loading: false,
  });
  const key = JSON.stringify([
    media?.source.uri,
    settings,
    output,
    connected,
    projectID,
  ]);
  const valid = Boolean(
    media &&
    connected &&
    !validateSettings(settings, output, media.source.uri).length,
  );

  useEffect(() => {
    if (!media || !valid) return;
    let alive = true;
    const timer = setTimeout(() => {
      setPreview({ key, plan: null, error: "", loading: true });
      void API.preview(makeProject(media, settings, output, projectID))
        .then((plan) => {
          if (alive) setPreview({ key, plan, error: "", loading: false });
        })
        .catch((cause: unknown) => {
          if (alive)
            setPreview({
              key,
              plan: null,
              error: errorMessage(cause),
              loading: false,
            });
        });
    }, 250);
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [media, settings, output, connected, projectID, key, valid]);

  return {
    plan: valid && preview.key === key ? preview.plan : null,
    error: valid && preview.key === key ? preview.error : "",
    loading: valid && (preview.key !== key || preview.loading),
  };
}
