/* eslint-disable react-refresh/only-export-components -- provider and hooks share one context module */
import type { AgentPromptUsageSeries, AgentRunTelemetry } from "@/lib/agentRunTelemetry";
import { emptyAgentRunTelemetry } from "@/lib/agentRunTelemetry";
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

const EMPTY_TELEMETRY = emptyAgentRunTelemetry();

export type PhaseAgentUsageEntry = {
  nodeId: string;
  name: string;
  telemetry: AgentRunTelemetry;
};

type PhaseAgentUsageContextValue = {
  report: (nodeId: string, entries: PhaseAgentUsageEntry[]) => void;
  agents: PhaseAgentUsageEntry[];
};

const PhaseAgentUsageContext = createContext<PhaseAgentUsageContextValue>({
  report: () => undefined,
  agents: [],
});

export function PhaseAgentUsageProvider({ children }: { children: ReactNode }) {
  const [byNode, setByNode] = useState<Record<string, PhaseAgentUsageEntry[]>>({});
  const report = useCallback((nodeId: string, entries: PhaseAgentUsageEntry[]) => {
    setByNode((prev) => {
      const current = prev[nodeId];
      if (sameUsageEntries(current, entries)) {
        return prev;
      }
      return { ...prev, [nodeId]: entries };
    });
  }, []);
  const agents = useMemo(() => Object.values(byNode).flat(), [byNode]);
  const value = useMemo(() => ({ report, agents }), [report, agents]);

  return <PhaseAgentUsageContext.Provider value={value}>{children}</PhaseAgentUsageContext.Provider>;
}

export function usePhaseAgentUsageAgents(): PhaseAgentUsageEntry[] {
  return useContext(PhaseAgentUsageContext).agents;
}

export function useReportPhaseAgentUsage(nodeId: string, name: string, telemetry: AgentRunTelemetry, enabled: boolean) {
  const series = useMemo(() => [{ name, telemetry }], [name, telemetry]);
  useReportPhaseAgentUsageSeries(nodeId, name, series, enabled);
}

export function useReportPhaseAgentUsageSeries(
  nodeId: string,
  fallbackName: string,
  series: AgentPromptUsageSeries[],
  enabled: boolean,
) {
  const { report } = useContext(PhaseAgentUsageContext);
  useEffect(() => {
    if (!enabled) {
      return;
    }
    if (series.length === 0) {
      report(nodeId, [{ nodeId, name: fallbackName, telemetry: EMPTY_TELEMETRY }]);
      return;
    }
    report(
      nodeId,
      series.map((item, index) => ({
        nodeId: `${nodeId}:${index}`,
        name: item.name || fallbackName,
        telemetry: item.telemetry,
      })),
    );
  }, [enabled, fallbackName, nodeId, report, series]);
}

function sameUsageEntries(current: PhaseAgentUsageEntry[] | undefined, next: PhaseAgentUsageEntry[]): boolean {
  if (!current || current.length !== next.length) {
    return false;
  }
  return current.every(
    (entry, index) =>
      entry.nodeId === next[index]?.nodeId &&
      entry.name === next[index]?.name &&
      entry.telemetry === next[index]?.telemetry,
  );
}
