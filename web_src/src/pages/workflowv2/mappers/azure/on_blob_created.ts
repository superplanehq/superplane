import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type React from "react";
import type { TriggerProps } from "@/ui/trigger";
import azureIcon from "@/assets/icons/integrations/azure.svg";
import type { AzureBlobEvent } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../utils";
import { getBackgroundColorClass } from "@/lib/colors";
import type { MetadataItem } from "@/ui/metadataList";

export interface OnBlobCreatedConfiguration {
  resourceGroup?: string;
  storageAccount?: string;
  containerFilter?: string;
  blobFilter?: string;
}

function extractBlobContainer(subject?: string): string | undefined {
  if (!subject) return undefined;
  const marker = "/containers/";
  const idx = subject.indexOf(marker);
  if (idx < 0) return undefined;
  const rest = subject.slice(idx + marker.length);
  const slash = rest.indexOf("/");
  return slash < 0 ? rest : rest.slice(0, slash);
}

function extractBlobName(subject?: string): string | undefined {
  if (!subject) return undefined;
  const marker = "/blobs/";
  const idx = subject.indexOf(marker);
  if (idx < 0) return undefined;
  return subject.slice(idx + marker.length);
}

export const onBlobCreatedTriggerRenderer: TriggerRenderer = {
  subtitle: (context: TriggerEventContext): string | React.ReactNode => {
    const subtitle = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";

    return subtitle;
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const envelope = context.event?.data as AzureBlobEvent | undefined;
    const blobData = envelope?.data;

    return {
      Container: stringOrDash(extractBlobContainer(envelope?.subject)),
      Blob: stringOrDash(extractBlobName(envelope?.subject)),
      "Blob Type": stringOrDash(blobData?.blobType),
      "Content Type": stringOrDash(blobData?.contentType),
      URL: stringOrDash(blobData?.url),
      Api: stringOrDash(blobData?.api),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnBlobCreatedConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    if (configuration?.storageAccount) {
      metadataItems.push({ icon: "database", label: configuration.storageAccount });
    }

    if (configuration?.containerFilter) {
      metadataItems.push({ icon: "funnel", label: configuration.containerFilter });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: azureIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const subtitle = onBlobCreatedTriggerRenderer.subtitle({ event: lastEvent });
      props.lastEventData = {
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
