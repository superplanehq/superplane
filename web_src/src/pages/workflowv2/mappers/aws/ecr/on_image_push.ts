import { getBackgroundColorClass } from "@/utils/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import type { TriggerProps } from "@/ui/trigger";
import awsEcrIcon from "@/assets/icons/integrations/aws.ecr.svg";
import type { EcrImagePushEvent, EcrTriggerConfiguration, EcrTriggerMetadata } from "./types";
import { buildRepositoryMetadataItems, getRepositoryLabel } from "./utils";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../../utils";

/**
 * Renderer for the "aws.ecr.onImagePush" trigger
 */
export const onImagePushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as EcrImagePushEvent;
    const detail = eventData?.detail;
    const repository = getRepositoryLabel(undefined, undefined, detail?.["repository-name"]);
    const tag = detail?.["image-tag"];

    const title = repository ? `${repository}${tag ? `:${tag}` : ""}` : "ECR image push";
    const subtitle = context.event?.createdAt ? renderTimeAgo(new Date(context.event?.createdAt || "")) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as EcrImagePushEvent;
    const detail = eventData?.detail;

    return {
      Repository: stringOrDash(getRepositoryLabel(undefined, undefined, detail?.["repository-name"])),
      "Image Tag": stringOrDash(detail?.["image-tag"]),
      "Image Digest": stringOrDash(detail?.["image-digest"]),
      Action: stringOrDash(detail?.["action-type"]),
      Result: stringOrDash(detail?.result),
      Region: stringOrDash(eventData?.region),
      Account: stringOrDash(eventData?.account),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as EcrTriggerMetadata | undefined;
    const configuration = node.configuration as EcrTriggerConfiguration | undefined;
    const metadataItems = buildRepositoryMetadataItems(metadata, configuration);

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsEcrIcon,
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
