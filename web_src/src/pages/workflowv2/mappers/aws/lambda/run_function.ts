import {
  ComponentsComponent,
  ComponentsNode,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, OutputPayload } from "../../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import awsLambdaIcon from "@/assets/icons/integrations/aws.lambda.svg";

interface RunFunctionConfiguration {
  functionArn?: string;
  payload?: string;
}

interface RunFunctionMetadata {
  functionArn?: string;
}

interface RunFunctionOutput {
  requestId: string;
  payload?: any;
  payloadRaw?: string;
  functionError?: string;
  report?: {
    duration: string;
    billedDuration: string;
    memorySize: string;
    maxMemoryUsed: string;
    initDuration: string;
  };
}

export const runFunctionMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    _items?: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.component?.name || "unknown";

    return {
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      iconSrc: awsLambdaIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      eventSections: lastExecution ? runFunctionEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: runFunctionMetadataList(node),
      specs: runFunctionSpecs(node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as RunFunctionOutput | undefined;
    if (!result) {
      return {};
    }

    let details: Record<string, string> = {
      "Request ID": stringOrDash(result.requestId),
      Duration: stringOrDash(result.report?.duration),
      "Billed Duration": stringOrDash(result.report?.billedDuration),
      "Memory Size": stringOrDash(result.report?.memorySize),
      "Max Memory Used": stringOrDash(result.report?.maxMemoryUsed),
      "Init Duration": stringOrDash(result.report?.initDuration),
    };

    if (result.functionError) {
      details["Function Error"] = stringOrDash(result.functionError);
    }

    return details;
  },

  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    if (!execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function runFunctionMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as RunFunctionMetadata | undefined;
  const configuration = node.configuration as RunFunctionConfiguration | undefined;

  const functionArn = nodeMetadata?.functionArn || configuration?.functionArn;
  const functionName = getFunctionName(functionArn);
  if (functionName) {
    metadata.push({ icon: "code", label: functionName });
  }

  return metadata;
}

function runFunctionSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as RunFunctionConfiguration | undefined;

  if (configuration?.payload) {
    specs.push({
      title: "payload",
      tooltipTitle: "payload",
      iconSlug: "file-text",
      value: configuration.payload,
      contentType: "text",
    });
  }

  return specs;
}

function runFunctionEventSections(
  nodes: ComponentsNode[],
  execution: CanvasesCanvasNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id!,
    },
  ];
}

function getFunctionName(functionArn?: string): string | undefined {
  const arn = (functionArn || "").trim();
  if (!arn) {
    return undefined;
  }

  const marker = "function:";
  const index = arn.indexOf(marker);
  if (index === -1) {
    return arn;
  }

  const suffix = arn.slice(index + marker.length);
  const name = suffix.split(":")[0];
  return name || arn;
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}
