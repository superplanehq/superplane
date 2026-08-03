import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { Predicate } from "../utils";
import { formatPredicate, stringOrDash } from "../utils";
import { buildGitlabSubtitle } from "./utils";
import type { GitLabNodeMetadata } from "./types";

interface OnJobConfiguration {
  statuses?: string[];
  names?: Predicate[];
  refs?: Predicate[];
}

interface OnJobEventData {
  object_kind?: string;
  ref?: string;
  sha?: string;
  build_id?: number;
  build_name?: string;
  build_stage?: string;
  build_status?: string;
  pipeline_id?: number;
  project?: {
    id?: number;
    name?: string;
    path_with_namespace?: string;
    web_url?: string;
  };
}

function formatReceivedAt(createdAt?: string): string {
  return createdAt ? new Date(createdAt).toLocaleString() : "-";
}

// GitLab's job payload has no job URL, so build it from the project's web_url.
function jobUrl(eventData?: OnJobEventData): string {
  const webUrl = eventData?.project?.web_url;
  const buildId = eventData?.build_id;
  if (!webUrl || !buildId) {
    return "-";
  }

  return `${webUrl}/-/jobs/${buildId}`;
}

function jobTitle(eventData?: OnJobEventData): string {
  const name = eventData?.build_name;
  return name ? `Job: ${name}` : "Job";
}

function buildMetadataItems(metadata?: GitLabNodeMetadata, configuration?: OnJobConfiguration) {
  const metadataItems = [];

  if (metadata?.project?.name) {
    metadataItems.push({
      icon: "book",
      label: metadata.project.name,
    });
  }

  if (configuration?.statuses?.length) {
    metadataItems.push({
      icon: "funnel",
      label: configuration.statuses.join(", "),
    });
  }

  if (configuration?.names?.length) {
    metadataItems.push({
      icon: "tag",
      label: configuration.names.map((name) => formatPredicate(name)).join(", "),
    });
  }

  if (configuration?.refs?.length) {
    metadataItems.push({
      icon: "git-branch",
      label: configuration.refs.map((ref) => formatPredicate(ref)).join(", "),
    });
  }

  return metadataItems;
}

export const onJobTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnJobEventData;

    return {
      title: jobTitle(eventData),
      subtitle: buildGitlabSubtitle(eventData?.build_status || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnJobEventData;

    return {
      "Received At": formatReceivedAt(context.event?.createdAt),
      Job: stringOrDash(eventData?.build_name),
      Status: stringOrDash(eventData?.build_status),
      Stage: stringOrDash(eventData?.build_stage),
      Ref: stringOrDash(eventData?.ref),
      "Job URL": jobUrl(eventData),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnJobConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: gitlabIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnJobEventData;

      props.lastEventData = {
        title: jobTitle(eventData),
        subtitle: buildGitlabSubtitle(eventData?.build_status || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
