import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsIcon from "@/assets/icons/integrations/aws.svg";
import { CodeBuildBuildEvent, CodeBuildTriggerConfiguration, CodeBuildTriggerMetadata } from "./types";
import { buildProjectMetadataItems, getProjectLabel } from "./utils";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../../utils";

/**
 * Renderer for the "aws.codebuild.onBuild" trigger
 */
export const onBuildTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = getBuildEvent(context);
    const detail = eventData?.detail;
    const project = getProjectLabel(undefined, undefined, detail?.["project-name"]);
    const status = detail?.["build-status"];

    const title = project ? `${project}${status ? ` · ${status}` : ""}` : "CodeBuild build";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt || "")) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = getBuildEvent(context);
    const detail = eventData?.detail;

    return {
      Project: stringOrDash(getProjectLabel(undefined, undefined, detail?.["project-name"])),
      "Build Status": stringOrDash(detail?.["build-status"]),
      "Build ID": stringOrDash(detail?.["build-id"]),
      "Current Phase": stringOrDash(detail?.["current-phase"]),
      "Phase Context": stringOrDash(detail?.["current-phase-context"]),
      "Source Version": stringOrDash(detail?.["additional-information"]?.["source-version"]),
      Initiator: stringOrDash(detail?.["additional-information"]?.initiator),
      "Logs Link": stringOrDash(detail?.["additional-information"]?.logs?.["deep-link"]),
      Region: stringOrDash(eventData?.region),
      Account: stringOrDash(eventData?.account),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as CodeBuildTriggerMetadata | undefined;
    const configuration = node.configuration as CodeBuildTriggerConfiguration | undefined;
    const metadataItems = buildProjectMetadataItems(metadata, configuration);

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
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

function getBuildEvent(context: TriggerEventContext): CodeBuildBuildEvent {
  const eventData = context.event?.data as CodeBuildBuildEvent | { data?: CodeBuildBuildEvent } | undefined;
  if (!eventData) {
    return {};
  }

  if (typeof eventData === "object" && "data" in eventData && eventData.data) {
    return eventData.data;
  }

  return eventData;
}
