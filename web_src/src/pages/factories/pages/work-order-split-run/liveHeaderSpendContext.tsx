/* eslint-disable react-refresh/only-export-components -- provider and hooks share one context module */
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

import { EMPTY_LIVE_HEADER_SPEND, sumLiveHeaderSpend, type LiveHeaderSpend } from "@/lib/overlayHeaderSpend";

type LiveHeaderSpendContextValue = {
  report: (phaseId: string, spend: LiveHeaderSpend) => void;
  overlay: LiveHeaderSpend;
};

const LiveHeaderSpendContext = createContext<LiveHeaderSpendContextValue>({
  report: () => undefined,
  overlay: EMPTY_LIVE_HEADER_SPEND,
});

export function LiveHeaderSpendProvider({ children }: { children: ReactNode }) {
  const [byPhase, setByPhase] = useState<Record<string, LiveHeaderSpend>>({});
  const report = useCallback((phaseId: string, spend: LiveHeaderSpend) => {
    setByPhase((prev) => {
      const current = prev[phaseId];
      const next: LiveHeaderSpend = {
        tokens: Math.max(current?.tokens ?? 0, spend.tokens),
        cents: Math.max(current?.cents ?? 0, spend.cents),
      };
      if (current && current.tokens === next.tokens && current.cents === next.cents) {
        return prev;
      }
      return { ...prev, [phaseId]: next };
    });
  }, []);
  const overlay = useMemo(() => sumLiveHeaderSpend(Object.values(byPhase)), [byPhase]);
  const value = useMemo(() => ({ report, overlay }), [report, overlay]);

  return <LiveHeaderSpendContext.Provider value={value}>{children}</LiveHeaderSpendContext.Provider>;
}

export function useReportLiveHeaderSpend(phaseId: string, tokens: number, cents: number) {
  const { report } = useContext(LiveHeaderSpendContext);
  useEffect(() => {
    report(phaseId, { tokens, cents });
  }, [cents, phaseId, report, tokens]);
}

export function useLiveHeaderSpendOverlay(): LiveHeaderSpend {
  return useContext(LiveHeaderSpendContext).overlay;
}
