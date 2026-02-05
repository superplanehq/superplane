import { useState } from "react";
import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer, TriggerRendererContext, TriggerEventContext, CustomFieldRenderer, NodeInfo } from "../types";
import dockerIcon from "@/assets/icons/integrations/docker.svg";
import { TriggerProps } from "@/ui/trigger";
import { OnImagePushMetadata, OnImagePushConfiguration, WebhookPayload } from "./types";
import { formatTimeAgo } from "@/utils/date";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";

/**
 * Renderer for the "dockerhub.onImagePush" trigger
 */
export const onImagePushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data?.data as WebhookPayload;
    const tag = eventData?.push_data?.tag || "latest";
    const repoName = eventData?.repository?.repo_name || "";

    return {
      title: `${repoName}:${tag}`,
      subtitle: eventData?.push_data?.pusher ? `by ${eventData.push_data.pusher}` : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data?.data as WebhookPayload;

    return {
      Repository: eventData?.repository?.repo_name || "",
      Tag: eventData?.push_data?.tag || "",
      Pusher: eventData?.push_data?.pusher || "",
      Namespace: eventData?.repository?.namespace || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnImagePushMetadata;
    const configuration = node.configuration as unknown as OnImagePushConfiguration;
    const metadataItems = [];

    if (metadata?.repository?.fullName) {
      metadataItems.push({
        icon: "box",
        label: metadata.repository.fullName,
      });
    } else if (configuration?.namespace && configuration?.repository) {
      metadataItems.push({
        icon: "box",
        label: `${configuration.namespace}/${configuration.repository}`,
      });
    }

    if (configuration?.tags && configuration.tags.length > 0) {
      const tagFilters = configuration.tags
        .map((t) => t.value)
        .filter(Boolean)
        .join(", ");
      if (tagFilters) {
        metadataItems.push({
          icon: "tag",
          label: `Tags: ${tagFilters}`,
        });
      }
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Docker Hub Push",
      iconSrc: dockerIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as WebhookPayload;
      const tag = eventData?.push_data?.tag || "latest";
      const repoName = eventData?.repository?.repo_name || "";

      props.lastEventData = {
        title: `${repoName}:${tag}`,
        subtitle: eventData?.push_data?.pusher
          ? `by ${eventData.push_data.pusher} ${formatTimeAgo(new Date(lastEvent.createdAt!))}`
          : formatTimeAgo(new Date(lastEvent.createdAt!)),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};

/**
 * Copy button component for code blocks
 */
const CopyCodeButton: React.FC<{ code: string }> = ({ code }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (_err) {
      showErrorToast("Failed to copy to clipboard");
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity p-1 bg-white outline-1 outline-black/20 hover:outline-black/30 rounded text-gray-600 dark:text-gray-400"
      title={copied ? "Copied!" : "Copy to clipboard"}
    >
      <Icon name={copied ? "check" : "copy"} size="sm" />
    </button>
  );
};

/**
 * Custom field renderer for Docker Hub webhook URL display
 */
export const onImagePushCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnImagePushMetadata | undefined;
    const configuration = node.configuration as OnImagePushConfiguration | undefined;
    const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    const repoDisplay =
      metadata?.repository?.fullName ||
      (configuration?.namespace && configuration?.repository
        ? `${configuration.namespace}/${configuration.repository}`
        : "your repository");

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Webhook Configuration</span>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Docker Hub webhooks must be manually configured. Copy the URL below and add it to your repository settings
              in Docker Hub.
            </p>

            {/* Webhook URL Copy Field */}
            <div className="mt-3">
              <label className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                Webhook URL
              </label>
              <div className="relative group mt-1">
                <input
                  type="text"
                  value={webhookUrl}
                  readOnly
                  className="w-full text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md font-mono"
                />
                <CopyCodeButton code={webhookUrl} />
              </div>
            </div>

            {/* Instructions */}
            <div className="mt-4 p-3 bg-gray-50 dark:bg-gray-800 rounded-md">
              <div className="flex items-start gap-2">
                <Icon name="info" className="w-4 h-4 text-blue-500 mt-0.5" />
                <div className="text-sm">
                  <p className="font-medium text-gray-700 dark:text-gray-300">Setup Instructions:</p>
                  <ol className="list-decimal list-inside mt-1 text-gray-600 dark:text-gray-400 space-y-1">
                    <li>
                      Go to{" "}
                      <a
                        href={`https://hub.docker.com/repository/docker/${repoDisplay}/webhooks`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-blue-500 hover:underline"
                      >
                        {repoDisplay} webhook settings
                      </a>
                    </li>
                    <li>Click "Create Webhook"</li>
                    <li>Enter a name and paste the webhook URL above</li>
                    <li>Save the webhook</li>
                  </ol>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  },
};
