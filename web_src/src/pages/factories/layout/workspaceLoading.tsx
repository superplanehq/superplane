import {
  lastLoadingMessage,
  nextLoadingMessages,
  workspaceLoadingOverlay,
  WorkspaceLoadingContext,
} from "@/hooks/useWorkspaceLoading";
import { prefersReducedMotion } from "@/lib/streamWords";
import { WORKSPACE_LOADING_EXIT_MS } from "@/lib/workspaceLoadingCopy";
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";

import { WorkspaceLoadingScreen } from "./WorkspaceLoadingScreen";

export function WorkspaceLoadingProvider({ children }: { children: ReactNode }) {
  const [messages, setMessages] = useState<Record<string, string>>({});
  const [leavingMessage, setLeavingMessage] = useState<string>();
  const lastActiveMessage = useRef<string>(undefined);
  const report = useCallback((id: string, message: string, pending: boolean) => {
    setMessages((current) => nextLoadingMessages(current, id, message, pending));
  }, []);
  const value = useMemo(() => ({ report }), [report]);
  const message = lastLoadingMessage(messages);
  if (message) {
    lastActiveMessage.current = message;
  }
  const overlay = workspaceLoadingOverlay(message, leavingMessage, lastActiveMessage.current, prefersReducedMotion());

  useEffect(() => {
    if (message) {
      setLeavingMessage(undefined);
      return;
    }
    const last = lastActiveMessage.current;
    if (!last) {
      return;
    }
    if (prefersReducedMotion()) {
      lastActiveMessage.current = undefined;
      setLeavingMessage(undefined);
      return;
    }
    setLeavingMessage(last);
    const timeout = window.setTimeout(() => {
      lastActiveMessage.current = undefined;
      setLeavingMessage(undefined);
    }, WORKSPACE_LOADING_EXIT_MS);
    return () => window.clearTimeout(timeout);
  }, [message]);

  return (
    <WorkspaceLoadingContext.Provider value={value}>
      {children}
      {overlay.message ? <WorkspaceLoadingScreen message={overlay.message} exiting={overlay.exiting} /> : null}
    </WorkspaceLoadingContext.Provider>
  );
}
