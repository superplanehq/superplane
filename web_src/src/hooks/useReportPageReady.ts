import { useEffect, useRef } from "react";
import { useLocation } from "react-router-dom";
import { sendPageObservabilityReady, type PageReadyAttributes } from "@/lib/dash0Observability";
import { resolvePageObservability } from "@/lib/pageObservability";

export function useReportPageReady(ready: boolean, attributes?: PageReadyAttributes): void {
  const location = useLocation();
  const reportedRef = useRef(false);
  const attributesRef = useRef(attributes);
  attributesRef.current = attributes;

  useEffect(() => {
    reportedRef.current = false;
  }, [location.pathname]);

  useEffect(() => {
    if (!ready || reportedRef.current) {
      return;
    }

    const context = resolvePageObservability(location.pathname);
    if (!context) {
      return;
    }

    reportedRef.current = true;
    sendPageObservabilityReady(context.pageKey, {
      ...context.attributes,
      ...(attributesRef.current ?? {}),
    });
  }, [ready, location.pathname]);
}
