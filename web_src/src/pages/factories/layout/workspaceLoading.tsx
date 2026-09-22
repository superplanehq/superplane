import { lastLoadingMessage, nextLoadingMessages, WorkspaceLoadingContext } from "@/hooks/useWorkspaceLoading";
import { prefersReducedMotion } from "@/lib/streamWords";
import { WORKSPACE_LOADING_EXIT_MS } from "@/lib/workspaceLoadingCopy";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

import { WorkspaceLoadingScreen } from "./WorkspaceLoadingScreen";

export function WorkspaceLoadingProvider({ children }: { children: ReactNode }) {
  const [messages, setMessages] = useState<Record<string, string>>({});
  const [visibleMessage, setVisibleMessage] = useState<string>();
  const [exiting, setExiting] = useState(false);
  const report = useCallback((id: string, message: string, pending: boolean) => {
    setMessages((current) => nextLoadingMessages(current, id, message, pending));
  }, []);
  const value = useMemo(() => ({ report }), [report]);
  const message = lastLoadingMessage(messages);

  useEffect(() => {
    if (message) {
      setVisibleMessage(message);
      setExiting(false);
      return;
    }
    if (!visibleMessage) {
      return;
    }
    if (prefersReducedMotion()) {
      setVisibleMessage(undefined);
      setExiting(false);
      return;
    }
    setExiting(true);
    const timeout = window.setTimeout(() => {
      setVisibleMessage(undefined);
      setExiting(false);
    }, WORKSPACE_LOADING_EXIT_MS);
    return () => window.clearTimeout(timeout);
  }, [message, visibleMessage]);

  return (
    <WorkspaceLoadingContext.Provider value={value}>
      {children}
      {visibleMessage ? <WorkspaceLoadingScreen message={visibleMessage} exiting={exiting} /> : null}
    </WorkspaceLoadingContext.Provider>
  );
}
