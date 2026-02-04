import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, Release } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnReleaseConfiguration {
  actions: string[];
}

interface OnReleaseEventData {
  action?: string;
  release?: Release;
}

/**
 * Renderer for the "github.onRelease" trigger
 */
export const onReleaseTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnReleaseEventData;
    const assetCount = eventData?.release?.assets?.length || 0;
    const releaseName = eventData?.release?.name || eventData?.release?.tag_name || "Release";

    return {
      title: `${releaseName} (${assetCount} asset${assetCount !== 1 ? "s" : ""})`,
      subtitle: buildGithubSubtitle(eventData?.action || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnReleaseEventData;
    const values: Record<string, string> = {
      Name: eventData?.release?.name || "",
      Tag: eventData?.release?.tag_name || "",
      Action: eventData?.action || "",
      Author: eventData?.release?.author?.login || "",
      Prerelease: eventData?.release?.prerelease ? "true" : "false",
    };

    if (eventData?.action !== "deleted") {
      values.URL = eventData?.release?.html_url || "";
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnReleaseConfiguration;
    const metadataItems = [];

    if (metadata?.repository?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.name,
      });
    }

    if (configuration?.actions) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.actions.join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnReleaseEventData;
      const assetCount = eventData?.release?.assets?.length || 0;
      const releaseName = eventData?.release?.name || eventData?.release?.tag_name || "Release";

      props.lastEventData = {
        title: `${releaseName} (${assetCount} asset${assetCount !== 1 ? "s" : ""})`,
        subtitle: buildGithubSubtitle(eventData?.action || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
