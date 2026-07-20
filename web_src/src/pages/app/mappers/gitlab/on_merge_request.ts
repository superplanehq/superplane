import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { buildGitlabSubtitle } from "./utils";
import type { GitLabNodeMetadata } from "./types";

interface OnMergeRequestConfiguration {
  actions: string[];
}

interface MergeRequestObjectAttributes {
  id?: number;
  iid?: number;
  title?: string;
  description?: string;
  state?: string;
  action?: string;
  url?: string;
}

interface OnMergeRequestEventData {
  object_kind?: string;
  event_type?: string;
  object_attributes?: MergeRequestObjectAttributes;
  user?: {
    id: number;
    name: string;
    username: string;
  };
  project?: {
    id: number;
    name: string;
    path_with_namespace: string;
    web_url: string;
  };
}

function formatReceivedAt(createdAt?: string): string {
  return createdAt ? new Date(createdAt).toLocaleString() : "-";
}

function mergeRequestEventTitle(eventData?: OnMergeRequestEventData): string {
  const mr = eventData?.object_attributes;
  return `!${mr?.iid ?? ""} - ${mr?.title || "Merge Request"}`;
}

function mergeRequestEventSubtitle(eventData?: OnMergeRequestEventData, createdAt?: string) {
  return buildGitlabSubtitle(eventData?.object_attributes?.action || "", createdAt);
}

function buildMetadataItems(metadata?: GitLabNodeMetadata, configuration?: OnMergeRequestConfiguration) {
  const metadataItems = [];

  if (metadata?.project?.name) {
    metadataItems.push({
      icon: "book",
      label: metadata.project.name,
    });
  }

  if (configuration?.actions) {
    metadataItems.push({
      icon: "funnel",
      label: configuration.actions.join(", "),
    });
  }

  return metadataItems;
}

export const onMergeRequestTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnMergeRequestEventData;

    return {
      title: mergeRequestEventTitle(eventData),
      subtitle: mergeRequestEventSubtitle(eventData, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnMergeRequestEventData;
    const mr = eventData?.object_attributes;

    return {
      "Received At": formatReceivedAt(context.event?.createdAt),
      Title: mr?.title || "-",
      URL: mr?.url || "-",
      Action: mr?.action || "-",
      State: mr?.state || "-",
      Author: eventData?.user?.username || "-",
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnMergeRequestConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: gitlabIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnMergeRequestEventData;

      props.lastEventData = {
        title: mergeRequestEventTitle(eventData),
        subtitle: mergeRequestEventSubtitle(eventData, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
