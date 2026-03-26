import type { ComponentBaseProps } from "@/ui/componentBase";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { addFormattedTimestamp, addOrderedDetails, buildEventSections } from "./utils";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

interface CreateReleaseConfiguration {
  project?: string;
  version?: string;
  commits?: Array<unknown>;
  refs?: Array<unknown>;
}

interface CreateReleaseNodeMetadata {
  project?: {
    name?: string;
    slug?: string;
  };
}

interface SentryReleaseOutput {
  version?: string;
  shortVersion?: string;
  url?: string;
  dateReleased?: string;
  commitCount?: number;
  deployCount?: number;
  projects?: Array<{
    name?: string;
    slug?: string;
  }>;
}

export const createReleaseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: sentryIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution
        ? buildEventSections(context.nodes, lastExecution, componentName, getTriggerRenderer, getState)
        : undefined,
      metadata: buildMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const release = outputs?.default?.[0]?.data as SentryReleaseOutput | undefined;
    const timestamp = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt));
    return [release?.shortVersion || release?.version, timestamp].filter(Boolean).join(" · ");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const release = outputs?.default?.[0]?.data as SentryReleaseOutput | undefined;
    const details: Record<string, string> = {};

    addFormattedTimestamp(details, "Started At", context.execution.createdAt);
    addOrderedDetails(details, [
      { label: "Version", value: release?.version },
      { label: "Project", value: getReleaseProjectLabel(release) },
      { label: "Release URL", value: release?.url },
      { label: "Released At", value: release?.dateReleased, isTimestamp: true },
      { label: "Commits", value: formatNumericValue(release?.commitCount) },
      { label: "Deploys", value: formatNumericValue(release?.deployCount) },
    ]);

    return details;
  },
};

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as CreateReleaseConfiguration | undefined;
  const nodeMetadata = node.metadata as CreateReleaseNodeMetadata | undefined;
  const metadata = [];
  const activityLabel = getReleaseActivityLabel(configuration);

  const projectLabel = nodeMetadata?.project?.name || nodeMetadata?.project?.slug || configuration?.project;
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }

  if (configuration?.version) {
    metadata.push({ icon: "tag", label: configuration.version });
  }

  if (activityLabel) {
    metadata.push(activityLabel);
  }

  return metadata.slice(0, 3);
}

function getReleaseActivityLabel(configuration: CreateReleaseConfiguration | undefined) {
  const commitsCount = configuration?.commits?.length ?? 0;
  if (commitsCount > 0) {
    return {
      icon: "git-commit-horizontal",
      label: `${commitsCount} commit${commitsCount === 1 ? "" : "s"}`,
    };
  }

  const refsCount = configuration?.refs?.length ?? 0;
  if (refsCount > 0) {
    return {
      icon: "git-branch",
      label: `${refsCount} ref${refsCount === 1 ? "" : "s"}`,
    };
  }

  return undefined;
}

function getReleaseProjectLabel(release: SentryReleaseOutput | undefined) {
  return release?.projects?.[0]?.name || release?.projects?.[0]?.slug;
}

function formatNumericValue(value: number | undefined) {
  return value !== undefined ? String(value) : undefined;
}
