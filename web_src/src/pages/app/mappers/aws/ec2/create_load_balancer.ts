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
import type { ElbLoadBalancer } from "./types";

interface Configuration {
  name?: string;
  region?: string;
  type?: string;
  scheme?: string;
  subnets?: string[];
}

interface CreateLoadBalancerNodeMetadata {
  region?: string;
  name?: string;
  type?: string;
  scheme?: string;
}

export const createLoadBalancerMapper: ComponentBaseMapper = {
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
        ? createLoadBalancerEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: createLoadBalancerMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const nodeMetadata = context.node.metadata as CreateLoadBalancerNodeMetadata | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as ElbLoadBalancer | undefined;
    const createdAt = context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : undefined;
    const details = resolveLoadBalancerDetails(output, nodeMetadata, configuration);

    return {
      "Created At": stringOrDash(createdAt),
      Name: stringOrDash(details.name),
      Region: stringOrDash(details.region),
      Type: stringOrDash(details.type),
      Scheme: stringOrDash(details.scheme),
      State: stringOrDash(details.state),
      "DNS Name": stringOrDash(details.dnsName),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function resolveLoadBalancerDetails(
  output: ElbLoadBalancer | undefined,
  nodeMetadata: CreateLoadBalancerNodeMetadata | undefined,
  configuration: Configuration | undefined,
) {
  const out: ElbLoadBalancer = output ?? {};
  const meta: CreateLoadBalancerNodeMetadata = nodeMetadata ?? {};
  const cfg: Configuration = configuration ?? {};
  return {
    name: out.name ?? meta.name ?? cfg.name,
    region: out.region ?? cfg.region ?? meta.region,
    type: out.type ?? cfg.type ?? meta.type,
    scheme: out.scheme ?? cfg.scheme ?? meta.scheme,
    state: out.state,
    dnsName: out.dnsName,
  };
}

function createLoadBalancerMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as CreateLoadBalancerNodeMetadata | undefined;
  const metadata: MetadataItem[] = [];

  const name = nodeMetadata?.name ?? configuration?.name;
  if (name) {
    metadata.push({ icon: "server", label: name });
  }

  const type = nodeMetadata?.type ?? configuration?.type;
  if (type) {
    metadata.push({ icon: "layers", label: type });
  }

  const region = configuration?.region ?? nodeMetadata?.region;
  if (region) {
    metadata.push({ icon: "globe", label: region });
  }

  return metadata;
}

function createLoadBalancerEventSections(
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
