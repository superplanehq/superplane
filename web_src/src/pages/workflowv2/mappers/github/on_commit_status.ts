import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { MetadataItem } from "@/ui/metadataList";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { Predicate } from "../utils";
import { formatPredicate, stringOrDash } from "../utils";
import githubIcon from "@/assets/icons/integrations/github.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { BaseNodeMetadata } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnCommitStatusConfiguration {
  states?: string[];
  contexts?: Predicate[];
  branches?: Predicate[];
}

interface StatusBranch {
  name?: string;
}

interface StatusCommit {
  sha?: string;
  html_url?: string;
  author?: {
    login?: string;
  };
  commit?: {
    author?: {
      name?: string;
    };
    message?: string;
  };
}

interface OnCommitStatusEventData {
  sha?: string;
  state?: string;
  context?: string;
  description?: string;
  target_url?: string;
  branches?: StatusBranch[];
  commit?: StatusCommit;
  repository?: {
    full_name?: string;
    html_url?: string;
  };
  sender?: {
    login?: string;
  };
}

export const onCommitStatusTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnCommitStatusEventData;

    return {
      title: statusTitle(eventData),
      subtitle: buildGithubSubtitle(eventData?.state || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnCommitStatusEventData;

    return {
      State: stringOrDash(statusState(eventData)),
      Context: stringOrDash(statusContext(eventData)),
      Description: stringOrDash(statusDescription(eventData)),
      SHA: stringOrDash(statusSha(eventData)),
      "Commit message": stringOrDash(statusCommitMessage(eventData)),
      "Commit author": stringOrDash(statusCommitAuthor(eventData)),
      Branches: stringOrDash(statusBranchNames(eventData).join(", ")),
      Repository: stringOrDash(statusRepositoryName(eventData)),
      "Status creator": stringOrDash(statusCreatorLogin(eventData)),
      "Commit URL": stringOrDash(statusCommitUrl(eventData)),
      "Target URL": stringOrDash(statusTargetUrl(eventData)),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnCommitStatusConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: statusMetadataItems(metadata?.repository?.name, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnCommitStatusEventData;
      props.lastEventData = {
        title: statusTitle(eventData),
        subtitle: buildGithubSubtitle(eventData?.state || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function statusMetadataItems(
  repositoryName: string | undefined,
  configuration?: OnCommitStatusConfiguration,
): MetadataItem[] {
  const metadataItems: MetadataItem[] = [];

  if (repositoryName) {
    metadataItems.push({
      icon: "book",
      label: repositoryName,
    });
  }

  if (configuration?.states && configuration.states.length > 0) {
    metadataItems.push({
      icon: "circle-check",
      label: configuration.states.join(", "),
    });
  }

  if (configuration?.contexts && configuration.contexts.length > 0) {
    metadataItems.push({
      icon: "funnel",
      label: `context ${configuration.contexts.map(formatPredicate).join(", ")}`,
    });
  }

  if (configuration?.branches && configuration.branches.length > 0) {
    metadataItems.push({
      icon: "git-branch",
      label: `branch ${configuration.branches.map(formatPredicate).join(", ")}`,
    });
  }

  return metadataItems;
}

function statusTitle(eventData: OnCommitStatusEventData | undefined): string {
  const context = statusContext(eventData) || "Commit status";
  const state = statusTitleState(statusState(eventData));
  const sha = shortSha(statusSha(eventData));
  const title = state ? `${context} ${state}` : context;

  if (sha) {
    return `${title} - ${sha}`;
  }

  return title;
}

function statusTitleState(state: string | undefined): string {
  switch (state) {
    case "success":
      return "succeeded";
    case "failure":
      return "failed";
    case "error":
      return "errored";
    case "pending":
      return "is pending";
    default:
      return "";
  }
}

function shortSha(sha: string | undefined): string {
  return sha ? sha.slice(0, 7) : "";
}

function statusState(eventData: OnCommitStatusEventData | undefined): string | undefined {
  return eventData?.state;
}

function statusContext(eventData: OnCommitStatusEventData | undefined): string | undefined {
  return eventData?.context;
}

function statusSha(eventData: OnCommitStatusEventData | undefined): string | undefined {
  return eventData?.sha || eventData?.commit?.sha;
}

function statusDescription(eventData: OnCommitStatusEventData | undefined): string | undefined {
  return eventData?.description;
}

function statusTargetUrl(eventData: OnCommitStatusEventData | undefined): string | undefined {
  return eventData?.target_url;
}

function statusRepositoryName(eventData: OnCommitStatusEventData | undefined): string | undefined {
  return eventData?.repository?.full_name;
}

function statusCreatorLogin(eventData: OnCommitStatusEventData | undefined): string | undefined {
  return eventData?.sender?.login;
}

function statusCommitAuthor(eventData: OnCommitStatusEventData | undefined): string | undefined {
  return eventData?.commit?.author?.login || eventData?.commit?.commit?.author?.name;
}

function statusCommitMessage(eventData: OnCommitStatusEventData | undefined): string | undefined {
  const message = eventData?.commit?.commit?.message;
  return message?.split("\n")[0];
}

function statusCommitUrl(eventData: OnCommitStatusEventData | undefined): string | undefined {
  return eventData?.commit?.html_url;
}

function statusBranchNames(eventData: OnCommitStatusEventData | undefined): string[] {
  return (eventData?.branches || []).map((branch) => branch.name).filter((name): name is string => Boolean(name));
}
