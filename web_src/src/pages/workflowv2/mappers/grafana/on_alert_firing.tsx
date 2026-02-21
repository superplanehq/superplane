import { useState, type FC } from "react";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { CustomFieldRenderer, NodeInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import { OnAlertFiringEventData } from "./types";
import { stringOrDash } from "../utils";
import { formatTimestamp } from "./utils";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";

/**
 * Renderer for the "grafana.onAlertFiring" trigger
 */
export const onAlertFiringTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnAlertFiringEventData | undefined;
    const alertName = getAlertName(eventData);
    const status = eventData?.status || "firing";
    const subtitle = buildSubtitle(status, context.event?.createdAt);

    return {
      title: alertName || "Grafana alert firing",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnAlertFiringEventData | undefined;
    const createdAt = formatTimestamp(context.event?.createdAt);

    return {
      "Triggered At": createdAt,
      Status: stringOrDash(eventData?.status || "firing"),
      "Alert Name": stringOrDash(getAlertName(eventData)),
      "Rule UID": stringOrDash(eventData?.ruleUid),
      "Rule ID": stringOrDash(eventData?.ruleId),
      "Org ID": stringOrDash(eventData?.orgId),
      "External URL": stringOrDash(eventData?.externalURL),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadataItems = [];

    if (lastEvent?.data) {
      const eventData = lastEvent.data as OnAlertFiringEventData;
      const alertName = getAlertName(eventData);
      if (alertName) {
        metadataItems.push({
          icon: "bell",
          label: alertName,
        });
      }
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: grafanaIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnAlertFiringEventData | undefined;
      const status = eventData?.status || "firing";
      const alertName = getAlertName(eventData);
      const subtitle = buildSubtitle(status, lastEvent.createdAt);

      props.lastEventData = {
        title: alertName || "Grafana alert firing",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

interface OnAlertFiringMetadata {
  webhookUrl?: string;
  webhook_url?: string;
  url?: string;
}

const CopyWebhookUrlButton: FC<{ webhookUrl: string }> = ({ webhookUrl }) => {
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

export const onAlertFiringCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnAlertFiringMetadata | undefined;
    const webhookUrl =
      metadata?.webhookUrl || metadata?.webhook_url || metadata?.url || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Grafana Contact Point Setup</span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md">
              <ol className="list-decimal ml-4 space-y-1">
                <li>Save the canvas to generate the webhook URL.</li>
                <li>SuperPlane auto-provisions a Grafana webhook contact point in the background after save.</li>
                <li>If it is not created immediately, wait a moment and re-open the node.</li>
                <li>If provisioning still fails, create/update the contact point manually using the URL below.</li>
              </ol>
              <div className="mt-3">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-medium text-gray-700 dark:text-gray-200">Webhook URL</span>
                  <CopyWebhookUrlButton webhookUrl={webhookUrl} />
                </div>
                <pre className="mt-1 text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-white dark:bg-gray-900 rounded-md font-mono whitespace-pre-wrap break-all">
                  {webhookUrl}
                </pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  },
};

function getAlertName(eventData?: OnAlertFiringEventData): string | undefined {
  if (!eventData) return undefined;

  if (eventData.title && eventData.title.trim() !== "") {
    return eventData.title;
  }

  const commonLabel = eventData.commonLabels?.alertname;
  if (commonLabel && commonLabel.trim() !== "") {
    return commonLabel;
  }

  const firstAlert = eventData.alerts?.[0];
  const labelName = firstAlert?.labels?.alertname;
  if (labelName && labelName.trim() !== "") {
    return labelName;
  }

  return undefined;
}

function buildSubtitle(status: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "-";
  if (status) {
    return `${status} - ${timeAgo}`;
  }

  return timeAgo;
}
