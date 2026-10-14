import { useReducer } from "react";
import {
  newEditHistory,
  reduceEditHistory,
  type EditorValue,
} from "../editHistory";
import type { Settings } from "../bridge";

export function useEditorHistory(initial: () => EditorValue, limit: number) {
  const [history, dispatch] = useReducer(reduceEditHistory, undefined, () =>
    newEditHistory(initial()),
  );
  function update(
    value: (current: EditorValue) => EditorValue,
    group: string | null = null,
  ) {
    dispatch({ type: "edit", value, group, at: Date.now(), limit });
  }
  return {
    ...history.present,
    canUndo: history.past.length > 0,
    canRedo: history.future.length > 0,
    undo: () => dispatch({ type: "undo", at: Date.now(), limit }),
    redo: () => dispatch({ type: "redo", at: Date.now(), limit }),
    update,
    setSettings: (
      value: Settings | ((current: Settings) => Settings),
      group = "settings",
    ) =>
      update(
        (current) => ({
          ...current,
          settings:
            typeof value === "function" ? value(current.settings) : value,
        }),
        group,
      ),
    setOutput: (value: string | ((current: string) => string)) =>
      update(
        (current) => ({
          ...current,
          output: typeof value === "function" ? value(current.output) : value,
        }),
        "output",
      ),
  };
}
