import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import { TriggerProps } from "@/ui/trigger";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { buildGitlabSubtitle } from "./utils";
import { GitLabNodeMetadata } from "./types";
import { stringOrDash } from "../utils";

interface OnMilestoneConfiguration {
  actions: string[];
}

interface MilestoneObjectAttributes {
  id?: number;
  iid?: number;
  title?: string;
  description?: string;
  state?: string;
  due_date?: string;
  start_date?: string;
}

interface OnMilestoneEventData {
  object_kind?: string;
  event_type?: string;
  action?: string;
  object_attributes?: MilestoneObjectAttributes;
  project?: {
    id: number;
    name: string;
    path_with_namespace: string;
    web_url: string;
  };
}

export const onMilestoneTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnMilestoneEventData;
    const milestone = eventData?.object_attributes;

    return {
      title: milestone?.title ? milestone.title : "Milestone",
      subtitle: buildGitlabSubtitle(eventData?.action || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnMilestoneEventData;
    const milestone = eventData?.object_attributes;
    const values: Record<string, string> = {
      Title: stringOrDash(milestone?.title),
      Action: stringOrDash(eventData?.action),
      State: stringOrDash(milestone?.state),
      IID: stringOrDash(milestone?.iid?.toString()),
      "Start Date": stringOrDash(milestone?.start_date),
      "Due Date": stringOrDash(milestone?.due_date),
    };

    if (eventData?.project?.path_with_namespace) {
      values.Project = eventData.project.path_with_namespace;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnMilestoneConfiguration;
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
      const eventData = lastEvent.data as OnMilestoneEventData;
      const milestone = eventData?.object_attributes;

      props.lastEventData = {
        title: milestone?.title ? milestone.title : "Milestone",
        subtitle: buildGitlabSubtitle(stringOrDash(eventData?.action), lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
