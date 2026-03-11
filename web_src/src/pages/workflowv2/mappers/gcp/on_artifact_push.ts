import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import gcpArtifactRegistryIcon from "@/assets/icons/integrations/gcp.artifactregistry.svg";
import { ArtifactPushData } from "./artifact_registry";

type OnArtifactPushConfiguration = {
  location?: string;
  repository?: string;
};

export const onArtifactPushTriggerRenderer: TriggerRenderer = {
  getEventState: () => "triggered",

  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const data = context.event?.data as ArtifactPushData | undefined;
    const imageRef = shortArtifactRef(data?.digest) ?? shortArtifactRef(data?.tag);
    const title = imageRef ? `Pushed ${imageRef}` : "Pushed artifact image";

    const subtitleParts: string[] = [];
    if (data?.action) {
      subtitleParts.push(formatPushAction(data.action));
    }
    if (context.event?.createdAt) {
      subtitleParts.push(formatTimeAgo(new Date(context.event.createdAt)));
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
          subtitle: eventTitleAndSubtitle?.subtitle ?? formatTimeAgo(new Date(lastEvent.createdAt)),
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
  if (!ref) {
    return imageName || path;
  }

  if (refSeparator === "@" && ref.startsWith("sha256:")) {
    const digest = ref.slice("sha256:".length);
    return `${imageName}@${digest.slice(0, 8)}`;
  }

  if (ref.length > 12) {
    ref = `${ref.slice(0, 12)}...`;
  }

  return `${imageName}${refSeparator}${ref}`;
}

function formatPushAction(action: string): string {
  switch (action.toUpperCase()) {
    case "INSERT":
      return "Pushed";
    default:
      return action;
  }
}
