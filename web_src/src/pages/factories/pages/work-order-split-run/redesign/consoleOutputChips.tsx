import { Badge } from "@/components/reui/badge";

import type { SplitRunPhase } from "../splitRunMocks";
import type { AutomationStage } from "./automationsViewModel";
import { outputCountLabel } from "./consoleCardText";
import { consolePages } from "./consolePages";

/**
 * Collapsed-card count of agent runs. Badges only — no hover card.
 */
export function StepOutputCounts({
  stage,
  phase,
  runs = [stage],
}: {
  stage: AutomationStage;
  phase?: SplitRunPhase;
  runs?: AutomationStage[];
}) {
  const runCount = consolePages(stage, phase, runs).includes("agent") ? runs.length : 0;
  if (runCount === 0) {
    return null;
  }
  return (
    <div className="flex min-w-0 items-center gap-1.5">
      <CountBadge label={outputCountLabel(runCount, "agent run", "agent runs")} />
    </div>
  );
}

function CountBadge({ label }: { label: string }) {
  return (
    <Badge variant="outline" className="h-5 px-1.5 text-[12px] font-normal">
      {label}
    </Badge>
  );
}
