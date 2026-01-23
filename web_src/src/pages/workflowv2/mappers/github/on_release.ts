import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
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
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnReleaseEventData;
    const assetCount = eventData?.release?.assets?.length || 0;
    const releaseName = eventData?.release?.name || eventData?.release?.tag_name || "Release";

    return {
      title: `${releaseName} (${assetCount} asset${assetCount !== 1 ? "s" : ""})`,
      subtitle: buildGithubSubtitle(eventData?.action || "", event.createdAt),
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnReleaseEventData;
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

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
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
      title: node.name!,
      iconSrc: githubIcon,
      iconColor: getColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnReleaseEventData;
      const assetCount = eventData?.release?.assets?.length || 0;
      const releaseName = eventData?.release?.name || eventData?.release?.tag_name || "Release";

      props.lastEventData = {
        title: `${releaseName} (${assetCount} asset${assetCount !== 1 ? "s" : ""})`,
        subtitle: buildGithubSubtitle(eventData?.action || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
