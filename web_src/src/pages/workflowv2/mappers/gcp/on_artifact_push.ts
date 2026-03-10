import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";
import gcpArtifactRegistryIcon from "@/assets/icons/integrations/gcp.artifactregistry.svg";
import { ArtifactPushData } from "./artifact_registry";

export const onArtifactPushTriggerRenderer: TriggerRenderer = {
  getEventState: () => "triggered",

  getTitleAndSubtitle: (_context: TriggerEventContext): { title: string; subtitle: string } => {
    return { title: "Artifact push event", subtitle: "" };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = context.event?.data as ArtifactPushData | undefined;
    const details: Record<string, string> = {};

    if (context.event?.createdAt) {
      details["Received At"] = new Date(context.event.createdAt).toLocaleString();
    }

    if (data?.action) {
      details["Action"] = data.action;
    }

    if (data?.digest) {
      details["Image (Digest)"] = data.digest;
    }

    if (data?.tag) {
      details["Image (Tag)"] = data.tag;
    }

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const data = lastEvent?.data as ArtifactPushData | undefined;
    return {
      title: node.name || definition.label || "On Artifact Push",
      iconSrc: gcpArtifactRegistryIcon,
      iconSlug: definition.icon || "cloud",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: data?.digest
        ? [{ icon: "package", label: data.digest }]
        : data?.tag
          ? [{ icon: "package", label: data.tag }]
          : [],
      ...(lastEvent && {
        lastEventData: {
          title: "Artifact push event",
          subtitle: formatTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
