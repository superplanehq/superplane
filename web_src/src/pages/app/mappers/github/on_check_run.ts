import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { MetadataItem } from "@/ui/metadataList";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { Predicate } from "../utils";
import { formatPredicate, stringOrDash } from "../utils";
import githubIcon from "@/assets/icons/integrations/github.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { BaseNodeMetadata } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnCheckRunConfiguration {
  statuses?: string[];
  conclusions?: string[];
  names?: Predicate[];
  branches?: Predicate[];
  pullRequestsOnly?: boolean;
}

interface CheckRunPullRequest {
  number?: number;
  html_url?: string;
  head?: {
    ref?: string;
    sha?: string;
  };
  base?: {
    ref?: string;
  };
}

interface CheckRun {
  name?: string;
  status?: string;
  conclusion?: string;
  head_sha?: string;
  html_url?: string;
  details_url?: string;
  app?: {
    name?: string;
  };
  check_suite?: {
    head_branch?: string;
    head_sha?: string;
  };
  pull_requests?: CheckRunPullRequest[];
}

interface OnCheckRunEventData {
  action?: string;
  check_run?: CheckRun;
  repository?: {
    full_name?: string;
    html_url?: string;
  };
  sender?: {
    login?: string;
  };
}

export const onCheckRunTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnCheckRunEventData;

    return {
      title: checkRunTitle(eventData),
      subtitle: buildGithubSubtitle(checkRunResult(eventData), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnCheckRunEventData;

    return {
      Action: stringOrDash(eventData?.action),
      Name: stringOrDash(checkRunName(eventData)),
      Status: stringOrDash(checkRunStatus(eventData)),
      Conclusion: stringOrDash(checkRunConclusion(eventData)),
      Branch: stringOrDash(checkRunBranch(eventData)),
      SHA: stringOrDash(checkRunSha(eventData)),
      "Pull request": stringOrDash(checkRunPullRequestLabel(eventData)),
      App: stringOrDash(checkRunAppName(eventData)),
      "Details URL": stringOrDash(checkRunDetailsUrl(eventData)),
      Repository: stringOrDash(checkRunRepositoryName(eventData)),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnCheckRunConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: checkRunMetadataItems(metadata?.repository?.name, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnCheckRunEventData;
      props.lastEventData = {
        title: checkRunTitle(eventData),
        subtitle: buildGithubSubtitle(checkRunResult(eventData), lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function checkRunMetadataItems(
  repositoryName: string | undefined,
  configuration?: OnCheckRunConfiguration,
): MetadataItem[] {
  const metadataItems: MetadataItem[] = [];

  if (repositoryName) {
    metadataItems.push({ icon: "book", label: repositoryName });
  }

  appendCheckRunConfigurationMetadata(metadataItems, configuration);

  return metadataItems;
}

function appendCheckRunConfigurationMetadata(
  metadataItems: MetadataItem[],
  configuration?: OnCheckRunConfiguration,
): void {
  appendJoinedMetadata(metadataItems, "circle-check", configuration?.statuses);
  if (configuration?.conclusions && configuration.conclusions.length > 0) {
    metadataItems.push({ icon: "funnel", label: `conclusion ${configuration.conclusions.join(", ")}` });
  }

  if (configuration?.names && configuration.names.length > 0) {
    metadataItems.push({ icon: "funnel", label: `name ${configuration.names.map(formatPredicate).join(", ")}` });
  }

  if (configuration?.branches && configuration.branches.length > 0) {
    metadataItems.push({
      icon: "git-branch",
      label: `branch ${configuration.branches.map(formatPredicate).join(", ")}`,
    });
  }

  if (configuration?.pullRequestsOnly) {
    metadataItems.push({ icon: "git-pull-request", label: "pull requests only" });
  }
}

function appendJoinedMetadata(metadataItems: MetadataItem[], icon: string, values: string[] | undefined): void {
  if (!values || values.length === 0) {
    return;
  }

  metadataItems.push({ icon, label: values.join(", ") });
}

function checkRunTitle(eventData: OnCheckRunEventData | undefined): string {
  const name = checkRunName(eventData) || "Check run";
  const state = checkRunTitleState(checkRunStatus(eventData), checkRunConclusion(eventData));
  const sha = shortSha(checkRunSha(eventData));
  const title = state ? `${name} ${state}` : name;

  if (sha) {
    return `${title} - ${sha}`;
  }

  return title;
}

function checkRunTitleState(status: string | undefined, conclusion: string | undefined): string {
  if (status === "completed") {
    switch (conclusion) {
      case "success":
        return "succeeded";
      case "failure":
        return "failed";
      case "cancelled":
        return "was cancelled";
      case "skipped":
        return "was skipped";
      case "timed_out":
        return "timed out";
      case "action_required":
        return "needs action";
      case "stale":
        return "is stale";
      case "neutral":
        return "completed";
      default:
        return "completed";
    }
  }

  if (status === "in_progress") {
    return "is running";
  }

  if (status === "queued") {
    return "is queued";
  }

  return "";
}

function checkRunResult(eventData: OnCheckRunEventData | undefined): string {
  return checkRunConclusion(eventData) || checkRunStatus(eventData) || "";
}

function checkRunName(eventData: OnCheckRunEventData | undefined): string | undefined {
  return eventData?.check_run?.name;
}

function checkRunStatus(eventData: OnCheckRunEventData | undefined): string | undefined {
  return eventData?.check_run?.status;
}

function checkRunConclusion(eventData: OnCheckRunEventData | undefined): string | undefined {
  return eventData?.check_run?.conclusion;
}

function checkRunBranch(eventData: OnCheckRunEventData | undefined): string | undefined {
  return eventData?.check_run?.check_suite?.head_branch || eventData?.check_run?.pull_requests?.[0]?.head?.ref;
}

function checkRunSha(eventData: OnCheckRunEventData | undefined): string | undefined {
  return eventData?.check_run?.head_sha || eventData?.check_run?.check_suite?.head_sha;
}

function checkRunPullRequestLabel(eventData: OnCheckRunEventData | undefined): string | undefined {
  const pullRequest = eventData?.check_run?.pull_requests?.[0];
  if (!pullRequest?.number) {
    return undefined;
  }

  return `#${pullRequest.number}`;
}

function checkRunAppName(eventData: OnCheckRunEventData | undefined): string | undefined {
  return eventData?.check_run?.app?.name;
}

function checkRunDetailsUrl(eventData: OnCheckRunEventData | undefined): string | undefined {
  return eventData?.check_run?.details_url || eventData?.check_run?.html_url;
}

function checkRunRepositoryName(eventData: OnCheckRunEventData | undefined): string | undefined {
  return eventData?.repository?.full_name;
}

function shortSha(sha: string | undefined): string {
  return sha ? sha.slice(0, 7) : "";
}
