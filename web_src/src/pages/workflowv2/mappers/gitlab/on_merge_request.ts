import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import { TriggerProps } from "@/ui/trigger";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { buildGitlabSubtitle } from "./utils";
import { GitLabNodeMetadata } from "./types";

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

export const onMergeRequestTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnMergeRequestEventData;
    const mr = eventData?.object_attributes;

    return {
      title: `#${mr?.iid ?? ""} - ${mr?.title || "Merge Request"}`,
      subtitle: buildGitlabSubtitle(mr?.action || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnMergeRequestEventData;
    const mr = eventData?.object_attributes;
    const values: Record<string, string> = {
      URL: mr?.url || "",
      Title: mr?.title || "",
      Action: mr?.action || "",
      State: mr?.state || "",
      IID: mr?.iid?.toString() || "",
    };

    if (eventData?.user?.username) {
      values.Author = eventData.user.username;
    }

    if (eventData?.project?.path_with_namespace) {
      values.Project = eventData.project.path_with_namespace;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnMergeRequestConfiguration;
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
      const eventData = lastEvent.data as OnMergeRequestEventData;
      const mr = eventData?.object_attributes;

      props.lastEventData = {
        title: `#${mr?.iid ?? ""} - ${mr?.title || "Merge Request"}`,
        subtitle: buildGitlabSubtitle(mr?.action || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
