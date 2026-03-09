import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsCodeBuildIcon from "@/assets/icons/integrations/aws.codebuild.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { formatPredicate, Predicate, stringOrDash } from "../../utils";

interface OnBuildConfiguration {
  region?: string;
  projects?: Predicate[];
  states?: string[];
}

interface BuildStateChangeEvent {
  account?: string;
  time?: string;
  region?: string;
  detail?: {
    "project-name"?: string;
    "build-status"?: string;
    "build-id"?: string;
    "current-phase"?: string;
  };
}

function buildMetadataItems(configuration?: OnBuildConfiguration): MetadataItem[] {
  const items: MetadataItem[] = [];

  if (configuration?.region) {
    items.push({ icon: "globe", label: configuration.region });
  }

  if (configuration?.projects && configuration.projects.length > 0) {
    items.push({
      icon: "funnel",
      label: configuration.projects.map(formatPredicate).join(", "),
    });
  }

  if (configuration?.states && configuration.states.length > 0) {
    items.push({ icon: "tag", label: configuration.states.join(", ") });
  }

  return items;
}

export const onBuildTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as BuildStateChangeEvent;
    const project = eventData?.detail?.["project-name"];
    const status = eventData?.detail?.["build-status"];

    let title = "CodeBuild build";
    if (project && status) {
      title = `${project} - ${status}`;
    } else if (project) {
      title = project;
    }

    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as BuildStateChangeEvent;
    const detail = eventData?.detail;

    return {
      Project: stringOrDash(detail?.["project-name"]),
      Status: stringOrDash(detail?.["build-status"]),
      "Build ID": stringOrDash(detail?.["build-id"]),
      Phase: stringOrDash(detail?.["current-phase"]),
      Timestamp: stringOrDash(eventData?.time),
      Region: stringOrDash(eventData?.region),
      Account: stringOrDash(eventData?.account),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnBuildConfiguration | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsCodeBuildIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadataItems(configuration),
    };

    if (lastEvent) {
      const { title, subtitle } = onBuildTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
