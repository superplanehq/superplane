import { renderTimeAgo } from "@/components/TimeAgo";
import type React from "react";
import jfrogIcon from "@/assets/icons/integrations/jfrog-artifactory.svg";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass } from "@/utils/colors";

interface OnArtifactUploadedConfiguration {
  repository?: string;
}

interface OnArtifactUploadedEventData {
  repo?: string;
  path?: string;
  name?: string;
  size?: number;
  sha256?: string;
}

export const onArtifactUploadedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as OnArtifactUploadedEventData;
    const name = eventData?.name || "Artifact";
    const repo = eventData?.repo;
    const title = repo ? `${name} in ${repo}` : name;
    const subtitle = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnArtifactUploadedEventData;
    const values: Record<string, string> = {};

    if (eventData?.repo !== undefined) {
      values["Repo"] = String(eventData.repo);
    }

    if (eventData?.path !== undefined) {
      values["Path"] = String(eventData.path);
    }

    if (eventData?.name !== undefined) {
      values["Name"] = String(eventData.name);
    }

    if (eventData?.size !== undefined) {
      values["Size"] = String(eventData.size);
    }

    if (eventData?.sha256 !== undefined) {
      values["SHA256"] = String(eventData.sha256);
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as unknown as OnArtifactUploadedConfiguration;
    const metadataItems = [];

    if (configuration?.repository) {
      metadataItems.push({
        icon: "archive",
        label: configuration.repository,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: jfrogIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnArtifactUploadedEventData;
      const name = eventData?.name || "Artifact";
      const repo = eventData?.repo;
      const title = repo ? `${name} in ${repo}` : name;
      const subtitle = lastEvent.createdAt ? renderTimeAgo(new Date(lastEvent.createdAt)) : "";

      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
