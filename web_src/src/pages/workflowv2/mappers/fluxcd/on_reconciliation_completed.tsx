import { useState, type FC } from "react";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { CustomFieldRenderer, NodeInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import fluxcdIcon from "@/assets/icons/integrations/fluxcd.svg";
import { FluxReconciliationEvent } from "./types";
import { stringOrDash } from "../utils";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";

export const onReconciliationCompletedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as FluxReconciliationEvent | undefined;
    const obj = eventData?.involvedObject;
    const title = obj ? `${obj.kind}/${obj.name}` : "Reconciliation completed";
    const subtitle = buildSubtitle(eventData?.reason, context.event?.createdAt);

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as FluxReconciliationEvent | undefined;
    const obj = eventData?.involvedObject;

    return {
      "Received At": context.event?.createdAt ? new Date(context.event.createdAt).toLocaleString() : "-",
      Kind: stringOrDash(obj?.kind),
      Name: stringOrDash(obj?.name),
      Namespace: stringOrDash(obj?.namespace),
      Reason: stringOrDash(eventData?.reason),
      Severity: stringOrDash(eventData?.severity),
      Message: stringOrDash(eventData?.message),
      Revision: stringOrDash(eventData?.metadata?.revision),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadataItems = [];

    if (lastEvent?.data) {
      const eventData = lastEvent.data as FluxReconciliationEvent;
      const obj = eventData.involvedObject;
      if (obj?.kind) {
        metadataItems.push({
          icon: "box",
          label: `${obj.kind}/${obj.name || ""}`,
        });
      }
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: fluxcdIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as FluxReconciliationEvent | undefined;
      const obj = eventData?.involvedObject;
      const title = obj ? `${obj.kind}/${obj.name}` : "Reconciliation completed";
      const subtitle = buildSubtitle(eventData?.reason, lastEvent.createdAt);

      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

interface OnReconciliationCompletedMetadata {
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

export const onReconciliationCompletedCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnReconciliationCompletedMetadata | undefined;
    const webhookUrl =
      metadata?.webhookUrl || metadata?.webhook_url || metadata?.url || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
              FluxCD Notification Provider Setup
            </span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md">
              <ol className="list-decimal ml-4 space-y-1">
                <li>Save the canvas to generate the webhook URL.</li>
                <li>
                  Create a FluxCD Notification Provider of type <code>generic</code> pointing to the URL below.
                </li>
                <li>Create a FluxCD Alert referencing the provider and the resources you want to monitor.</li>
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

function buildSubtitle(reason?: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "-";
  if (reason) {
    return `${reason} - ${timeAgo}`;
  }
  return timeAgo;
}
