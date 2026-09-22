import { createContext, useContext, useId, useLayoutEffect } from "react";

export type WorkspaceLoadingContextValue = {
  report: (id: string, message: string, pending: boolean) => void;
};

export const WorkspaceLoadingContext = createContext<WorkspaceLoadingContextValue | null>(null);

/** True when a shared overlay will show this pending state. */
export function useWorkspaceLoading(message: string, pending: boolean): boolean {
  const context = useContext(WorkspaceLoadingContext);
  const id = useId();

  useLayoutEffect(() => {
    if (!context) {
      return;
    }
    context.report(id, message, pending);
    return () => context.report(id, message, false);
  }, [context, id, message, pending]);

  return context != null;
}

export function nextLoadingMessages(
  current: Record<string, string>,
  id: string,
  message: string,
  pending: boolean,
): Record<string, string> {
  if (!pending) {
    if (!(id in current)) {
      return current;
    }
    const next = { ...current };
    delete next[id];
    return next;
  }
  if (current[id] === message) {
    return current;
  }
  return { ...current, [id]: message };
}

export function lastLoadingMessage(messages: Record<string, string>): string | undefined {
  const values = Object.values(messages);
  return values[values.length - 1];
}

/** Overlay shown this render. The active message wins so a new load does not wait for an effect. */
export function workspaceLoadingOverlay(
  activeMessage: string | undefined,
  leavingMessage: string | undefined,
  lastActiveMessage: string | undefined,
  reduceMotion: boolean,
): { message?: string; exiting: boolean } {
  if (activeMessage) {
    return { message: activeMessage, exiting: false };
  }
  if (reduceMotion) {
    return {};
  }
  const message = leavingMessage ?? lastActiveMessage;
  return { message, exiting: Boolean(message) };
}
