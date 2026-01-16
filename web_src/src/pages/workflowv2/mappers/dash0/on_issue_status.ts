import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { formatTimeAgo } from "@/utils/date";

interface OnIssueStatusConfiguration {
  minutesInterval?: number;
  listenToAllCheckRules?: boolean;
  checkRules?: string[];
}

interface OnIssueStatusMetadata {
  nextTrigger?: string;
  referenceTime?: string;
}

interface OnIssueStatusEventData {
  query?: string;
  dataset?: string;
  results?: Array<{
    metric?: Record<string, string>;
    value?: [number, string];
    values?: Array<[number, string]>;
  }>;
  count?: number;
}

interface CheckTimelineEntry {
  label: string;
  status: string;
  timestamp?: string;
  comment?: string;
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

function formatIssueTitle(eventData: OnIssueStatusEventData | undefined): string {
  if (!eventData?.results || !Array.isArray(eventData.results)) {
    const count = eventData?.count || 0;
    return `${count} issue${count === 1 ? "" : "s"} detected`;
  }

  let criticalCount = 0;
  let degradedCount = 0;

  eventData.results.forEach((result) => {
    let severity = "UNKNOWN";
    if (result.value && Array.isArray(result.value) && result.value.length >= 2) {
      const severityValue = typeof result.value[1] === "string" ? parseFloat(result.value[1]) : result.value[1];
      if (severityValue === 2) {
        severity = "CRITICAL";
      } else if (severityValue === 1) {
        severity = "DEGRADED";
      }
    }
    // Also check for severity in labels
    const metric = result.metric || {};
    if (metric["severity"]) {
      const severityLabel = metric["severity"].toUpperCase();
      if (severityLabel === "CRITICAL" || severityLabel === "DEGRADED") {
        severity = severityLabel;
      }
    }

    if (severity === "CRITICAL") {
      criticalCount++;
    } else if (severity === "DEGRADED") {
      degradedCount++;
    }
  });

  const parts: string[] = [];
  if (criticalCount > 0) {
    parts.push(`${criticalCount} critical`);
  }
  if (degradedCount > 0) {
    parts.push(`${degradedCount} degraded`);
  }

  if (parts.length === 0) {
    const totalCount = eventData?.count || eventData.results.length;
    return `${totalCount} issue${totalCount === 1 ? "" : "s"} detected`;
  }

  return `Issues: ${parts.join(", ")}`;
}

/**
 * Renderer for the "dash0.onIssueStatus" trigger type
 */
export const onIssueStatusTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnIssueStatusEventData;

    return {
      title: formatIssueTitle(eventData),
      subtitle: formatTimeAgo(new Date(event.createdAt!)),
    };
  },

  getRootEventValues: (event: WorkflowsWorkflowEvent): Record<string, any> => {
    const eventData = event.data?.data as OnIssueStatusEventData;
    const values: Record<string, any> = {};

    // Add "Received at" timestamp
    if (event.createdAt) {
      values["Received at"] = new Date(event.createdAt).toLocaleString();
    }

    // Parse results and create checks timeline
    if (eventData?.results && Array.isArray(eventData.results)) {
      const checks: CheckTimelineEntry[] = eventData.results.map((result) => {
        const metric = result.metric || {};
        
        // Extract check name from dash0_check_name label
        const checkName = metric["dash0_check_name"] || metric["check_rule_name"] || "Unknown Check";

        // Extract summary from dash0_check_summary_template label (preferred) or description template
        const summary = metric["dash0_check_summary_template"] || metric["dash0_check_description_template"] || "";

        // Extract severity from value array (value[1] where "2" = CRITICAL, "1" = DEGRADED)
        let severity = "UNKNOWN";
        if (result.value && Array.isArray(result.value) && result.value.length >= 2) {
          const severityValue = typeof result.value[1] === "string" ? parseFloat(result.value[1]) : result.value[1];
          if (severityValue === 2) {
            severity = "CRITICAL";
          } else if (severityValue === 1) {
            severity = "DEGRADED";
          }
        }

        // Format label with status after check name, separated by middle dot
        // Keep status populated for dot color, but mark it as combined in label
        const statusText = severity === "CRITICAL" ? "CRITICAL" : severity === "DEGRADED" ? "DEGRADED" : "";
        const labelWithStatus = statusText ? `${checkName} · ${statusText}` : checkName;

        return {
          label: labelWithStatus,
          status: severity, // Keep status for dot color indicator
          comment: summary || undefined,
          // Add a flag to indicate status is combined in label (we'll check for " · " in the renderer)
        };
      });

      if (checks.length > 0) {
        // Sort checks: CRITICAL first, then DEGRADED, then others
        checks.sort((a, b) => {
          if (a.status === "CRITICAL" && b.status !== "CRITICAL") return -1;
          if (a.status !== "CRITICAL" && b.status === "CRITICAL") return 1;
          if (a.status === "DEGRADED" && b.status !== "DEGRADED") return -1;
          if (a.status !== "DEGRADED" && b.status === "DEGRADED") return 1;
          return 0;
        });

        // Use ApprovalTimelineEntry format for timeline rendering
        values["Checks"] = checks as unknown as Array<{
          label: string;
          status: string;
          timestamp?: string;
          comment?: string;
        }>;
      }
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

      props.lastEventData = {
        title: formatIssueTitle(eventData),
        subtitle: formatTimeAgo(new Date(lastEvent.createdAt!)),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
