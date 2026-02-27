import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { formatPredicate, Predicate } from "../utils";

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
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnPipelineDoneEventData;
    const result = eventData?.pipeline?.result || "";
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt)) : "";
    const pipelineFile = `${eventData?.pipeline?.working_directory || ""}/${eventData?.pipeline?.yaml_file_name}`;
    const title = `${pipelineFile} (${eventData?.pipeline?.name || ""})`;
    const subtitle = result && timeAgo ? `${result} · ${timeAgo}` : result || timeAgo;

    return {
      title: title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnPipelineDoneEventData;
    const doneAt = eventData?.pipeline?.done_at ? new Date(eventData.pipeline.done_at).toLocaleString() : "";
    const repositoryUrl = eventData?.repository?.url || "";
    const pipelineFile = `${eventData?.pipeline?.working_directory || ""}/${eventData?.pipeline?.yaml_file_name}`;
    const commitSha = eventData?.revision?.commit_sha || "";
    const commitUrl = repositoryUrl && commitSha ? `${repositoryUrl}/commit/${commitSha}` : "";

    return {
      "Done At": doneAt,
      Result: eventData?.pipeline?.result || "",
      Project: eventData?.project?.name || "",
      Repository: eventData?.repository?.slug || "",
      "Repository URL": repositoryUrl,
      "Commit URL": commitUrl,
      Pipeline: eventData?.pipeline?.name || "",
      "Pipeline File": pipelineFile,
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnPipelineDoneMetadata;
    const configuration = node.configuration as unknown as OnPipelineDoneConfiguration;
    const metadataItems: MetadataItem[] = [];

    if (metadata?.project?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.project.name,
      });
    }

    if (configuration?.refs?.length) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.refs.map(formatPredicate).join(", "),
      });
    }

    if (configuration?.results?.length) {
      metadataItems.push({
        icon: "list-filter",
        label: configuration.results.join(", "),
      });
    }

    if (configuration?.pipelines?.length) {
      metadataItems.push({
        icon: "file-code",
        label: configuration.pipelines.map(formatPredicate).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: SemaphoreLogo,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnPipelineDoneEventData;
      const result = eventData?.pipeline?.result || "";
      const pipelineFile = `${eventData?.pipeline?.working_directory || ""}/${eventData?.pipeline?.yaml_file_name}`;
      const timeAgo = lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "";
      const subtitle = result && timeAgo ? `${result} · ${timeAgo}` : result || timeAgo;

      props.lastEventData = {
        title: `${pipelineFile} (${eventData?.pipeline?.name || ""})`,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
