import type { ComponentBaseProps } from "@/ui/componentBase";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { addDetail, addFormattedTimestamp, buildEventSections } from "./utils";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

interface CreateDeployConfiguration {
  project?: string;
  releaseVersion?: string;
  environment?: string;
}

interface CreateDeployNodeMetadata {
  project?: {
    name?: string;
    slug?: string;
  };
}

interface SentryDeployOutput {
  releaseVersion?: string;
  environment?: string;
  name?: string;
  url?: string;
  dateStarted?: string;
  dateFinished?: string;
}

export const createDeployMapper: ComponentBaseMapper = {
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
    const deploy = outputs?.default?.[0]?.data as SentryDeployOutput | undefined;
    const timestamp = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt));
    return [deploy?.environment, timestamp].filter(Boolean).join(" · ");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const deploy = outputs?.default?.[0]?.data as SentryDeployOutput | undefined;
    const details: Record<string, string> = {};

    addFormattedTimestamp(details, "Started At", context.execution.createdAt);
    addDetail(details, "Release Version", deploy?.releaseVersion);
    addDetail(details, "Environment", deploy?.environment);
    addDetail(details, "Name", deploy?.name);
    addDetail(details, "Deploy URL", deploy?.url);
    addFormattedTimestamp(details, "Finished At", deploy?.dateFinished);

    return details;
  },
};

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as CreateDeployConfiguration | undefined;
  const nodeMetadata = node.metadata as CreateDeployNodeMetadata | undefined;
  const metadata = [];

  if (configuration?.releaseVersion) {
    metadata.push({ icon: "tag", label: configuration.releaseVersion });
  }

  if (configuration?.environment) {
    metadata.push({ icon: "rocket", label: configuration.environment });
  }

  const projectLabel = nodeMetadata?.project?.name || nodeMetadata?.project?.slug || configuration?.project;
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }

  return metadata.slice(0, 3);
}
