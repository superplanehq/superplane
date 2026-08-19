import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { buildGitlabSubtitle } from "./utils";
import type { GitLabNodeMetadata } from "./types";

interface OnMRDiffNoteConfiguration {
  contentFilter?: string;
}

interface DiffNotePosition {
  old_path?: string;
  new_path?: string;
  old_line?: number;
  new_line?: number;
}

interface DiffNoteObjectAttributes {
  id?: number;
  note?: string;
  noteable_type?: string;
  type?: string;
  url?: string;
  position?: DiffNotePosition;
}

interface OnMRDiffNoteEventData {
  object_kind?: string;
  event_type?: string;
  object_attributes?: DiffNoteObjectAttributes;
  merge_request?: {
    id?: number;
    iid?: number;
    title?: string;
    state?: string;
    url?: string;
  };
  user?: {
    id: number;
    name: string;
    username: string;
  };
  project?: {
    id: number;
    name: string;
    path_with_namespace: string;
    web_url: string;
  };
}

function formatReceivedAt(createdAt?: string): string {
  return createdAt ? new Date(createdAt).toLocaleString() : "-";
}

function mergeRequestRef(mr?: OnMRDiffNoteEventData["merge_request"]): string {
  if (!mr?.iid) {
    return "-";
  }

  return `!${mr.iid} - ${mr.title || ""}`;
}

function diffLocation(position?: DiffNotePosition): string {
  const path = position?.new_path || position?.old_path;
  if (!path) {
    return "-";
  }

  const line = position?.new_line ?? position?.old_line;
  return line ? `${path}:${line}` : path;
}

function commentEventTitle(eventData?: OnMRDiffNoteEventData): string {
  const mr = eventData?.merge_request;
  return `!${mr?.iid ?? ""} - ${mr?.title || "MR Diff Note"}`;
}

function commentEventSubtitle(eventData?: OnMRDiffNoteEventData, createdAt?: string) {
  const author = eventData?.user?.username;
  return buildGitlabSubtitle(author ? `By ${author}` : "", createdAt);
}

function buildMetadataItems(metadata?: GitLabNodeMetadata, configuration?: OnMRDiffNoteConfiguration) {
  const metadataItems = [];

  if (metadata?.project?.name) {
    metadataItems.push({
      icon: "book",
      label: metadata.project.name,
    });
  }

  if (configuration?.contentFilter) {
    metadataItems.push({
      icon: "funnel",
      label: `Filter: ${configuration.contentFilter}`,
    });
  }

  return metadataItems;
}

export const onMRDiffNoteTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnMRDiffNoteEventData;

    return {
      title: commentEventTitle(eventData),
      subtitle: commentEventSubtitle(eventData, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnMRDiffNoteEventData;
    const comment = eventData?.object_attributes;

    return {
      "Received At": formatReceivedAt(context.event?.createdAt),
      Comment: comment?.note || "-",
      "Diff Location": diffLocation(comment?.position),
      Author: eventData?.user?.username || "-",
      "Merge Request": mergeRequestRef(eventData?.merge_request),
      Project: eventData?.project?.path_with_namespace || "-",
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnMRDiffNoteConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: gitlabIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnMRDiffNoteEventData;

      props.lastEventData = {
        title: commentEventTitle(eventData),
        subtitle: commentEventSubtitle(eventData, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
