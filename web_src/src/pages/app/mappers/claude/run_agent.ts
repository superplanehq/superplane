import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import claudeIcon from "@/assets/icons/integrations/claude.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

type SessionArtifact = {
  fileId?: string;
  filename?: string;
  mimeType?: string;
  sizeBytes?: number;
  downloadUrl?: string;
};

type RunAgentPayloadData = {
  status?: string;
  sessionId?: string;
  lastMessage?: string;
  messages?: unknown[];
  artifacts?: SessionArtifact[];
  parsed?: unknown;
};

type RunAgentNodeMetadata = {
  structuredOutput?: boolean;
};

type RunAgentConfiguration = {
  outputSchema?: string;
};

function addDetail(details: Record<string, string>, key: string, value: string | undefined) {
  if (value) {
    details[key] = value;
  }
}

// formatArtifacts joins the names of the files the agent produced, falling
// back to the file id when a name is missing. Returns undefined when the run
// produced no artifacts so the entry is omitted.
function formatArtifacts(artifacts?: SessionArtifact[]): string | undefined {
  const names = (artifacts ?? [])
    .map((artifact) => artifact.filename || artifact.fileId)
    .filter((name): name is string => Boolean(name));
  return names.length > 0 ? names.join(", ") : undefined;
}

// formatParsed renders the structured-output JSON extracted from the agent's
// final message. Best-effort: the backend only sets this when a schema is
// configured and the session finished normally, so it's absent otherwise.
function formatParsed(parsed?: unknown): string | undefined {
  if (parsed === undefined || parsed === null) {
    return undefined;
  }
  try {
    return JSON.stringify(parsed);
  } catch {
    return undefined;
  }
}

function hasSchema(schema: unknown): boolean {
  return typeof schema === "string" && schema.trim().length > 0;
}

// metadataList shows the structured-output badge on the canvas node tile. It's
// derived from the live configuration when available, since autosave updates
// configuration only, not metadata (which can lag behind).
function metadataList(node: NodeInfo): MetadataItem[] {
  const items: MetadataItem[] = [];
  const meta = node.metadata as RunAgentNodeMetadata | undefined;
  const config = node.configuration as RunAgentConfiguration | undefined;

  const structured = config ? hasSchema(config.outputSchema) : Boolean(meta?.structuredOutput);
  if (structured) {
    items.push({ icon: "braces", label: "Structured output" });
  }

  return items;
}

export const runAgentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "claude";

    return {
      iconSrc: claudeIcon,
      iconSlug: context.componentDefinition?.icon ?? "bot",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Run Managed Agent",
      eventSections: lastExecution ? runAgentEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    if (timestamp) {
      details["Executed At"] = new Date(timestamp).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = (outputs?.default?.[0]?.data ?? {}) as RunAgentPayloadData;

    addDetail(details, "Status", data.status);
    addDetail(details, "Session ID", data.sessionId);
    addDetail(details, "Parsed Output", formatParsed(data.parsed));
    addDetail(details, "Artifacts", formatArtifacts(data.artifacts));

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function runAgentEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
