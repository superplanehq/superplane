import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import type { ComponentBaseContext, ComponentBaseMapper, NodeInfo, SubtitleContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections, getDeleteWorkerExecutionDetails, workerScriptDisplayLabel } from "./base";

interface DeleteWorkerConfiguration {
  workerScript?: string;
  force?: boolean;
}

export const deleteWorkerMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cloudflare";

    return {
      iconSrc: cloudflareIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails: getDeleteWorkerExecutionDetails,

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as DeleteWorkerConfiguration;

  const scriptLabel = workerScriptDisplayLabel(node, configuration?.workerScript);
  if (scriptLabel) {
    metadata.push({ icon: "trash-2", label: scriptLabel });
  }

  if (configuration?.force === true) {
    metadata.push({ icon: "alert-triangle", label: "Force" });
  }

  return metadata;
}
