import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import { TriggerProps } from "@/ui/trigger";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { buildGitlabSubtitle } from "./utils";
import { GitLabNodeMetadata } from "./types";
import { stringOrDash } from "../utils";

interface OnPipelineConfiguration {
  statuses: string[];
}

interface PipelineObjectAttributes {
  id?: number;
  iid?: number;
  status?: string;
  ref?: string;
  sha?: string;
  url?: string;
}

interface OnPipelineEventData {
  object_kind?: string;
  object_attributes?: PipelineObjectAttributes;
  project?: {
    id: number;
    name: string;
    path_with_namespace: string;
    web_url: string;
  };
}

function getPipelineTitle(eventData: OnPipelineEventData): string {
  const attrs = eventData?.object_attributes;
  if (attrs?.iid) {
    return `Pipeline #${attrs.iid}`;
  }
  if (attrs?.id) {
    return `Pipeline #${attrs.id}`;
  }
  return "Pipeline";
}

export const onPipelineTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnPipelineEventData;
    const attrs = eventData?.object_attributes;

    return {
      title: getPipelineTitle(eventData),
      subtitle: buildGitlabSubtitle(attrs?.status || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnPipelineEventData;
    const attrs = eventData?.object_attributes;
    const values: Record<string, string> = {
      ID: stringOrDash(attrs?.id?.toString()),
      IID: stringOrDash(attrs?.iid?.toString()),
      Status: stringOrDash(attrs?.status),
      Ref: stringOrDash(attrs?.ref),
      SHA: stringOrDash(attrs?.sha),
      URL: stringOrDash(attrs?.url),
    };

    if (eventData?.project?.path_with_namespace) {
      values.Project = eventData.project.path_with_namespace;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnPipelineConfiguration;
    const metadataItems = [];

    if (metadata?.project?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.project.name,
      });
    }

    if (configuration?.statuses?.length > 0) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.statuses.join(", "),
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
      const eventData = lastEvent.data as OnPipelineEventData;
      const attrs = eventData?.object_attributes;

      props.lastEventData = {
        title: getPipelineTitle(eventData),
        subtitle: buildGitlabSubtitle(attrs?.status || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
