// @vitest-environment happy-dom

import { act, cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { API } from "../bridge";
import type { Media, SampleComparison } from "../bridge";
import { ComparisonPlayer } from "./ComparisonPlayer";

vi.mock("../bridge", () => ({
  API: {
    getMediaPreviewURL: vi.fn(),
    releaseMediaPreviewURL: vi.fn(async () => undefined),
  },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const media = (name: string): Media => ({
  source: { uri: name, displayName: name, kind: "local-file" },
  name,
  container: "mp4",
  durationSeconds: 10,
  sizeBytes: 100,
  bitrate: 80,
  streams: [],
});

function comparison(source: string, output: string): SampleComparison {
  return {
    original: media(source),
    output: media(output),
    startSeconds: 0,
    durationSeconds: 10,
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => {
    resolve = next;
  });
  return { promise, resolve };
}

describe("comparison preview capability lifecycle", () => {
  it("releases stale grants and never publishes an older source response", async () => {
    const oldOriginal = deferred<string>();
    const oldOutput = deferred<string>();
    const newOriginal = deferred<string>();
    const newOutput = deferred<string>();
    vi.mocked(API.getMediaPreviewURL)
      .mockReturnValueOnce(oldOriginal.promise)
      .mockReturnValueOnce(oldOutput.promise)
      .mockReturnValueOnce(newOriginal.promise)
      .mockReturnValueOnce(newOutput.promise);

    const props = {
      position: 0,
      onPosition: () => undefined,
      unavailable: () => undefined,
    };
    const view = render(
      <ComparisonPlayer
        {...props}
        data={comparison("old-source.mp4", "old-output.mp4")}
      />,
    );
    view.rerender(
      <ComparisonPlayer
        {...props}
        data={comparison("new-source.mp4", "new-output.mp4")}
      />,
    );

    await act(async () => {
      newOriginal.resolve("/__media/new-original");
      newOutput.resolve("/__media/new-output");
      await Promise.all([newOriginal.promise, newOutput.promise]);
    });
    const videos = view.container.querySelectorAll("video");
    await waitFor(() =>
      expect(videos[0].getAttribute("src")).toBe("/__media/new-original"),
    );
    expect(videos[1].getAttribute("src")).toBe("/__media/new-output");

    await act(async () => {
      oldOriginal.resolve("/__media/old-original");
      oldOutput.resolve("/__media/old-output");
      await Promise.all([oldOriginal.promise, oldOutput.promise]);
    });
    expect(API.releaseMediaPreviewURL).toHaveBeenCalledWith(
      "/__media/old-original",
    );
    expect(API.releaseMediaPreviewURL).toHaveBeenCalledWith(
      "/__media/old-output",
    );
    expect(videos[0].getAttribute("src")).toBe("/__media/new-original");

    view.unmount();
    expect(API.releaseMediaPreviewURL).toHaveBeenCalledWith(
      "/__media/new-original",
    );
    expect(API.releaseMediaPreviewURL).toHaveBeenCalledWith(
      "/__media/new-output",
    );
  });
});
