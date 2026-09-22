import { lastLoadingMessage, nextLoadingMessages, WorkspaceLoadingContext } from "@/hooks/useWorkspaceLoading";
import { useCallback, useMemo, useState, type ReactNode } from "react";

import { WorkspaceLoadingScreen } from "./WorkspaceLoadingScreen";

export function WorkspaceLoadingProvider({ children }: { children: ReactNode }) {
  const [messages, setMessages] = useState<Record<string, string>>({});
  const report = useCallback((id: string, message: string, pending: boolean) => {
    setMessages((current) => nextLoadingMessages(current, id, message, pending));
  }, []);
  const value = useMemo(() => ({ report }), [report]);
  const message = lastLoadingMessage(messages);

  return (
    <WorkspaceLoadingContext.Provider value={value}>
      {children}
      {message ? <WorkspaceLoadingScreen message={message} /> : null}
    </WorkspaceLoadingContext.Provider>
  );
}
