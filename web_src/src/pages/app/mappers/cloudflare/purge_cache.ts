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
  prefixes?: string[];
}

interface PurgeCacheOutput {
  zoneId?: string;
  zoneName?: string;
  id?: string;
  mode?: string;
  files?: string[];
  tags?: string[];
  hosts?: string[];
  prefixes?: string[];
}

type PurgeItemMode = "files" | "tags" | "hosts" | "prefixes";

const purgeMetadataByMode: Record<PurgeItemMode, { icon: string; singular: string; plural: string }> = {
  files: { icon: "link", singular: "URL", plural: "URLs" },
  tags: { icon: "tag", singular: "tag", plural: "tags" },
  hosts: { icon: "server", singular: "host", plural: "hosts" },
  prefixes: { icon: "folder-tree", singular: "prefix", plural: "prefixes" },
};

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
    addPurgeItemCounts(details, result);

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

function addPurgeItemCounts(details: Record<string, string>, result: PurgeCacheOutput): void {
  const fields: Array<[string, string[] | undefined]> = [
    ["Files", result.files],
    ["Tags", result.tags],
    ["Hosts", result.hosts],
    ["Prefixes", result.prefixes],
  ];

  for (const [label, values] of fields) {
    if (values?.length) details[label] = String(values.length);
  }
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as PurgeCacheConfiguration | undefined;

  const mode = config?.mode;
  if (mode === "everything") {
    return [{ icon: "zap", label: "Purge everything" }];
  }

  const itemMetadata = metadataForItemMode(config, mode);
  if (itemMetadata) return [itemMetadata];
  return mode ? [{ icon: "zap", label: mode }] : [];
}

function metadataForItemMode(
  config: PurgeCacheConfiguration | undefined,
  mode: string | undefined,
): MetadataItem | undefined {
  if (!isPurgeItemMode(mode)) return undefined;

  const items = config?.[mode];
  if (!items?.length) return undefined;

  const metadata = purgeMetadataByMode[mode];
  const label = items.length === 1 ? metadata.singular : metadata.plural;
  return { icon: metadata.icon, label: `${items.length} ${label}` };
}

function isPurgeItemMode(mode: string | undefined): mode is PurgeItemMode {
  return mode === "files" || mode === "tags" || mode === "hosts" || mode === "prefixes";
}
