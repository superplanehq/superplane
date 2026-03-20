import {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "./types";
import { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatTimeAgo } from "@/utils/date";
import { getTriggerRenderer } from ".";

const SEND_EMAIL_EVENT_STATE_MAP: EventStateMap = {
  triggered: {
    icon: "circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-violet-100",
    badgeColor: "bg-violet-400",
  },
  sent: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "Sent",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  cancelled: {
    icon: "circle-slash-2",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  error: {
    icon: "triangle-alert",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
  neutral: {
    icon: "circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-50",
    badgeColor: "bg-gray-400",
  },
  queued: {
    icon: "circle-dashed",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
  running: {
    icon: "refresh-cw",
    textColor: "text-gray-800",
    backgroundColor: "bg-sky-100",
    badgeColor: "bg-blue-500",
  },
};

const sendEmailStateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    return "sent";
  }

  return "failed";
};

export const SEND_EMAIL_STATE_REGISTRY: EventStateRegistry = {
  stateMap: SEND_EMAIL_EVENT_STATE_MAP,
  getState: sendEmailStateFunction,
};

type SendEmailConfiguration = {
  recipients?: Array<{ type: string; user?: string; role?: string; group?: string }>;
  subject?: string;
};

export const sendEmailMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      iconSlug: context.componentDefinition.icon || "mail",
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Send Email Notification",
      eventSections: lastExecution ? getSendEmailEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getSendEmailMetadata(context.node),
      eventStateMap: SEND_EMAIL_EVENT_STATE_MAP,
    };
  },

  subtitle(context: SubtitleContext): string {
    const state = sendEmailStateFunction(context.execution);

    if (state === "running") {
      return "Sending...";
    }

    if (state === "sent") {
      const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
      const payload = outputs?.default?.[0]?.data as { subject?: string; to?: string[] } | undefined;
      const recipientCount = payload?.to?.length ?? 0;
      const timeAgo = context.execution.updatedAt ? formatTimeAgo(new Date(context.execution.updatedAt)) : "";

      if (recipientCount > 0 && timeAgo) {
        return `Sent to ${recipientCount} recipient${recipientCount > 1 ? "s" : ""} · ${timeAgo}`;
      }

      return timeAgo || "Sent";
    }

    if (context.execution.updatedAt) {
      return formatTimeAgo(new Date(context.execution.updatedAt));
    }

    return "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0]?.data as
      | {
          subject?: string;
          to?: string[];
          groups?: string[];
          roles?: string[];
        }
      | undefined;

    if (payload?.subject) {
      details["Subject"] = payload.subject;
    }

    if (payload?.to && payload.to.length > 0) {
      details["To"] = payload.to.join(", ");
    }

    if (payload?.groups && payload.groups.length > 0) {
      details["Groups"] = payload.groups.join(", ");
    }

    if (payload?.roles && payload.roles.length > 0) {
      details["Roles"] = payload.roles.join(", ");
    }

    return details;
  },
};

function RecipientsLabel({ recipients }: { recipients: SendEmailConfiguration["recipients"] }) {
  if (!recipients || recipients.length === 0) {
    return null;
  }

  const count = recipients.length;
  const label = `${count} recipient${count > 1 ? "s" : ""}`;

  const counts = { user: 0, role: 0, group: 0 };
  for (const r of recipients) {
    if (r.type in counts) counts[r.type as keyof typeof counts]++;
  }

  const parts: string[] = [];
  if (counts.user > 0) parts.push(`${counts.user} user${counts.user > 1 ? "s" : ""}`);
  if (counts.role > 0) parts.push(`${counts.role} role${counts.role > 1 ? "s" : ""}`);
  if (counts.group > 0) parts.push(`${counts.group} group${counts.group > 1 ? "s" : ""}`);

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="cursor-default underline underline-offset-3 decoration-dotted decoration-1">{label}</span>
      </TooltipTrigger>
      <TooltipContent side="bottom">{parts.join(", ")}</TooltipContent>
    </Tooltip>
  );
}

function getSendEmailMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as SendEmailConfiguration;
  const metadata: MetadataItem[] = [];

  if (configuration.subject) {
    metadata.push({ icon: "text", label: configuration.subject });
  }

  if (configuration.recipients && configuration.recipients.length > 0) {
    metadata.push({ icon: "users", label: <RecipientsLabel recipients={configuration.recipients} /> });
  }

  return metadata;
}

function getSendEmailEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: sendEmailStateFunction(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
