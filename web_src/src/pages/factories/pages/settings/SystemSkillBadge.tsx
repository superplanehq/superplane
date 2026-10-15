import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

export function SystemSkillBadge() {
  return (
    <Tooltip delayDuration={200}>
      <TooltipTrigger asChild>
        <Badge variant="secondary" className="shrink-0 cursor-help" tabIndex={0}>
          {AGENT_RESOURCES_COPY.systemSkillLabel}
        </Badge>
      </TooltipTrigger>
      <TooltipContent side="top" sideOffset={6} className="z-[10000] max-w-72">
        {AGENT_RESOURCES_COPY.systemSkillExplanation}
      </TooltipContent>
    </Tooltip>
  );
}
