import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { NodeMetadata, PullRequest, PullRequestEvent } from "./types";
import type { Predicate } from "../utils";
import { formatPredicate } from "../utils";
import { buildBitbucketSubtitle } from "./utils";

export interface OnPullRequestConfiguration {
  repository?: string;
  actions?: string[];
  targetBranches?: Predicate[];
}

const ACTION_LABELS: Record<string, string> = {
  created: "Opened",
  updated: "Updated",
  fulfilled: "Merged",
  rejected: "Declined",
  approved: "Approved",
  unapproved: "Approval removed",
  changes_request_created: "Changes requested",
  changes_request_removed: "Changes request removed",
};

function formatAction(action: string): string {
  return ACTION_LABELS[action] || action;
}

function pullRequestTitle(event?: PullRequestEvent): string {
  const pullRequest = event?.pullrequest;
  if (!pullRequest) {
    return "";
  }

  return `#${pullRequest.id ?? ""} ${pullRequest.title || ""}`.trim();
}

function branchNames(pullRequest?: PullRequest): { source: string; target: string } {
  return {
    source: pullRequest?.source?.branch?.name ?? "",
    target: pullRequest?.destination?.branch?.name ?? "",
  };
}

function pullRequestSubtitle(event?: PullRequestEvent): string {
  const { source, target } = branchNames(event?.pullrequest);

  if (source && target) {
    return `${source} → ${target}`;
  }

  return source || target;
}

function pullRequestEventValues(event?: PullRequestEvent): Record<string, string> {
  const pullRequest = event?.pullrequest;
  const branches = branchNames(pullRequest);

  return {
    "Pull Request": pullRequest?.id ? `#${pullRequest.id}` : "",
    Title: pullRequest?.title ?? "",
    State: pullRequest?.state ?? "",
    Source: branches.source,
    Target: branches.target,
    Author: event?.actor?.display_name ?? "",
  };
}

/**
 * Renderer for the "bitbucket.onPullRequest" trigger
 */
export const onPullRequestTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as PullRequestEvent;

    return {
      title: pullRequestTitle(eventData),
      subtitle: buildBitbucketSubtitle(pullRequestSubtitle(eventData), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return pullRequestEventValues(context.event?.data as PullRequestEvent);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as NodeMetadata;
    const configuration = node.configuration as unknown as OnPullRequestConfiguration;
    const metadataItems = [];

    if (metadata?.repository) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.full_name || metadata.repository.name || "",
      });
    }

    if (configuration?.actions?.length) {
      metadataItems.push({
        icon: "activity",
        label: configuration.actions.map(formatAction).join(", "),
      });
    }

    if (configuration?.targetBranches?.length) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.targetBranches.map(formatPredicate).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: bitbucketIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as PullRequestEvent;

      props.lastEventData = {
        title: pullRequestTitle(eventData),
        subtitle: buildBitbucketSubtitle(pullRequestSubtitle(eventData), lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
