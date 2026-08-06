import { SimpleTooltip } from "@/ui/componentSidebar/SimpleTooltip";

import type { WorkOrderTimelineAutomationActor } from "../lib/workOrderTimelineEvents";
import { formatAutomationLabel } from "./authorLabels";

interface TimelineAutomationActorProps {
  actor: WorkOrderTimelineAutomationActor;
  fallbackLabel?: string;
}

export function TimelineAutomationActor({ actor, fallbackLabel = "Automation" }: TimelineAutomationActorProps) {
  const lineName = actor.lineName?.trim();
  const stepName = actor.stepName?.trim();
  const nodeName = actor.nodeName?.trim();
  const appName = actor.appName?.trim();
  const tooltip = formatStepNodeTooltip(stepName, nodeName);

  if (lineName && tooltip) {
    return (
      <SimpleTooltip content={tooltip}>
        <span className="font-semibold cursor-help underline decoration-dotted decoration-gray-400 underline-offset-2 hover:decoration-gray-500 dark:decoration-gray-500 dark:hover:decoration-gray-300">
          {lineName}
        </span>
      </SimpleTooltip>
    );
  }

  if (lineName) {
    return <span className="font-semibold">{lineName}</span>;
  }

  const fallback = formatAutomationLabel(nodeName, appName) ?? fallbackLabel;
  return <span className="font-semibold">{fallback}</span>;
}

function formatStepNodeTooltip(stepName: string | undefined, nodeName: string | undefined): string | undefined {
  const parts = [stepName, nodeName].filter((v): v is string => Boolean(v));
  return parts.length > 0 ? parts.join(" · ") : undefined;
}
