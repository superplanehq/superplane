import type { ComponentBaseProps, EventSection, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getTriggerRenderer } from "../..";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../../utils";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";
import { defaultStateFunction } from "../../stateRegistry";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";

interface Configuration {
  region?: string;
  operation?: string;
  allocationId?: string;
  instance?: string;
  associationId?: string;
}

interface ManageElasticIPNodeMetadata {
  region?: string;
  operation?: string;
  instanceName?: string;
}

interface Output {
  associationId?: string;
  allocationId?: string;
  instanceId?: string;
  region?: string;
}

const operationLabels: Record<string, string> = {
  associate: "Associate",
  disassociate: "Disassociate",
};

const operationFromPayloadType: Record<string, string> = {
  "aws.ec2.elastic-ip.associated": "associate",
  "aws.ec2.elastic-ip.disassociated": "disassociate",
};

export const elasticIPAssociationStateMap: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  "aws.ec2.elastic-ip.associated": {
    icon: "link",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "ASSOCIATED",
  },
  "aws.ec2.elastic-ip.disassociated": {
    icon: "unlink",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "DISASSOCIATED",
  },
};

const elasticIPPayloadPrefix = "aws.ec2.elastic-ip.";

export const MANAGE_ELASTIC_IP_STATE_REGISTRY: EventStateRegistry = {
  stateMap: elasticIPAssociationStateMap,
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") return state;

    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const elasticIPEvent = outputs?.default?.find((output) => output.type?.startsWith(elasticIPPayloadPrefix));
    if (elasticIPEvent?.type && elasticIPAssociationStateMap[elasticIPEvent.type]) {
      return elasticIPEvent.type;
    }

    return "success";
  },
};

export const manageElasticIPMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? manageElasticIPEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: manageElasticIPMetadata(context.node),
      eventStateMap: elasticIPAssociationStateMap,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const outputEvent = outputs?.default?.find((output) => output.type?.startsWith(elasticIPPayloadPrefix));
    const output = outputEvent?.data as Output | undefined;
    const completedAt = formatCompletedAt(context.execution);
    const operationLabel = operationLabelForExecution(configuration, outputEvent?.type);

    if (!output) {
      return {
        "Completed At": stringOrDash(completedAt),
        Operation: operationLabel,
        Region: stringOrDash(configuration?.region),
      };
    }

    return {
      "Completed At": stringOrDash(completedAt),
      Operation: operationLabel,
      Region: stringOrDash(output.region ?? configuration?.region),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function formatCompletedAt(execution: ExecutionInfo): string | undefined {
  const timestamp = execution.updatedAt ?? execution.createdAt;
  if (!timestamp) {
    return undefined;
  }

  return new Date(timestamp).toLocaleString();
}

function operationLabelForExecution(configuration: Configuration | undefined, payloadType?: string): string {
  const operationFromOutput = payloadType ? operationFromPayloadType[payloadType] : undefined;
  const operation = operationFromOutput ?? configuration?.operation;
  return (operation && operationLabels[operation]) ?? operation ?? "-";
}

function manageElasticIPMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as ManageElasticIPNodeMetadata | undefined;

  const operation = configuration?.operation ?? nodeMetadata?.operation;
  const region = configuration?.region ?? nodeMetadata?.region;
  const instanceName = nodeMetadata?.instanceName ?? configuration?.instance;

  const metadata: MetadataItem[] = [];

  if (operation === "associate" && instanceName) {
    metadata.push({ icon: "server", label: instanceName });
  }

  if (operation) {
    metadata.push({ icon: "link", label: operationLabels[operation] ?? operation });
  }

  if (region) {
    metadata.push({ icon: "globe", label: region });
  }

  return metadata;
}

function manageElasticIPEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: MANAGE_ELASTIC_IP_STATE_REGISTRY.getState(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
