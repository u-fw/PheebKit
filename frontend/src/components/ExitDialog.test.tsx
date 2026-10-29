// @vitest-environment happy-dom

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { API } from "../bridge";
import { setLanguage } from "../i18n";
import { ExitDialog } from "./ExitDialog";

vi.mock("../bridge", () => ({ API: { confirmQuit: vi.fn() } }));
beforeEach(() => {
  setLanguage("zh-CN");
  vi.mocked(API.confirmQuit).mockReset().mockResolvedValue(undefined);
});
afterEach(cleanup);

describe("exit after a failed final history write", () => {
  const stopped = {
    jobs: 0,
    downloading: false,
    shutdownStarted: true,
    historySaveFailed: true,
  };

  it("keeps the stopped app visible and retries saving before exit", async () => {
    vi.mocked(API.confirmQuit).mockRejectedValueOnce(new Error("disk full"));
    const dismiss = vi.fn();
    render(<ExitDialog request={stopped} dismiss={dismiss} />);
    const back = screen.getByRole("button", { name: "返回工作台" });
    expect((back as HTMLButtonElement).disabled).toBe(true);
    fireEvent(
      screen.getByRole("dialog"),
      new Event("cancel", { cancelable: true }),
    );
    expect(dismiss).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "重试保存并退出" }));
    await waitFor(() =>
      expect(screen.getByRole("alert").textContent).toContain("disk full"),
    );
    expect(API.confirmQuit).toHaveBeenCalledWith(false);
    expect((back as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "重试保存并退出" }));
    await waitFor(() => expect(API.confirmQuit).toHaveBeenCalledTimes(2));
    expect(API.confirmQuit).toHaveBeenLastCalledWith(false);
  });

  it("permits history loss only through its explicitly named action", async () => {
    render(<ExitDialog request={stopped} dismiss={vi.fn()} />);
    expect(API.confirmQuit).not.toHaveBeenCalled();
    fireEvent.click(
      screen.getByRole("button", { name: "不保存任务历史并退出" }),
    );
    await waitFor(() =>
      expect(API.confirmQuit).toHaveBeenCalledExactlyOnceWith(true),
    );
  });

  it("returns to editing when saving a project was cancelled before shutdown", async () => {
    const dismiss = vi.fn();
    const save = vi.fn().mockResolvedValue(false);
    render(
      <ExitDialog
        request={{ jobs: 0, downloading: false, unsaved: true }}
        dismiss={dismiss}
        save={save}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "保存项目并退出" }));
    await waitFor(() =>
      expect(
        (
          screen.getByRole("button", {
            name: "返回工作台",
          }) as HTMLButtonElement
        ).disabled,
      ).toBe(false),
    );
    expect(save).toHaveBeenCalledOnce();
    expect(API.confirmQuit).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "返回工作台" }));
    expect(dismiss).toHaveBeenCalledOnce();
  });
});
