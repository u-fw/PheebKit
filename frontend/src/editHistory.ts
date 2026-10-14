import type { Settings } from "./bridge";

export type EditorValue = { settings: Settings; output: string };
export type EditHistory = {
  present: EditorValue;
  past: EditorValue[];
  future: EditorValue[];
  group: string | null;
  changedAt: number;
};
export type EditAction = {
  type: "edit" | "undo" | "redo";
  value?: (current: EditorValue) => EditorValue;
  group?: string | null;
  at: number;
  limit: number;
};
export function newEditHistory(value: EditorValue): EditHistory {
  return {
    present: structuredClone(value),
    past: [],
    future: [],
    group: null,
    changedAt: 0,
  };
}
export function reduceEditHistory(
  state: EditHistory,
  action: EditAction,
): EditHistory {
  const limit = Math.max(10, Math.min(200, action.limit));
  const past = state.past.slice(-limit);
  if (action.type === "undo") {
    if (!past.length) return state;
    return {
      present: past[past.length - 1],
      past: past.slice(0, -1),
      future: [state.present, ...state.future].slice(0, limit),
      group: null,
      changedAt: action.at,
    };
  }
  if (action.type === "redo") {
    if (!state.future.length) return state;
    return {
      present: state.future[0],
      past: [...past, state.present].slice(-limit),
      future: state.future.slice(1),
      group: null,
      changedAt: action.at,
    };
  }
  if (!action.value) return state;
  const next = action.value(state.present);
  if (JSON.stringify(next) === JSON.stringify(state.present)) return state;
  const coalesce =
    action.group != null &&
    action.group === state.group &&
    action.at - state.changedAt <= 500 &&
    !state.future.length;
  return {
    present: structuredClone(next),
    past: coalesce ? past : [...past, state.present].slice(-limit),
    future: [],
    group: action.group ?? null,
    changedAt: action.at,
  };
}
