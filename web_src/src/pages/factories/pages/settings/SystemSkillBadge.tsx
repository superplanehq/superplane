import type { FactoryAgentResourceSystemSkillState } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

export function SystemSkillBadge({ state }: { state?: FactoryAgentResourceSystemSkillState }) {
  const status = state && state !== "STATE_UNSPECIFIED" ? AGENT_RESOURCES_COPY.systemSkillStates[state] : undefined;
  return (
    <>
      <SkillBadgeHint
        label={AGENT_RESOURCES_COPY.systemSkillLabel}
        explanation={AGENT_RESOURCES_COPY.systemSkillExplanation}
      />
      {status ? <SkillBadgeHint {...status} outline /> : null}
    </>
  );
}

function SkillBadgeHint({
  label,
  explanation,
  outline = false,
}: {
  label: string;
  explanation: string;
  outline?: boolean;
}) {
  return (
    <Tooltip delayDuration={200}>
      <TooltipTrigger asChild>
        <Badge variant={outline ? "outline" : "secondary"} className="shrink-0 cursor-help" tabIndex={0}>
          {label}
        </Badge>
      </TooltipTrigger>
      <TooltipContent side="top" sideOffset={6} className="z-[10000] max-w-72">
        {explanation}
      </TooltipContent>
    </Tooltip>
  );
}
