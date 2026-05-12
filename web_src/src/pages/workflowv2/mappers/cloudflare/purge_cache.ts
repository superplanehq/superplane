import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./base";

interface PurgeCacheConfiguration {
  zone?: string;
  mode?: string;
  files?: string[];
  tags?: string[];
  hosts?: string[];
}

interface PurgeCacheOutput {
  zoneId?: string;
  zoneName?: string;
  id?: string;
  mode?: string;
  files?: string[];
  tags?: string[];
  hosts?: string[];
}

export const purgeCacheMapper: ComponentBaseMapper = {
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

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as PurgeCacheOutput | undefined;
    if (!result) return details;

    if (result.mode) details["Mode"] = result.mode;
    const zoneLabel = purgeCacheZoneLabel(result);
    if (zoneLabel) details["Zone"] = zoneLabel;
    if (result.files?.length) details["Files"] = String(result.files.length);
    if (result.tags?.length) details["Tags"] = String(result.tags.length);
    if (result.hosts?.length) details["Hosts"] = String(result.hosts.length);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function purgeCacheZoneLabel(result: PurgeCacheOutput): string | undefined {
  return result.zoneName?.trim() || result.zoneId;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as PurgeCacheConfiguration | undefined;

  const mode = config?.mode;
  if (mode === "everything") {
    metadata.push({ icon: "zap", label: "Purge everything" });
  } else if (mode === "files" && config?.files?.length) {
    metadata.push({ icon: "link", label: `${config.files.length} URL${config.files.length > 1 ? "s" : ""}` });
  } else if (mode === "tags" && config?.tags?.length) {
    metadata.push({ icon: "tag", label: `${config.tags.length} tag${config.tags.length > 1 ? "s" : ""}` });
  } else if (mode === "hosts" && config?.hosts?.length) {
    metadata.push({ icon: "server", label: `${config.hosts.length} host${config.hosts.length > 1 ? "s" : ""}` });
  } else if (mode) {
    metadata.push({ icon: "zap", label: mode });
  }

  return metadata;
}
