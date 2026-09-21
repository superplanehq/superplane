import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";
import type { Predicate } from "../utils";
import { formatPredicate } from "../utils";

interface OnPipelineDoneMetadata {
  project?: {
    id: string;
    name: string;
    url: string;
  };
}

interface OnPipelineDoneConfiguration {
  refs?: Predicate[];
  results?: string[];
  pipelines?: Predicate[];
}

interface OnPipelineDoneEventData {
  project?: {
    name: string;
  };
  repository?: {
    slug: string;
    url: string;
  };
  revision?: {
    commit_sha: string;
  };
  pipeline?: {
    working_directory: string;
    yaml_file_name: string;
    name: string;
    state: string;
    result: string;
    done_at: string;
  };
}

/**
 * Renderer for the "semaphore.onPipelineDone" trigger type
 */
export const onPipelineDoneTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as OnPipelineDoneEventData;

    return {
      title: pipelineTitle(eventData),
      subtitle: pipelineSubtitle(pipelineResult(eventData), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return pipelineDoneValues(context.event?.data as OnPipelineDoneEventData);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnPipelineDoneMetadata;
    const configuration = node.configuration as unknown as OnPipelineDoneConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: SemaphoreLogo,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: pipelineDoneMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const { title, subtitle } = onPipelineDoneTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function pipelineDoneValues(eventData?: OnPipelineDoneEventData): Record<string, string> {
  return {
    "Done At": pipelineDoneAt(eventData),
    Result: pipelineResult(eventData),
    Project: eventData?.project?.name || "",
    Repository: eventData?.repository?.slug || "",
    "Repository URL": repositoryUrl(eventData),
    "Commit URL": commitUrl(eventData),
    Pipeline: eventData?.pipeline?.name || "",
    "Pipeline File": pipelineFile(eventData),
  };
}

function pipelineTitle(eventData?: OnPipelineDoneEventData): string {
  return `${pipelineFile(eventData)} (${eventData?.pipeline?.name || ""})`;
}

function pipelineFile(eventData?: OnPipelineDoneEventData): string {
  const pipeline = eventData?.pipeline;
  return `${pipeline?.working_directory || ""}/${pipeline?.yaml_file_name}`;
}

function pipelineResult(eventData?: OnPipelineDoneEventData): string {
  return eventData?.pipeline?.result || "";
}

function pipelineDoneAt(eventData?: OnPipelineDoneEventData): string {
  const doneAt = eventData?.pipeline?.done_at;
  if (!doneAt) {
    return "";
  }

  return new Date(doneAt).toLocaleString();
}

function repositoryUrl(eventData?: OnPipelineDoneEventData): string {
  return eventData?.repository?.url || "";
}

function commitUrl(eventData?: OnPipelineDoneEventData): string {
  const url = repositoryUrl(eventData);
  const commitSha = eventData?.revision?.commit_sha || "";
  if (!url || !commitSha) {
    return "";
  }

  return `${url}/commit/${commitSha}`;
}

function pipelineSubtitle(result: string, createdAt?: string): string | React.ReactNode {
  if (result && createdAt) {
    return renderWithTimeAgo(result, new Date(createdAt));
  }

  return result || (createdAt ? renderTimeAgo(new Date(createdAt)) : "");
}

function pipelineDoneMetadataItems(metadata?: OnPipelineDoneMetadata, configuration?: OnPipelineDoneConfiguration) {
  const metadataItems: MetadataItem[] = [];

  if (metadata?.project?.name) {
    metadataItems.push({
      icon: "book",
      label: metadata.project.name,
    });
  }

  appendPredicateMetadata(metadataItems, "funnel", configuration?.refs);
  appendJoinedMetadata(metadataItems, "list-filter", configuration?.results);
  appendPredicateMetadata(metadataItems, "file-code", configuration?.pipelines);

  return metadataItems;
}

function appendJoinedMetadata(metadataItems: MetadataItem[], icon: string, values: string[] | undefined): void {
  if (!values || values.length === 0) {
    return;
  }

  metadataItems.push({ icon, label: values.join(", ") });
}

function appendPredicateMetadata(
  metadataItems: MetadataItem[],
  icon: string,
  predicates: Predicate[] | undefined,
): void {
  if (!predicates || predicates.length === 0) {
    return;
  }

  metadataItems.push({ icon, label: predicates.map(formatPredicate).join(", ") });
}
