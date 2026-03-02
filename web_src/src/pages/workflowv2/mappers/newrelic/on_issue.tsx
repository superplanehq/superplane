import { useState } from "react";
import { CustomFieldRenderer, NodeInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { showErrorToast } from "@/utils/toast";
import { Icon } from "@/components/Icon";
import newrelicIcon from "@/assets/icons/integrations/newrelic.svg";
import { NewRelicIssuePayload, OnIssueConfiguration, OnIssueMetadata } from "./types";

const stateLabels: Record<string, string> = {
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
    const metadata = node.metadata as OnIssueMetadata | undefined;

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

    if (metadata?.webhookUrl) {
      metadataItems.push({
        icon: "link",
        label: "Webhook configured",
      });
    }

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

const CopyWebhookUrlButton: React.FC<{ webhookUrl: string }> = ({ webhookUrl }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(webhookUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (_err) {
      showErrorToast("Failed to copy webhook URL");
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="inline-flex items-center gap-1.5 px-2 py-1 text-xs font-medium text-gray-700 dark:text-gray-200 border-1 border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-900 hover:bg-gray-100 dark:hover:bg-gray-800"
      title={copied ? "Copied!" : "Copy webhook URL"}
    >
      <Icon name={copied ? "check" : "copy"} size="sm" />
      {copied ? "Copied" : "Copy"}
    </button>
  );
};

export const onIssueCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnIssueMetadata | undefined;
    const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">New Relic Webhook Setup</span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md">
              <ol className="list-decimal ml-4 space-y-1">
                <li>Save the canvas to generate the webhook URL.</li>
                <li>In New Relic, go to <strong>Alerts & AI → Destinations</strong> and create a Webhook destination with the URL below.</li>
                <li>Create a <strong>Workflow</strong> that sends notifications to the webhook destination.</li>
              </ol>
              <div className="mt-3">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-medium text-gray-700 dark:text-gray-200">Webhook URL</span>
                  {metadata?.webhookUrl && <CopyWebhookUrlButton webhookUrl={metadata.webhookUrl} />}
                </div>
                <pre className="mt-1 text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-white dark:bg-gray-900 rounded-md font-mono whitespace-pre-wrap break-all">{webhookUrl}</pre>
              </div>
            </div>
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

  if (eventData?.updatedAt) {
    details["Updated At"] = new Date(eventData.updatedAt).toLocaleString();
  }

  if (eventData?.issueUrl) {
    details["Issue URL"] = eventData.issueUrl;
  }

  return details;
}
