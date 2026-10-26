import { useCallback, useReducer } from "react";
import {
  emptySnackbarState,
  snackbarReducer,
  type SnackbarAction,
} from "../snackbar";

export function useSnackbar() {
  const [state, dispatch] = useReducer(snackbarReducer, emptySnackbarState);
  const show = useCallback(
    (
      message: string,
      options?: { action?: SnackbarAction; duration?: number },
    ) => {
      dispatch({
        type: "show",
        input: {
          message,
          action: options?.action,
          duration: options?.duration,
        },
      });
    },
    [],
  );
  const dismiss = useCallback((id?: number) => {
    dispatch({ type: "dismiss", id });
  }, []);
  return { notice: state.current, show, dismiss };
}
