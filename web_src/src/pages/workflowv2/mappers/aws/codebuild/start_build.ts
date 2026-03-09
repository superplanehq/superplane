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
} from "../../types";
import {
  ComponentBaseProps,
  DEFAULT_EVENT_STATE_MAP,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getTriggerRenderer } from "../..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../../utils";
import { defaultStateFunction } from "../../stateRegistry";
import awsCodeBuildIcon from "@/assets/icons/integrations/aws.codebuild.svg";

interface StartBuildConfiguration {
  region?: string;
  project?: string;
}

interface StartBuildMetadata {
  project?: {
    name?: string;
    region?: string;
  };
}

interface StartBuildOutput {
  build?: {
    project?: string;
    id?: string;
    status?: string;
  };
}

export const START_BUILD_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  passed: DEFAULT_EVENT_STATE_MAP.success,
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};

export const startBuildStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as { failed?: OutputPayload[] } | undefined;
  if (outputs?.failed && outputs.failed.length > 0) {
    return "failed";
  }

  const state = defaultStateFunction(execution);
  return state === "success" ? "passed" : state;
};

export const START_BUILD_STATE_REGISTRY: EventStateRegistry = {
  stateMap: START_BUILD_STATE_MAP,
  getState: startBuildStateFunction,
};

export const startBuildMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsCodeBuildIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getMetadataList(context.node),
      eventStateMap: START_BUILD_STATE_MAP,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const passedOutputs = context.execution.outputs as { passed?: OutputPayload[] } | undefined;
    const failedOutputs = context.execution.outputs as { failed?: OutputPayload[] } | undefined;
    const result =
      (passedOutputs?.passed?.[0]?.data as StartBuildOutput | undefined) ||
      (failedOutputs?.failed?.[0]?.data as StartBuildOutput | undefined);

    const details: Record<string, string> = {
      "Started At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    if (!result?.build) {
      return details;
    }

    details["Project"] = stringOrDash(result.build.project);
    details["Build ID"] = stringOrDash(result.build.id);
    details["Status"] = stringOrDash(result.build.status);

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as StartBuildMetadata | undefined;
  const configuration = node.configuration as StartBuildConfiguration | undefined;

  const projectName = nodeMetadata?.project?.name || configuration?.project;
  if (projectName) {
    metadata.push({ icon: "hammer", label: projectName });
  }

  const region = nodeMetadata?.project?.region || configuration?.region;
  if (region) {
    metadata.push({ icon: "globe", label: region });
  }

  return metadata;
}

function getEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: startBuildStateFunction(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}
