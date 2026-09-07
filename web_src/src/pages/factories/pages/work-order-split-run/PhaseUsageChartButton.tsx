import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { AgentRunUsageChart } from "@/ui/agentRun/AgentRunUsageChart";
import { useState } from "react";

import { usePhaseAgentUsageAgents } from "./phaseAgentUsageContext";

export const SHOW_USAGE_LABEL = "Show usage";

export function PhaseUsageSpendButton({
  phaseId,
  phaseName,
  spendLabel,
  className,
}: {
  phaseId: string;
  phaseName: string;
  spendLabel: string;
  className?: string;
}) {
  const agents = usePhaseAgentUsageAgents();
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        type="button"
        data-testid={`split-run-phase-usage-${phaseId}`}
        aria-label={SHOW_USAGE_LABEL}
        title={SHOW_USAGE_LABEL}
        className={className}
        onClick={() => setOpen(true)}
      >
        {spendLabel}
      </button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent
          size="large"
          className="max-h-[min(90vh,52rem)] w-[min(72rem,calc(100vw-2rem))] max-w-[min(72rem,calc(100vw-2rem))] min-w-0 overflow-x-hidden overflow-y-auto [grid-template-columns:minmax(0,1fr)]"
          onClick={(event) => event.stopPropagation()}
        >
          <DialogHeader>
            <DialogTitle>{phaseName} usage</DialogTitle>
            <DialogDescription>Token usage and tool calls for each agent in this automation.</DialogDescription>
          </DialogHeader>
          <div className="min-w-0 space-y-8">
            {agents.length === 0 ? (
              <p className="text-sm text-muted-foreground">No usage data yet.</p>
            ) : (
              agents.map((agent) => (
                <AgentRunUsageChart
                  key={agent.nodeId}
                  telemetry={agent.telemetry}
                  title={agents.length > 1 ? agent.name : undefined}
                />
              ))
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
