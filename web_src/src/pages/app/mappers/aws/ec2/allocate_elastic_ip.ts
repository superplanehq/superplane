import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../../utils";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";

interface Configuration {
  region?: string;
  ipSource?: string;
}

interface AllocateElasticIPNodeMetadata {
  region?: string;
  ipSource?: string;
}

const ipSourceLabels: Record<string, string> = {
  amazon: "Amazon pool",
  byoip: "BYOIP pool",
  customerOwned: "Customer-owned pool",
  ipam: "IPAM pool",
};

interface Output {
  allocationId?: string;
  publicIp?: string;
  domain?: string;
  region?: string;
}

export const allocateElasticIPMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? allocateElasticIPEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: allocateElasticIPMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Output | undefined;
    const completedAt = formatCompletedAt(context.execution);

    if (!output) {
      return {
        "Completed At": stringOrDash(completedAt),
        Region: stringOrDash(configuration?.region),
        "Public IP": "-",
      };
    }

    return {
      "Completed At": stringOrDash(completedAt),
      Region: stringOrDash(output.region ?? configuration?.region),
      "Public IP": stringOrDash(output.publicIp),
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

function allocateElasticIPMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as AllocateElasticIPNodeMetadata | undefined;
  const region = configuration?.region ?? nodeMetadata?.region;
  const ipSource = configuration?.ipSource ?? nodeMetadata?.ipSource ?? "amazon";
  const metadata: MetadataItem[] = [];

  if (ipSource && ipSource !== "amazon") {
    metadata.push({ icon: "layers", label: ipSourceLabels[ipSource] ?? ipSource });
  }

  if (region) {
    metadata.push({ icon: "globe", label: region });
  }

  return metadata;
}

function allocateElasticIPEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
