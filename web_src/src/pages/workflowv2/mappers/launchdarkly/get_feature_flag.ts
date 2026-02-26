import { ComponentBaseProps } from "@/ui/componentBase";
import { ComponentBaseMapper, ComponentBaseContext, SubtitleContext, ExecutionDetailsContext, OutputPayload, NodeInfo } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap } from "..";
import launchdarklyIcon from "@/assets/icons/integrations/launchdarkly.svg";
import { buildSubtitle } from "../utils";

interface GetFeatureFlagConfiguration {
  projectKey?: string;
  flagKey?: string;
}

interface FeatureFlagOutput {
  key?: string;
  name?: string;
  description?: string;
  kind?: string;
  archived?: boolean;
  temporary?: boolean;
  creationDate?: number;
}

function getFeatureFlagMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetFeatureFlagConfiguration | undefined;

  if (configuration?.projectKey) {
    metadata.push({ icon: "folder", label: configuration.projectKey });
  }

  if (configuration?.flagKey) {
    metadata.push({ icon: "flag", label: configuration.flagKey });
  }

  return metadata;
}

export const getFeatureFlagMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { node, componentDefinition, lastExecutions } = context;
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.componentName || "unknown";

    return {
      iconSrc: launchdarklyIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || "Get Feature Flag",
      metadata: getFeatureFlagMetadata(node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
      eventSections: lastExecution
        ? [
            {
              receivedAt: new Date(lastExecution.createdAt!),
              eventState: getState(componentName)(lastExecution),
              eventTitle: "Flag retrieved",
              eventSubtitle: buildSubtitle("", lastExecution.updatedAt || lastExecution.createdAt),
              eventId: lastExecution.rootEvent?.id || "",
            },
          ]
        : undefined,
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildSubtitle("", context.execution.updatedAt || context.execution.createdAt);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default?.length) {
      return details;
    }

    const flag = outputs.default[0].data as FeatureFlagOutput;
    if (!flag) return details;

    if (flag.key) details["Key"] = flag.key;
    if (flag.name) details["Name"] = flag.name;
    if (flag.description) details["Description"] = flag.description;
    if (flag.kind) details["Kind"] = flag.kind;
    if (flag.archived !== undefined) details["Archived"] = flag.archived ? "Yes" : "No";
    if (flag.temporary !== undefined) details["Temporary"] = flag.temporary ? "Yes" : "No";
    if (flag.creationDate) details["Created At"] = new Date(flag.creationDate).toLocaleString();

    return details;
  },
};
