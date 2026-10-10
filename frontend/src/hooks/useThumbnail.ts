import { useEffect, useState } from "react";
import { API } from "../bridge";
import type { Media, Thumbnail } from "../bridge";

const thumbnails = new Map<string, Promise<Thumbnail>>();
const maxCachedThumbnails = 24;

function loadThumbnail(media: Media): Promise<Thumbnail> {
  const key = JSON.stringify([
    media.source.uri,
    media.sourceFingerprint,
    media.sizeBytes,
    media.durationSeconds,
  ]);
  const cached = thumbnails.get(key);
  if (cached) return cached;
  const request = API.getThumbnail(media.source.uri).catch((cause: unknown) => {
    thumbnails.delete(key);
    throw cause;
  });
  thumbnails.set(key, request);
  if (thumbnails.size > maxCachedThumbnails) {
    const oldest = thumbnails.keys().next().value;
    if (oldest !== undefined) thumbnails.delete(oldest);
  }
  return request;
}

export function useThumbnail(media: Media) {
  const key = JSON.stringify([
    media.source.uri,
    media.sourceFingerprint,
    media.sizeBytes,
    media.durationSeconds,
  ]);
  const [state, setState] = useState<{
    key: string;
    thumbnail: Thumbnail | null;
    failed: boolean;
  }>({ key: "", thumbnail: null, failed: false });
  useEffect(() => {
    let alive = true;
    void loadThumbnail(media)
      .then((thumbnail) => {
        if (alive) setState({ key, thumbnail, failed: false });
      })
      .catch(() => {
        if (alive) setState({ key, thumbnail: null, failed: true });
      });
    return () => {
      alive = false;
    };
  }, [media, key]);
  return {
    thumbnail: state.key === key ? state.thumbnail : null,
    loading: state.key !== key,
    failed: state.key === key && state.failed,
  };
}
