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

type NodeUsageReport = {
  entries: PhaseAgentUsageEntry[];
  loading: boolean;
};

type PhaseAgentUsageContextValue = {
  report: (nodeId: string, entries: PhaseAgentUsageEntry[], loading?: boolean) => void;
  agents: PhaseAgentUsageEntry[];
  isLoading: boolean;
};

const PhaseAgentUsageContext = createContext<PhaseAgentUsageContextValue>({
  report: () => undefined,
  agents: [],
  isLoading: false,
});

export function PhaseAgentUsageProvider({
  children,
  streamLoading = false,
  expectUsage = false,
}: {
  children: ReactNode;
  streamLoading?: boolean;
  expectUsage?: boolean;
}) {
  const [byNode, setByNode] = useState<Record<string, NodeUsageReport>>({});
  const report = useCallback((nodeId: string, entries: PhaseAgentUsageEntry[], loading = false) => {
    setByNode((prev) => {
      const current = prev[nodeId];
      if (current && current.loading === loading && sameUsageEntries(current.entries, entries)) {
        return prev;
      }
      return { ...prev, [nodeId]: { entries, loading } };
    });
  }, []);
  const agents = useMemo(() => Object.values(byNode).flatMap((node) => node.entries), [byNode]);
  const nodeLoading = Object.values(byNode).some((node) => node.loading);
  const hasReport = Object.keys(byNode).length > 0;
  const isLoading = streamLoading || nodeLoading || (expectUsage && !hasReport);
  const value = useMemo(() => ({ report, agents, isLoading }), [report, agents, isLoading]);

  return <PhaseAgentUsageContext.Provider value={value}>{children}</PhaseAgentUsageContext.Provider>;
}

export function usePhaseAgentUsageAgents(): PhaseAgentUsageEntry[] {
  return useContext(PhaseAgentUsageContext).agents;
}

export function usePhaseAgentUsageLoading(): boolean {
  return useContext(PhaseAgentUsageContext).isLoading;
}

export function useReportPhaseAgentUsage(nodeId: string, name: string, telemetry: AgentRunTelemetry, enabled: boolean) {
  const series = useMemo(() => [{ name, telemetry }], [name, telemetry]);
  useReportPhaseAgentUsageSeries({ nodeId, fallbackName: name, series, enabled });
}

export function useReportPhaseAgentUsageSeries({
  nodeId,
  fallbackName,
  series,
  enabled,
  loading = false,
}: {
  nodeId: string;
  fallbackName: string;
  series: AgentPromptUsageSeries[];
  enabled: boolean;
  loading?: boolean;
}) {
  const { report } = useContext(PhaseAgentUsageContext);
  useEffect(() => {
    if (!enabled) {
      return;
    }
    if (series.length === 0) {
      report(nodeId, loading ? [] : [{ nodeId, name: fallbackName, telemetry: EMPTY_TELEMETRY }], loading);
      return;
    }
    report(
      nodeId,
      series.map((item, index) => ({
        nodeId: `${nodeId}:${index}`,
        name: item.name || fallbackName,
        telemetry: item.telemetry,
      })),
      loading,
    );
  }, [enabled, fallbackName, loading, nodeId, report, series]);
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
