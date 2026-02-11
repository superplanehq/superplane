import {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../types";
import {
  ComponentBaseProps,
  ComponentBaseSpec,
  DEFAULT_EVENT_STATE_MAP,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { CanvasesCanvasNodeExecution } from "@/api-client";
import { getTriggerRenderer } from "..";

interface BuildkiteBuildInfo {
  id?: string;
  number?: number;
  web_url?: string;
  state?: string;
  result?: string;
  blocked?: boolean;
  branch?: string;
  commit?: string;
  message?: string;
  done_at?: string;
}

interface BuildkitePipelineInfo {
  id?: string;
  slug?: string;
  name?: string;
}

interface BuildkiteOrganizationInfo {
  id?: string;
  slug?: string;
  name?: string;
}

interface BuildkiteSenderInfo {
  id?: string;
  name?: string;
  email?: string;
}

type BuildkiteEventData = {
  build?: BuildkiteBuildInfo;
  pipeline?: BuildkitePipelineInfo;
  organization?: BuildkiteOrganizationInfo;
  sender?: BuildkiteSenderInfo;
};

interface TriggerBuildExecutionMetadata {
  extra?: BuildkiteEventData;
  blocked?: boolean;
  build_id?: string;
  build_number?: number;
  organization?: string;
  pipeline?: string;
  state?: string;
  web_url?: string;
}

interface TriggerBuildNodeMetadataValue {
  name?: string;
}

interface TriggerBuildNodeMetadata {
  organization?: TriggerBuildNodeMetadataValue;
  pipeline?: TriggerBuildNodeMetadataValue;
}

interface BuildkiteEnvironmentVariable {
  name?: string;
  value?: string;
}

interface BuildkiteMetadataItem {
  name?: string;
  value?: string;
}

interface TriggerBuildConfiguration {
  organization?: string;
  pipeline?: string;
  branch?: string;
  commit?: string;
  message?: string;
  env?: BuildkiteEnvironmentVariable[];
  meta_data?: BuildkiteMetadataItem[];
}

type OutputPayloadMap = {
  passed?: OutputPayload[];
  failed?: OutputPayload[];
  default?: OutputPayload[];
};

type BuildkiteExecutionPayload = BuildkiteEventData | { data?: BuildkiteEventData };

export const TRIGGER_BUILD_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  running: {
    icon: "loader-circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
  },
  passed: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};

/**
 * Buildkite-specific state logic function
 */
export const triggerBuildStateFunction: StateFunction = (execution: CanvasesCanvasNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  // If build is still running
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  const metadata = execution.metadata as TriggerBuildExecutionMetadata;
  const buildState = metadata?.state;
  // const blocked = metadata?.blocked;

  if (buildState === "passed") {
    return "passed";
  }

  // All other states including failed, canceled, skipped, not_run, or blocked
  return "failed";
};

/**
 * Buildkite-specific state registry
 */
export const TRIGGER_BUILD_STATE_REGISTRY: EventStateRegistry = {
  stateMap: TRIGGER_BUILD_STATE_MAP,
  getState: triggerBuildStateFunction,
};

export const triggerBuildMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconColor: getColorClass(context.componentDefinition?.color || "gray"),
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: lastExecution ? triggerBuildEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: triggerBuildMetadataList(context.node),
      specs: triggerBuildSpecs(context.node),
      eventStateMap: TRIGGER_BUILD_STATE_MAP,
    };
  },
  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as OutputPayloadMap | undefined;
    const payload =
      (outputs?.passed?.[0]?.data as BuildkiteExecutionPayload | undefined) ??
      (outputs?.failed?.[0]?.data as BuildkiteExecutionPayload | undefined) ??
      (outputs?.default?.[0]?.data as BuildkiteExecutionPayload | undefined);
    const payloadData = unwrapBuildkiteEventData(payload);
    const metadata = context.execution.metadata as TriggerBuildExecutionMetadata | undefined;
    const sourceData = payloadData ?? metadata?.extra;

    if (!sourceData) {
      return details;
    }

    const build = sourceData.build;
    const pipeline = sourceData.pipeline;
    const organization = sourceData.organization;
    const sender = sourceData.sender;

    const addDetail = (key: string, value?: string) => {
      if (value) {
        details[key] = value;
      }
    };

    addDetail("Done At", build?.done_at ? new Date(build.done_at).toLocaleString() : undefined);
    addDetail("Build URL", build?.web_url ?? metadata?.web_url);
    addDetail("Build Number", build?.number?.toString() ?? metadata?.build_number?.toString());
    addDetail("Build State", build?.state ?? metadata?.state);
    addDetail("Pipeline", pipeline?.name ?? metadata?.pipeline);
    addDetail("Organization", organization?.name ?? metadata?.organization);
    addDetail("Branch", build?.branch);
    addDetail("Commit", build?.commit ? `${build.commit.toString().slice(0, 7)} · ${build.message || ""}` : undefined);
    addDetail("Triggered By", sender?.name);

    const blocked = build?.blocked ?? metadata?.blocked;
    if (blocked === true) {
      addDetail("Blocked", "Yes");
    }

    return details;
  },
};

function triggerBuildMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as TriggerBuildConfiguration | undefined;
  const nodeMetadata = node.metadata as TriggerBuildNodeMetadata | undefined;

  const organizationName = nodeMetadata?.organization?.name;
  const pipelineName = nodeMetadata?.pipeline?.name;

  if (organizationName) {
    metadata.push({ icon: "folder", label: organizationName });
  } else if (configuration?.organization) {
    metadata.push({ icon: "folder", label: configuration.organization });
  }

  if (pipelineName) {
    metadata.push({ icon: "git-branch", label: pipelineName });
  } else if (configuration?.pipeline) {
    metadata.push({ icon: "git-branch", label: configuration.pipeline });
  }

  if (configuration?.branch) {
    metadata.push({ icon: "git-branch", label: configuration.branch });
  }

  if (configuration?.commit) {
    metadata.push({ icon: "git-commit", label: configuration.commit.slice(0, 7) });
  }

  return metadata;
}

function triggerBuildSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as TriggerBuildConfiguration | undefined;

  if (configuration?.message) {
    specs.push({
      title: "message",
      iconSlug: "message-square",
      values: [
        {
          badges: [
            {
              label: configuration.message,
              bgColor: "bg-gray-100",
              textColor: "text-gray-800",
            },
          ],
        },
      ],
    });
  }

  // Environment variables
  if (Array.isArray(configuration?.env) && configuration.env.length > 0) {
    specs.push({
      title: "variable",
      iconSlug: "globe",
      values: configuration.env.map((env) => ({
        badges: [
          {
            label: env.name ?? "",
            bgColor: "bg-purple-100",
            textColor: "text-purple-800",
          },
          {
            label: env.value ?? "",
            bgColor: "bg-gray-100",
            textColor: "text-gray-800",
          },
        ],
      })),
    });
  }

  // Metadata
  if (Array.isArray(configuration?.meta_data) && configuration.meta_data.length > 0) {
    specs.push({
      title: "metadata",
      iconSlug: "tag",
      values: configuration.meta_data.map((meta) => ({
        badges: [
          {
            label: meta.name ?? "",
            bgColor: "bg-blue-100",
            textColor: "text-blue-800",
          },
          {
            label: meta.value ?? "",
            bgColor: "bg-gray-100",
            textColor: "text-gray-800",
          },
        ],
      })),
    });
  }

  return specs;
}

function isPayloadWithData(payload: BuildkiteExecutionPayload): payload is { data?: BuildkiteEventData } {
  return typeof payload === "object" && payload !== null && "data" in payload;
}

function unwrapBuildkiteEventData(payload?: BuildkiteExecutionPayload): BuildkiteEventData | undefined {
  if (!payload) {
    return undefined;
  }

  if (isPayloadWithData(payload)) {
    return payload.data;
  }

  return payload;
}

function triggerBuildEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] | undefined {
  if (!execution) {
    return undefined;
  }

  const rootEvent = execution.rootEvent;
  const rootTriggerNode = rootEvent ? nodes.find((n) => n.id === rootEvent.nodeId) : undefined;
  const renderer = rootTriggerNode ? getTriggerRenderer(rootTriggerNode.componentName) : undefined;
  const title = renderer && rootEvent ? renderer.getTitleAndSubtitle({ event: rootEvent }).title : "Unknown";
  const executionState = triggerBuildStateFunction(execution);
  const subtitleTimestamp =
    executionState === "running" ? execution.createdAt : execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : undefined;
  const eventId = rootEvent?.id || execution.id;
  const receivedAt = execution.createdAt ? new Date(execution.createdAt) : undefined;

  if (!eventId) {
    return undefined;
  }

  return [
    {
      receivedAt,
      eventTitle: title,
      eventSubtitle,
      eventState: executionState,
      eventId,
    },
  ];
}
