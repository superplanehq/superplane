import { useEffect } from "react";
import { useLocation } from "react-router-dom";
import {
  clearPageObservabilityTag,
  sendPageObservabilityStart,
  setPageObservabilityTag,
} from "@/lib/dash0Observability";
import { resolvePageObservability } from "@/lib/pageObservability";

export function usePageObservability(): void {
  const location = useLocation();

  useEffect(() => {
    const context = resolvePageObservability(location.pathname);
    if (!context) {
      clearPageObservabilityTag();
      return;
    }

    setPageObservabilityTag(context.pageKey);
    sendPageObservabilityStart(context.pageKey, context.attributes);

    return () => {
      clearPageObservabilityTag();
    };
  }, [location.pathname]);
}
