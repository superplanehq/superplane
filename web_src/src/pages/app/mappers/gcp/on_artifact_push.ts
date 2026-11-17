import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";
import gcpArtifactRegistryIcon from "@/assets/icons/integrations/gcp.artifactregistry.svg";
import type { ArtifactPushData } from "./artifact_registry";
import { shortArtifactRef } from "./artifact_reference";

type OnArtifactPushConfiguration = {
  location?: string;
  repository?: string;
};

export const onArtifactPushTriggerRenderer: TriggerRenderer = {
  getEventState: () => "triggered",

  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const data = context.event?.data as ArtifactPushData | undefined;
    const imageRef = shortArtifactRef(data?.digest) ?? shortArtifactRef(data?.tag);
    const title = imageRef ? `Pushed ${imageRef}` : "Pushed artifact image";

    const subtitleParts: (string | React.ReactNode)[] = [];
    if (data?.action) {
      subtitleParts.push(formatPushAction(data.action));
    }
    if (context.event?.createdAt) {
      subtitleParts.push(renderTimeAgo(new Date(context.event.createdAt)));
    }

    return { title, subtitle: subtitleParts.join(" · ") };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = context.event?.data as ArtifactPushData | undefined;
    const details: Record<string, string> = {};

    if (context.event?.createdAt) {
      details["Received At"] = new Date(context.event.createdAt).toLocaleString();
    }

    if (data?.action) {
      details["Action"] = formatPushAction(data.action);
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
    const configuration = node.configuration as OnArtifactPushConfiguration | undefined;
    const metadata = buildConfigurationMetadata(configuration);
    const eventTitleAndSubtitle = lastEvent
      ? onArtifactPushTriggerRenderer.getTitleAndSubtitle({ event: lastEvent })
      : undefined;

    return {
      title: node.name || definition.label || "On Artifact Push",
      iconSrc: gcpArtifactRegistryIcon,
      iconSlug: definition.icon || "cloud",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata,
      ...(lastEvent && {
        lastEventData: {
          title: eventTitleAndSubtitle?.title ?? "Artifact push event",
          subtitle: eventTitleAndSubtitle?.subtitle ?? renderTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};

function buildConfigurationMetadata(configuration?: OnArtifactPushConfiguration): MetadataItem[] {
  return [
    { icon: "map-pin", label: configuration?.location || "All locations" },
    { icon: "folder", label: configuration?.repository || "All repositories" },
  ];
}

function formatPushAction(action: string): string {
  switch (action.toUpperCase()) {
    case "INSERT":
      return "Pushed";
    default:
      return action;
  }
}
