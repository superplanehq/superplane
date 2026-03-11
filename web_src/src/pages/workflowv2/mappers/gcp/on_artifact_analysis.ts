import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import gcpArtifactRegistryIcon from "@/assets/icons/integrations/gcp.artifactregistry.svg";
import { OccurrenceData } from "./artifact_registry";

type OnArtifactAnalysisConfiguration = {
  kinds?: string[];
  location?: string;
  repository?: string;
  package?: string;
};

export const onArtifactAnalysisTriggerRenderer: TriggerRenderer = {
  getEventState: () => "triggered",

  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const data = context.event?.data as OccurrenceData | undefined;
    const kindLabel = getKindLabel(data?.kind);
    const imageRef = shortArtifactRef(data?.resourceUri);
    const title = imageRef ? `${kindLabel}: ${imageRef}` : `${kindLabel} event`;

    const subtitleParts: string[] = [];
    if (data?.vulnerability?.severity) {
      subtitleParts.push(`${data.vulnerability.severity} severity`);
    }
    if (context.event?.createdAt) {
      subtitleParts.push(formatTimeAgo(new Date(context.event.createdAt)));
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
          subtitle: eventTitleAndSubtitle?.subtitle ?? formatTimeAgo(new Date(lastEvent.createdAt)),
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

function shortArtifactRef(uri?: string): string | undefined {
  if (!uri) {
    return undefined;
  }

  let normalized = uri.trim();
  normalized = normalized.replace(/^https?:\/\//, "");

  const slashIndex = normalized.indexOf("/");
  if (slashIndex < 0 || slashIndex === normalized.length - 1) {
    return undefined;
  }

  const path = normalized.slice(slashIndex + 1);
  const pathParts = path.split("/");
  if (pathParts.length < 3) {
    return path;
  }

  const imageWithRef = pathParts.slice(2).join("/");
  let imagePath = imageWithRef;
  let ref = "";
  let refSeparator = "";

  const atIndex = imageWithRef.indexOf("@");
  if (atIndex >= 0) {
    imagePath = imageWithRef.slice(0, atIndex);
    ref = imageWithRef.slice(atIndex + 1);
    refSeparator = "@";
  } else {
    const lastSlash = imageWithRef.lastIndexOf("/");
    const lastColon = imageWithRef.lastIndexOf(":");
    if (lastColon > lastSlash) {
      imagePath = imageWithRef.slice(0, lastColon);
      ref = imageWithRef.slice(lastColon + 1);
      refSeparator = ":";
    }
  }

  const imageName = imagePath.split("/").pop() || imagePath;
  const compactBase = imageName;
  if (!ref) {
    return compactBase || path;
  }

  if (refSeparator === "@" && ref.startsWith("sha256:")) {
    const digest = ref.slice("sha256:".length);
    return `${compactBase}@${digest.slice(0, 8)}`;
  }

  if (ref.length > 12) {
    ref = `${ref.slice(0, 12)}...`;
  }

  return `${compactBase}${refSeparator}${ref}`;
}
