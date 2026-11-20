import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";
import gcpArtifactRegistryIcon from "@/assets/icons/integrations/gcp.artifactregistry.svg";
import type { OccurrenceData } from "./artifact_registry";
import { shortArtifactRef } from "./artifact_reference";

type OnArtifactAnalysisConfiguration = {
  kinds?: string[];
  location?: string;
  repository?: string;
  package?: string;
};

export const onArtifactAnalysisTriggerRenderer: TriggerRenderer = {
  getEventState: () => "triggered",

  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const data = context.event?.data as OccurrenceData | undefined;
    const kindLabel = getKindLabel(data?.kind);
    const imageRef = shortArtifactRef(data?.resourceUri);
    const title = imageRef ? `${kindLabel}: ${imageRef}` : `${kindLabel} event`;

    const subtitleParts: (string | React.ReactNode)[] = [];
    if (data?.vulnerability?.severity) {
      subtitleParts.push(`${data.vulnerability.severity} severity`);
    }
    if (context.event?.createdAt) {
      subtitleParts.push(renderTimeAgo(new Date(context.event.createdAt)));
    }

    return { title, subtitle: subtitleParts.join(" · ") };
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
    const configuration = node.configuration as OnArtifactAnalysisConfiguration | undefined;
    const metadata = buildConfigurationMetadata(configuration);
    const eventTitleAndSubtitle = lastEvent
      ? onArtifactAnalysisTriggerRenderer.getTitleAndSubtitle({ event: lastEvent })
      : undefined;

    return {
      title: node.name || definition.label || "On Artifact Analysis",
      iconSrc: gcpArtifactRegistryIcon,
      iconSlug: definition.icon || "cloud",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata,
      ...(lastEvent && {
        lastEventData: {
          title: eventTitleAndSubtitle?.title ?? "Container analysis event",
          subtitle: eventTitleAndSubtitle?.subtitle ?? renderTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};

function buildConfigurationMetadata(configuration?: OnArtifactAnalysisConfiguration): MetadataItem[] {
  const kinds = configuration?.kinds?.filter(Boolean) ?? [];
  const scope = [configuration?.location || "All locations", configuration?.repository || "All repositories"]
    .filter(Boolean)
    .join(" / ");
  const packageScope = configuration?.package || "All packages";

  return [
    { icon: "funnel", label: kinds.length > 0 ? kinds.join(", ") : "DISCOVERY (default)" },
    { icon: "package", label: `${scope} / ${packageScope}` },
  ];
}

function getKindLabel(kind?: string): string {
  switch (kind?.toUpperCase()) {
    case "DISCOVERY":
      return "Scan";
    case "VULNERABILITY":
      return "Vulnerability";
    case "BUILD":
      return "Build";
    case "ATTESTATION":
      return "Attestation";
    case "SBOM":
      return "SBOM";
    default:
      return "Analysis";
  }
}
