import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { AgentRunUsageChart } from "@/ui/agentRun/AgentRunUsageChart";
import { LoaderCircle } from "lucide-react";
import { useState } from "react";

import {
  usePhaseAgentUsageAgents,
  usePhaseAgentUsageLoading,
  type PhaseAgentUsageEntry,
} from "./phaseAgentUsageContext";

export const SHOW_USAGE_LABEL = "Show usage";
export const LOADING_USAGE_LABEL = "Loading usage...";

export function PhaseUsageSpendButton({
  phaseId,
  phaseName,
  spendLabel,
  live = false,
  className,
  onOpenChange,
}: {
  phaseId: string;
  phaseName: string;
  spendLabel: string;
  live?: boolean;
  className?: string;
  onOpenChange?: (open: boolean) => void;
}) {
  const agents = usePhaseAgentUsageAgents();
  const usageLoading = usePhaseAgentUsageLoading();
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        type="button"
        data-testid={`split-run-phase-usage-${phaseId}`}
        aria-label={SHOW_USAGE_LABEL}
        title={SHOW_USAGE_LABEL}
        className={className}
        onClick={() => {
          onOpenChange?.(true);
          setOpen(true);
        }}
      >
        {spendLabel}
      </button>
      <Dialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          onOpenChange?.(next);
        }}
      >
        <DialogContent
          size="large"
          className="max-h-[min(90vh,52rem)] w-[min(72rem,calc(100vw-2rem))] max-w-[min(72rem,calc(100vw-2rem))] min-w-0 overflow-x-hidden overflow-y-auto [grid-template-columns:minmax(0,1fr)]"
          onClick={(event) => event.stopPropagation()}
        >
          <DialogHeader>
            <DialogTitle>{phaseName} usage</DialogTitle>
            <DialogDescription>Token usage and tool calls for each agent in this automation.</DialogDescription>
          </DialogHeader>
          <PhaseUsageDialogBody agents={agents} usageLoading={usageLoading} live={live} />
        </DialogContent>
      </Dialog>
    </>
  );
}

function PhaseUsageDialogBody({
  agents,
  usageLoading,
  live,
}: {
  agents: PhaseAgentUsageEntry[];
  usageLoading: boolean;
  live: boolean;
}) {
  const hasUsageTurns = agents.some((agent) => agent.telemetry.turns.length > 0);
  if (usageLoading && !hasUsageTurns) {
    return (
      <p role="status" className="flex items-center gap-2 py-2 text-sm text-muted-foreground">
        <LoaderCircle className="size-3.5 shrink-0 animate-spin text-[color:var(--status-running-dot)]" aria-hidden />
        {LOADING_USAGE_LABEL}
      </p>
    );
  }
  if (!hasUsageTurns) {
    return <p className="text-sm text-muted-foreground">No usage data yet.</p>;
  }
  return (
    <div className="min-w-0 space-y-8">
      {agents.map((agent) => (
        <AgentRunUsageChart
          key={agent.nodeId}
          telemetry={agent.telemetry}
          title={agents.length > 1 ? agent.name : undefined}
          live={live}
        />
      ))}
    </div>
  );
}
