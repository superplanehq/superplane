import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { CustomFieldRenderer, NodeInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";
import harnessIcon from "@/assets/icons/integrations/harness.svg";

interface OnPipelineCompletedEventData {
  eventData?: {
    pipelineName: string;
    pipelineIdentifier: string;
    projectIdentifier: string;
    orgIdentifier: string;
    eventType: string;
    executionUrl: string;
    planExecutionId: string;
    nodeStatus: string;
    triggeredBy?: {
      triggerType: string;
      name: string;
      email: string;
    };
    startTime?: string;
    endTime?: string;
    startTs?: number;
    endTs?: number;
  };
}

interface OnPipelineCompletedMetadata {
  webhookUrl?: string;
}

export const onPipelineCompletedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnPipelineCompletedEventData;
    const pipelineName = eventData?.eventData?.pipelineName || "";
    const eventType = eventData?.eventData?.eventType || "";
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt)) : "";
    const subtitle = eventType && timeAgo ? `${eventType} · ${timeAgo}` : eventType || timeAgo;

    return {
      title: pipelineName,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnPipelineCompletedEventData;
    const data = eventData?.eventData;

    return {
      Pipeline: data?.pipelineName || "",
      "Event Type": data?.eventType || "",
      Project: data?.projectIdentifier || "",
      Organization: data?.orgIdentifier || "",
      "Execution URL": data?.executionUrl || "",
      "Triggered By": data?.triggeredBy?.name || "",
      "Start Time": data?.startTime || "",
      "End Time": data?.endTime || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadataItems = [];

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: harnessIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnPipelineCompletedEventData;
      const pipelineName = eventData?.eventData?.pipelineName || "";
      const eventType = eventData?.eventData?.eventType || "";
      const timeAgo = lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "";
      const subtitle = eventType && timeAgo ? `${eventType} · ${timeAgo}` : eventType || timeAgo;

      props.lastEventData = {
        title: pipelineName,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

export const onPipelineCompletedCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnPipelineCompletedMetadata | undefined;
    const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Harness Webhook Setup</span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md">
              <ol className="list-decimal ml-4 space-y-1">
                <li>Save the canvas to generate the webhook URL.</li>
                <li>In Harness, go to <strong>Project Settings &gt; Notifications</strong>.</li>
                <li>Create a new notification with <strong>Webhook</strong> as the channel.</li>
                <li>Paste the webhook URL below and select pipeline events (e.g. Pipeline Success, Pipeline Failed).</li>
              </ol>
              <div className="mt-3">
                <span className="text-xs font-medium text-gray-700 dark:text-gray-200">Webhook URL</span>
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
