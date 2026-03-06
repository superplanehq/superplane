import { CustomFieldRenderer, NodeInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import newrelicIcon from "@/assets/icons/integrations/newrelic.svg";
import { NewRelicIssuePayload, OnIssueConfiguration } from "./types";

const stateLabels: Record<string, string> = {
  CREATED: "Created",
  ACTIVATED: "Activated",
  ACKNOWLEDGED: "Acknowledged",
  CLOSED: "Closed",
};

const priorityLabels: Record<string, string> = {
  CRITICAL: "Critical",
  HIGH: "High",
  MEDIUM: "Medium",
  LOW: "Low",
};

export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as NewRelicIssuePayload;
    const title = buildEventTitle(eventData);
    const subtitle = buildEventSubtitle(eventData, context.event?.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as NewRelicIssuePayload;
    return getDetailsForIssue(eventData);
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnIssueConfiguration | undefined;
    const metadataItems = [];

    if (configuration?.statuses && configuration.statuses.length > 0) {
      const formattedStatuses = configuration.statuses
        .map((status) => stateLabels[status] || status)
        .filter((status, index, values) => values.indexOf(status) === index);

      metadataItems.push({
        icon: "funnel",
        label: `Statuses: ${formattedStatuses.join(", ")}`,
      });
    }

    if (configuration?.priorities && configuration.priorities.length > 0) {
      const formattedPriorities = configuration.priorities
        .map((priority) => priorityLabels[priority] || priority)
        .filter((priority, index, values) => values.indexOf(priority) === index);

      metadataItems.push({
        icon: "flag",
        label: `Priorities: ${formattedPriorities.join(", ")}`,
      });
    }

    metadataItems.push({
      icon: "link",
      label: "Webhook auto-configured",
    });

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: newrelicIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems.slice(0, 3),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as NewRelicIssuePayload;
      props.lastEventData = {
        title: buildEventTitle(eventData),
        subtitle: buildEventSubtitle(eventData, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

export const onIssueCustomFieldRenderer: CustomFieldRenderer = {
  render: (_node: NodeInfo) => {
    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">New Relic Webhook Setup</span>
            <p className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md">
              SuperPlane automatically creates a Webhook Notification Channel in your New Relic account. Just attach it
              to your alert workflow to start receiving alerts.
            </p>
          </div>
        </div>
      </div>
    );
  },
};

function buildEventTitle(eventData: NewRelicIssuePayload): string {
  const title = eventData?.title || "Issue";
  const state = eventData?.state ? stateLabels[eventData.state] || eventData.state : "";

  if (state) {
    return `${title} · ${state}`;
  }

  return title;
}

function buildEventSubtitle(eventData: NewRelicIssuePayload, createdAt?: string): string {
  const parts: string[] = [];

  if (eventData?.priority) {
    parts.push(priorityLabels[eventData.priority] || eventData.priority);
  }

  if (createdAt) {
    parts.push(formatTimeAgo(new Date(createdAt)));
  }

  return parts.join(" · ");
}

function getDetailsForIssue(eventData: NewRelicIssuePayload): Record<string, string> {
  const details: Record<string, string> = {};

  if (eventData?.issueId) {
    details["Issue ID"] = eventData.issueId;
  }

  if (eventData?.state) {
    details["State"] = stateLabels[eventData.state] || eventData.state;
  }

  if (eventData?.priority) {
    details["Priority"] = priorityLabels[eventData.priority] || eventData.priority;
  }

  if (eventData?.policyName) {
    details["Policy"] = eventData.policyName;
  }

  if (eventData?.conditionName) {
    details["Condition"] = eventData.conditionName;
  }

  if (eventData?.accountId) {
    details["Account ID"] = String(eventData.accountId);
  }

  if (eventData?.createdAt) {
    details["Created At"] = new Date(eventData.createdAt).toLocaleString();
  }

  if (eventData?.issueUrl) {
    details["Issue URL"] = eventData.issueUrl;
  }

  return details;
}
