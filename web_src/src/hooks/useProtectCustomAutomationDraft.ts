import { useEffect } from "react";
import { useLocation } from "react-router";
import { customAutomationIdFromSearch } from "@/pages/factories/lib/customAutomationPaths";

export function useProtectCustomAutomationDraft(dirty: boolean) {
  const location = useLocation();

  useEffect(() => {
    if (!dirty) return;

    const automationId = customAutomationIdFromSearch(location.search);
    const editorURL = window.location.href;
    const editorState: unknown = window.history.state;
    const editorIndex = historyIndex(editorState);

    const onUnload = (event: BeforeUnloadEvent) => event.preventDefault();
    const onHistoryChange = (event: PopStateEvent) => {
      const sameEditor =
        window.location.pathname === location.pathname &&
        customAutomationIdFromSearch(window.location.search) === automationId;
      if (sameEditor || window.confirm("Discard unsaved changes to this automation?")) return;

      // BrowserRouter listens for popstate in the bubble phase. Keep its current
      // location until the browser restores the editor's history entry.
      event.stopImmediatePropagation();
      const destinationIndex = historyIndex(event.state);
      if (editorIndex !== undefined && destinationIndex !== undefined && editorIndex !== destinationIndex) {
        window.history.go(editorIndex - destinationIndex);
        return;
      }
      window.history.pushState(editorState, "", editorURL);
    };

    window.addEventListener("beforeunload", onUnload);
    window.addEventListener("popstate", onHistoryChange, true);
    return () => {
      window.removeEventListener("beforeunload", onUnload);
      window.removeEventListener("popstate", onHistoryChange, true);
    };
  }, [dirty, location]);
}

function historyIndex(state: unknown): number | undefined {
  // React Router adds an index to its browser history entries.
  if (state && typeof state === "object" && "idx" in state && typeof state.idx === "number") return state.idx;
}
