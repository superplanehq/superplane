import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import type { ComponentBaseContext, ComponentBaseMapper, NodeInfo, SubtitleContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections, getWorkerRouteExecutionDetails, workerScriptDisplayLabel } from "./base";

interface UpdateWorkerRouteConfiguration {
  zone?: string;
  routeId?: string;
  pattern?: string;
  workerScript?: string;
}

export const updateWorkerRouteMapper: ComponentBaseMapper = {
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

  getExecutionDetails: getWorkerRouteExecutionDetails,

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpdateWorkerRouteConfiguration;

  if (configuration?.pattern) {
    metadata.push({ icon: "route", label: configuration.pattern });
  }

  const scriptLabel = workerScriptDisplayLabel(node, configuration?.workerScript);
  if (scriptLabel) {
    metadata.push({ icon: "code", label: scriptLabel });
  }

  if (configuration?.routeId) {
    metadata.push({ icon: "edit", label: "Update" });
  } else {
    metadata.push({ icon: "plus", label: "Create" });
  }

  return metadata;
}
