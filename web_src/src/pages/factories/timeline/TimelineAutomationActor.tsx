import { SimpleTooltip } from "@/ui/componentSidebar/SimpleTooltip";

import type { WorkOrderTimelineAutomationActor } from "../lib/workOrderTimelineEvents";
import { formatAutomationLabel } from "./authorLabels";

interface TimelineAutomationActorProps {
  actor: WorkOrderTimelineAutomationActor;
  fallbackLabel?: string;
}

export function TimelineAutomationActor({ actor, fallbackLabel = "Automation" }: TimelineAutomationActorProps) {
  const lineName = actor.lineName?.trim();
  const nodeName = actor.nodeName?.trim();
  const appName = actor.appName?.trim() || actor.stepName?.trim();
  const tooltip = formatStepNodeTooltip(appName, nodeName);

  if (lineName && tooltip) {
    return (
      <SimpleTooltip content={tooltip}>
        <span className="cursor-help font-medium underline decoration-dotted decoration-border underline-offset-2 hover:decoration-foreground">
          {lineName}
        </span>
      </SimpleTooltip>
    );
  }

  if (lineName) {
    return <span className="font-medium">{lineName}</span>;
  }

  const fallback = formatAutomationLabel(nodeName, appName) ?? fallbackLabel;
  return <span className="font-medium">{fallback}</span>;
}

function formatStepNodeTooltip(stepName: string | undefined, nodeName: string | undefined): string | undefined {
  const parts = [stepName, nodeName].filter((v): v is string => Boolean(v));
  return parts.length > 0 ? parts.join(" · ") : undefined;
}
