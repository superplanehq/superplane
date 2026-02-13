import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import { TriggerProps } from "@/ui/trigger";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { Predicate, formatPredicate, stringOrDash } from "../utils";
import { buildGitlabSubtitle } from "./utils";
import { GitLabNodeMetadata } from "./types";

interface OnTagConfiguration {
  tags: Predicate[];
}

interface OnTagEventData {
  object_kind?: string;
  event_name?: string;
  ref?: string;
  before?: string;
  after?: string;
  user_name?: string;
  project?: {
    id: number;
    name: string;
    path_with_namespace: string;
    web_url: string;
  };
}

export const onTagTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnTagEventData;

    return {
      title: eventData?.ref ? eventData.ref : "Tag Push",
      subtitle: buildGitlabSubtitle(eventData?.event_name || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnTagEventData;
    const values: Record<string, string> = {
      Ref: stringOrDash(eventData?.ref),
      Before: stringOrDash(eventData?.before),
      After: stringOrDash(eventData?.after),
    };

    if (eventData?.user_name) {
      values.Author = eventData.user_name;
    }

    if (eventData?.project?.path_with_namespace) {
      values.Project = eventData.project.path_with_namespace;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnTagConfiguration;
    const metadataItems = [];

    if (metadata?.project?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.project.name,
      });
    }

    if (configuration?.tags?.length > 0) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.tags.map((tag) => formatPredicate(tag)).join(", "),
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
      const eventData = lastEvent.data as OnTagEventData;

      props.lastEventData = {
        title: eventData?.ref ? eventData.ref : "Tag Push",
        subtitle: buildGitlabSubtitle(eventData?.event_name || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
