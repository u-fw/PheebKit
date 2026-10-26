export type SnackbarAction = {
  label: string;
  run: () => Promise<unknown> | unknown;
};

export type SnackbarInput = {
  message: string;
  action?: SnackbarAction;
  duration?: number;
};

export type SnackbarNotice = SnackbarInput & {
  id: number;
};

export type SnackbarState = {
  current: SnackbarNotice | null;
  sequence: number;
};

export const emptySnackbarState: SnackbarState = {
  current: null,
  sequence: 0,
};

export type SnackbarEvent =
  { type: "show"; input: SnackbarInput } | { type: "dismiss"; id?: number };

export function snackbarReducer(
  state: SnackbarState,
  event: SnackbarEvent,
): SnackbarState {
  if (event.type === "show") {
    const message = event.input.message.trim();
    if (!message) return state;
    const sequence = state.sequence + 1;
    return {
      sequence,
      current: {
        ...event.input,
        message,
        duration: Math.max(4000, Math.min(10000, event.input.duration ?? 6000)),
        id: sequence,
      },
    };
  }
  if (event.id !== undefined && state.current?.id !== event.id) return state;
  return { ...state, current: null };
}
