import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { Predicate } from "../utils";
import { formatPredicate } from "../utils";
import { buildGitlabSubtitle } from "./utils";
import type { GitLabNodeMetadata } from "./types";

interface OnBranchCreatedConfiguration {
  branches: Predicate[];
}

interface OnBranchCreatedEventData {
  object_kind?: string;
  event_name?: string;
  ref?: string;
  before?: string;
  after?: string;
  user_name?: string;
  user_username?: string;
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

function branchName(ref?: string): string {
  if (!ref) {
    return "-";
  }

  return ref.replace(/^refs\/heads\//, "");
}

function branchUrl(eventData?: OnBranchCreatedEventData): string {
  const webUrl = eventData?.project?.web_url;
  const branch = branchName(eventData?.ref);
  if (!webUrl || branch === "-") {
    return "-";
  }

  // Encode each path segment so branch names with URL-significant characters
  // (e.g. "fix#42") produce a valid link, while keeping "/" separators intact.
  const encodedBranch = branch.split("/").map(encodeURIComponent).join("/");
  return `${webUrl}/-/tree/${encodedBranch}`;
}

function branchEventTitle(eventData?: OnBranchCreatedEventData): string {
  const branch = branchName(eventData?.ref);
  return branch !== "-" ? `Branch: ${branch}` : "Branch Created";
}

function buildMetadataItems(metadata?: GitLabNodeMetadata, configuration?: OnBranchCreatedConfiguration) {
  const metadataItems = [];

  if (metadata?.project?.name) {
    metadataItems.push({
      icon: "book",
      label: metadata.project.name,
    });
  }

  if (configuration?.branches?.length) {
    metadataItems.push({
      icon: "funnel",
      label: configuration.branches.map((branch) => formatPredicate(branch)).join(", "),
    });
  }

  return metadataItems;
}

export const onBranchCreatedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnBranchCreatedEventData;

    return {
      title: branchEventTitle(eventData),
      subtitle: buildGitlabSubtitle(branchName(eventData?.ref), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnBranchCreatedEventData;

    return {
      "Received At": formatReceivedAt(context.event?.createdAt),
      Branch: branchName(eventData?.ref),
      Project: eventData?.project?.path_with_namespace || "-",
      Author: eventData?.user_name || eventData?.user_username || "-",
      SHA: eventData?.after ? eventData.after.slice(0, 8) : "-",
      "Branch URL": branchUrl(eventData),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnBranchCreatedConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: gitlabIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnBranchCreatedEventData;

      props.lastEventData = {
        title: branchEventTitle(eventData),
        subtitle: buildGitlabSubtitle(branchName(eventData?.ref), lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
