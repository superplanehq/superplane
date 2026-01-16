import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { formatTimeAgo } from "@/utils/date";

interface OnIssueStatusConfiguration {
  minutesInterval?: number;
  checkRules?: string[];
}

interface OnIssueStatusMetadata {
  nextTrigger?: string;
  referenceTime?: string;
}

interface OnIssueStatusEventData {
  query?: string;
  dataset?: string;
  results?: any[];
  count?: number;
}

function formatFrequency(configuration: OnIssueStatusConfiguration): string {
  if (!configuration.minutesInterval) {
    return "Not configured";
  }

  const interval = configuration.minutesInterval;
  return `Every ${interval} minute${interval === 1 ? "" : "s"}`;
}

function calculateNextTrigger(
  configuration: OnIssueStatusConfiguration,
  metadata?: OnIssueStatusMetadata,
): Date | null {
  // Always use backend-calculated nextTrigger first if available
  if (metadata?.nextTrigger) {
    try {
      return new Date(metadata.nextTrigger);
    } catch {
      // Fall through to frontend calculation if parsing fails
    }
  }

  if (!configuration.minutesInterval) return null;

  const now = new Date();
  const interval = configuration.minutesInterval;

  if (interval < 1 || interval > 59) return null;

  // Use reference time if available, otherwise use current time
  let reference = now;
  if (metadata?.referenceTime) {
    try {
      reference = new Date(metadata.referenceTime);
    } catch {
      // Use current time if parsing fails
    }
  }

  const minutesElapsed = Math.floor((now.getTime() - reference.getTime()) / (60 * 1000));
  const completedIntervals = Math.floor(Math.max(0, minutesElapsed) / interval);
  const nextTriggerMinutes = (completedIntervals + 1) * interval;

  const nextTrigger = new Date(reference.getTime() + nextTriggerMinutes * 60 * 1000);

  // If nextTrigger is in the past or now, add another interval
  if (nextTrigger <= now) {
    nextTrigger.setTime(nextTrigger.getTime() + interval * 60 * 1000);
  }

  return nextTrigger;
}

function formatNextTrigger(configuration: OnIssueStatusConfiguration, metadata?: OnIssueStatusMetadata): string {
  const nextTrigger = calculateNextTrigger(configuration, metadata);

  if (!nextTrigger) {
    return "-";
  }

  try {
    const now = new Date();
    const diffMs = nextTrigger.getTime() - now.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins <= 0) {
      return "Checking soon...";
    }

    if (diffMins < 60) {
      return `Next: in ${diffMins}m`;
    }

    if (diffMins < 1440) {
      return `Next: in ${Math.floor(diffMins / 60)}h`;
    }

    return formatTimestampInUserTimezone(nextTrigger.toISOString(), "UTC");
  } catch {
    return "";
  }
}

/**
 * Renderer for the "dash0.onIssueStatus" trigger type
 */
export const onIssueStatusTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnIssueStatusEventData;
    const count = eventData?.count || 0;

    return {
      title: `${count} issue${count === 1 ? "" : "s"} detected`,
      subtitle: formatTimeAgo(new Date(event.createdAt!)),
    };
  },

  getRootEventValues: (event: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = event.data?.data as OnIssueStatusEventData;
    const values: Record<string, string> = {};

    if (eventData?.query) {
      values["Query"] = eventData.query;
    }

    if (eventData?.dataset) {
      values["Dataset"] = eventData.dataset;
    }

    if (eventData?.count !== undefined) {
      values["Issues Found"] = eventData.count.toString();
    }

    return values;
  },

  getTriggerProps: (node: ComponentsNode, _trigger: TriggersTrigger, lastEvent?: WorkflowsWorkflowEvent) => {
    const configuration = node.configuration as unknown as OnIssueStatusConfiguration;
    const metadata = node.metadata as unknown as OnIssueStatusMetadata;

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: dash0Icon,
      iconBackground: "bg-white",
      headerColor: "bg-white",
      collapsedBackground: "bg-white",
      metadata: [
        {
          icon: "clock",
          label: formatFrequency(configuration),
        },
        {
          icon: "arrow-big-right",
          label: formatNextTrigger(configuration, metadata),
        },
      ],
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnIssueStatusEventData;
      const count = eventData?.count || 0;

      props.lastEventData = {
        title: `${count} issue${count === 1 ? "" : "s"} detected`,
        subtitle: formatTimeAgo(new Date(lastEvent.createdAt!)),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
