import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/pages/workflowv2/mappers/rendererTypes";
import { renderTimeAgo } from "@/components/TimeAgo";
import cloudBuildIcon from "@/assets/icons/integrations/cloud_build.svg";
import { buildCloudBuildSummaryDetails, cloudBuildStatusToTriggerState, type CloudBuildData } from "./cloudbuild";

export const onBuildCompleteTriggerRenderer: TriggerRenderer = {
  getEventState: (context: TriggerEventContext) => {
    const data = context.event?.data as CloudBuildData | undefined;
    return cloudBuildStatusToTriggerState(data?.status);
  },

  subtitle: (_context: TriggerEventContext): string | React.ReactNode => {
    return "";
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return buildCloudBuildSummaryDetails({
      build: context.event?.data as CloudBuildData | undefined,
      receivedAt: context.event?.createdAt,
    });
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const data = lastEvent?.data as CloudBuildData | undefined;
    return {
      title: node.name || definition.label || "On Build Complete",
      iconSrc: cloudBuildIcon,
      iconSlug: definition.icon || "cloud",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: [],
      ...(lastEvent && {
        lastEventData: {
          subtitle: renderTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: cloudBuildStatusToTriggerState(data?.status),
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
