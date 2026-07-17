import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { Predicate } from "../utils";
import { formatPredicate } from "../utils";
import { buildGitlabSubtitle } from "./utils";
import type { GitLabNodeMetadata } from "./types";

interface OnPushConfiguration {
  branches: Predicate[];
}

interface PushCommit {
  id?: string;
  message?: string;
  title?: string;
  url?: string;
  author?: {
    name?: string;
    email?: string;
  };
}

interface OnPushEventData {
  object_kind?: string;
  event_name?: string;
  ref?: string;
  before?: string;
  after?: string;
  checkout_sha?: string;
  user_name?: string;
  user_username?: string;
  total_commits_count?: number;
  commits?: PushCommit[];
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

function headCommit(eventData?: OnPushEventData): PushCommit | undefined {
  const commits = eventData?.commits;
  if (!commits || commits.length === 0) {
    return undefined;
  }

  const head = eventData?.after || eventData?.checkout_sha;
  return commits.find((commit) => commit.id === head) || commits[commits.length - 1];
}

function commitMessage(commit?: PushCommit): string {
  const message = commit?.title || commit?.message;
  return message ? message.split("\n")[0].trim() : "";
}

function pushAuthor(eventData?: OnPushEventData): string {
  return eventData?.user_name || eventData?.user_username || headCommit(eventData)?.author?.name || "";
}

function pushEventTitle(eventData?: OnPushEventData): string {
  const message = commitMessage(headCommit(eventData));
  if (message) {
    return message;
  }

  const branch = branchName(eventData?.ref);
  return branch !== "-" ? `Push to ${branch}` : "Push";
}

function buildMetadataItems(metadata?: GitLabNodeMetadata, configuration?: OnPushConfiguration) {
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

export const onPushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnPushEventData;
    const shortSha = (eventData?.after || eventData?.checkout_sha || "").slice(0, 8);

    return {
      title: pushEventTitle(eventData),
      subtitle: buildGitlabSubtitle(shortSha, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnPushEventData;
    const commit = headCommit(eventData);

    return {
      "Received At": formatReceivedAt(context.event?.createdAt),
      Branch: branchName(eventData?.ref),
      Commit: commitMessage(commit) || "-",
      Author: pushAuthor(eventData) || "-",
      Commits: eventData?.total_commits_count != null ? String(eventData.total_commits_count) : "-",
      "Commit URL": commit?.url || "-",
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnPushConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: gitlabIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnPushEventData;
      const shortSha = (eventData?.after || eventData?.checkout_sha || "").slice(0, 8);

      props.lastEventData = {
        title: pushEventTitle(eventData),
        subtitle: buildGitlabSubtitle(shortSha, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
