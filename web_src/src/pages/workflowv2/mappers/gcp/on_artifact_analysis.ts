import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";
import gcpArtifactRegistryIcon from "@/assets/icons/integrations/gcp.artifactregistry.svg";
import { OccurrenceData } from "./artifact_registry";

export const onArtifactAnalysisTriggerRenderer: TriggerRenderer = {
  getEventState: () => "triggered",

  getTitleAndSubtitle: (_context: TriggerEventContext): { title: string; subtitle: string } => {
    return { title: "Container analysis event", subtitle: "" };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = context.event?.data as OccurrenceData | undefined;
    const details: Record<string, string> = {};

    if (context.event?.createdAt) {
      details["Received At"] = new Date(context.event.createdAt).toLocaleString();
    }

    if (data?.kind) {
      details["Kind"] = data.kind;
    }

    if (data?.resourceUri) {
      details["Resource URI"] = data.resourceUri;
    }

    if (data?.vulnerability?.severity) {
      details["Severity"] = data.vulnerability.severity;
    }

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const data = lastEvent?.data as OccurrenceData | undefined;
    return {
      title: node.name || definition.label || "On Artifact Analysis",
      iconSrc: gcpArtifactRegistryIcon,
      iconSlug: definition.icon || "cloud",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: data?.resourceUri
        ? [{ icon: "package", label: data.resourceUri }]
        : data?.resourceUri
          ? [{ icon: "package", label: data.resourceUri }]
          : [],
      ...(lastEvent && {
        lastEventData: {
          title: "Container analysis event",
          subtitle: formatTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
