import { getBackgroundColorClass } from "@/utils/colors";
import { CustomFieldRenderer, NodeInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import dockerIcon from "@/assets/icons/integrations/docker.svg";
import { DockerHubImagePushEvent, DockerHubTriggerConfiguration, DockerHubTriggerMetadata } from "./types";
import { buildRepositoryMetadataItems, getRepositoryLabel } from "./utils";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { formatPredicate, stringOrDash } from "../../utils";
import { CopyCodeButton } from "@/components/CopyCodeButton";

/**
 * Renderer for the "dockerhub.onImagePush" trigger
 */
export const onImagePushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as DockerHubImagePushEvent;
    const repository = getRepositoryLabel(undefined, undefined, eventData?.repository?.repo_name);
    const tag = eventData?.push_data?.tag;

    const title = repository ? `${repository}${tag ? `:${tag}` : ""}` : "Docker Hub image push";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt || "")) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as DockerHubImagePushEvent;
    const repository = eventData?.repository;
    const pushData = eventData?.push_data;
    const pushedAt = pushData?.pushed_at ? new Date(pushData.pushed_at * 1000).toISOString() : undefined;

    const visibility =
      repository?.is_private === undefined ? "-" : repository.is_private ? "Private" : "Public";

    return {
      Repository: stringOrDash(getRepositoryLabel(undefined, undefined, repository?.repo_name)),
      Tag: stringOrDash(pushData?.tag),
      Pusher: stringOrDash(pushData?.pusher),
      "Pushed At": pushedAt ? formatTimestampInUserTimezone(pushedAt) : "-",
      "Repository URL": stringOrDash(repository?.repo_url),
      Visibility: visibility,
      Stars: stringOrDash(repository?.star_count),
      Pulls: stringOrDash(repository?.pull_count),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as DockerHubTriggerMetadata | undefined;
    const configuration = node.configuration as DockerHubTriggerConfiguration | undefined;
    const metadataItems = buildRepositoryMetadataItems(metadata, configuration);

    if (configuration?.tags?.length) {
      metadataItems.push({
        icon: "tag",
        label: configuration.tags.map(formatPredicate).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: dockerIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onImagePushTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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

export const onImagePushCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as DockerHubTriggerMetadata | undefined;
    const configuration = node.configuration as DockerHubTriggerConfiguration | undefined;
    const repositoryLabel = getRepositoryLabel(metadata, configuration) || "<namespace>/<repository>";
    const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Docker Hub Webhook Setup</span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md">
              <ol className="list-decimal ml-4 space-y-1">
                <li>Open the Docker Hub repository: {repositoryLabel}</li>
                <li>Go to <strong>Webhooks</strong> and click <strong>Add webhook</strong></li>
                <li>Set the webhook URL below and save</li>
              </ol>
              <div className="mt-3">
                <span className="text-xs font-medium text-gray-700 dark:text-gray-200">Webhook URL</span>
                <div className="relative group mt-1">
                  <pre className="text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-white dark:bg-gray-900 rounded-md font-mono whitespace-pre-wrap break-all">
                    {webhookUrl}
                  </pre>
                  {metadata?.webhookUrl && <CopyCodeButton code={webhookUrl} />}
                </div>
              </div>
              <p className="mt-3">
                Docker Hub will send tag push events to SuperPlane once the webhook is configured.
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  },
};
