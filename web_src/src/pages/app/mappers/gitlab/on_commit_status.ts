import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { Predicate } from "../utils";
import { formatPredicate, stringOrDash } from "../utils";
import { buildGitlabSubtitle, shortSha } from "./utils";
import type { GitLabNodeMetadata } from "./types";

interface OnCommitStatusConfiguration {
  statuses?: string[];
  names?: Predicate[];
  refs?: Predicate[];
}

interface CommitStatusBuild {
  id?: number;
  name?: string;
  stage?: string;
  status?: string;
}

interface CommitStatusPipelineAttributes {
  id?: number;
  iid?: number;
  ref?: string;
  sha?: string;
  status?: string;
  source?: string;
  url?: string;
}

interface OnCommitStatusEventData {
  object_kind?: string;
  object_attributes?: CommitStatusPipelineAttributes;
  builds?: CommitStatusBuild[];
  commit?: {
    id?: string;
    title?: string;
    url?: string;
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

function commitSha(eventData?: OnCommitStatusEventData): string {
  const sha = eventData?.object_attributes?.sha || eventData?.commit?.id;
  return sha ? shortSha(sha) : "-";
}

function statusNames(eventData?: OnCommitStatusEventData): string {
  const names = (eventData?.builds || []).map((build) => build.name).filter((name): name is string => !!name);
  return names.length ? names.join(", ") : "-";
}

function commitStatusTitle(eventData?: OnCommitStatusEventData): string {
  const sha = commitSha(eventData);
  return sha !== "-" ? `Commit ${sha}` : "Commit Status";
}

function buildMetadataItems(metadata?: GitLabNodeMetadata, configuration?: OnCommitStatusConfiguration) {
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

export const onCommitStatusTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnCommitStatusEventData;

    return {
      title: commitStatusTitle(eventData),
      subtitle: buildGitlabSubtitle(eventData?.object_attributes?.status || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnCommitStatusEventData;
    const attributes = eventData?.object_attributes;

    return {
      "Received At": formatReceivedAt(context.event?.createdAt),
      Status: stringOrDash(attributes?.status),
      "Status Names": statusNames(eventData),
      Ref: stringOrDash(attributes?.ref),
      Commit: commitSha(eventData),
      "Pipeline URL": stringOrDash(attributes?.url),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnCommitStatusConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: gitlabIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnCommitStatusEventData;

      props.lastEventData = {
        title: commitStatusTitle(eventData),
        subtitle: buildGitlabSubtitle(eventData?.object_attributes?.status || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
