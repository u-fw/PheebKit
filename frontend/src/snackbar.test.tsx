import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { Snackbar } from "./components/Snackbar";
import {
  emptySnackbarState,
  snackbarReducer,
  type SnackbarNotice,
} from "./snackbar";

describe("snackbar", () => {
  it("replaces stale notices and normalizes action duration", () => {
    const first = snackbarReducer(emptySnackbarState, {
      type: "show",
      input: { message: " First action ", duration: 100 },
    });
    const second = snackbarReducer(first, {
      type: "show",
      input: { message: "Second action", duration: 60_000 },
    });
    expect(first.current?.message).toBe("First action");
    expect(first.current?.duration).toBe(4000);
    expect(second.current).toMatchObject({
      id: 2,
      message: "Second action",
      duration: 10_000,
    });
  });

  it("ignores blank messages and stale dismissal events", () => {
    const blank = snackbarReducer(emptySnackbarState, {
      type: "show",
      input: { message: "   " },
    });
    expect(blank).toBe(emptySnackbarState);
    const shown = snackbarReducer(blank, {
      type: "show",
      input: { message: "Visible" },
    });
    expect(
      snackbarReducer(shown, { type: "dismiss", id: shown.current!.id + 1 }),
    ).toBe(shown);
    expect(
      snackbarReducer(shown, { type: "dismiss", id: shown.current!.id })
        .current,
    ).toBeNull();
  });

  it("renders a polite status with the inverse-color action", () => {
    const notice: SnackbarNotice = {
      id: 1,
      message: "Queue order changed",
      action: { label: "Undo", run: vi.fn() },
      duration: 6000,
    };
    const html = renderToStaticMarkup(
      <Snackbar notice={notice} dismiss={() => {}} />,
    );
    expect(html).toContain('aria-live="polite"');
    expect(html).toContain('role="status"');
    expect(html).toContain("Queue order changed");
    expect(html).toContain(">Undo</button>");
  });
});
