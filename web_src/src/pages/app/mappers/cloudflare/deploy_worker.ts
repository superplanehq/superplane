import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import type { ComponentBaseContext, ComponentBaseMapper, NodeInfo, SubtitleContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections, getDeployWorkerExecutionDetails } from "./base";

interface DeployWorkerConfiguration {
  scriptName?: string;
  source?: string;
  provisionIfMissing?: boolean;
  provision?: Record<string, unknown>;
}

export const deployWorkerMapper: ComponentBaseMapper = {
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

  getExecutionDetails: getDeployWorkerExecutionDetails,

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

const maxMetadataItems = 3;

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as DeployWorkerConfiguration;

  if (configuration?.scriptName) {
    metadata.push({ icon: "code", label: configuration.scriptName });
  }

  if (configuration?.source === "url") {
    metadata.push({ icon: "link", label: "From URL" });
  } else if (configuration?.source === "inline" || configuration?.source === undefined) {
    metadata.push({ icon: "file-text", label: "Inline" });
  }

  const provisionOn = configuration?.provisionIfMissing !== false;
  metadata.push({ icon: "package", label: provisionOn ? "Provision on" : "Provision off" });

  return metadata.slice(0, maxMetadataItems);
}
