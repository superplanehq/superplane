import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import { TriggerProps } from "@/ui/trigger";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { buildGitlabSubtitle } from "./utils";
import { GitLabNodeMetadata } from "./types";

interface OnReleaseConfiguration {
  actions: string[];
}

interface OnReleaseEventData {
  id?: number;
  object_kind?: string;
  action?: string;
  name?: string;
  tag?: string;
  url?: string;
  project?: {
    id: number;
    name: string;
    path_with_namespace: string;
    web_url: string;
  };
}

function getReleaseTitle(eventData: OnReleaseEventData): string {
  const releaseName = eventData?.name || eventData?.tag || "Release";
  if (eventData?.tag) {
    return `${releaseName} (${eventData.tag})`;
  }

  return releaseName;
}

export const onReleaseTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnReleaseEventData;

    return {
      title: getReleaseTitle(eventData),
      subtitle: buildGitlabSubtitle(eventData?.action || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnReleaseEventData;
    const values: Record<string, string> = {
      Name: eventData?.name || "",
      Tag: eventData?.tag || "",
      Action: eventData?.action || "",
      URL: eventData?.url || "",
    };

    if (eventData?.project?.path_with_namespace) {
      values.Project = eventData.project.path_with_namespace;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnReleaseConfiguration;
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

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: gitlabIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnReleaseEventData;

      props.lastEventData = {
        title: getReleaseTitle(eventData),
        subtitle: buildGitlabSubtitle(eventData?.action || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
