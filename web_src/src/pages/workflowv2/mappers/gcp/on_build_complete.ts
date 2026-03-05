import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";
import cloudBuildIcon from "@/assets/icons/integrations/cloud_build.svg";
import { buildCloudBuildSummaryDetails, cloudBuildStatusToTriggerState, type CloudBuildData } from "./cloudbuild";

export const onBuildCompleteTriggerRenderer: TriggerRenderer = {
  getEventState: (context: TriggerEventContext) => {
    const data = context.event?.data as CloudBuildData | undefined;
    return cloudBuildStatusToTriggerState(data?.status);
  },

  getTitleAndSubtitle: (_context: TriggerEventContext): { title: string; subtitle: string } => {
    const title = "Cloud Build event";
    return { title, subtitle: "" };
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
          title: "Cloud Build event",
          subtitle: formatTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: cloudBuildStatusToTriggerState(data?.status),
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
