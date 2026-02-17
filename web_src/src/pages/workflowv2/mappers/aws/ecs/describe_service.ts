import { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEcsComponentProps, ecsConsoleUrl, ecsSubtitle, MAX_METADATA_ITEMS } from "./common";

interface DescribeServiceConfiguration {
  region?: string;
  cluster?: string;
  service?: string;
}

interface EcsFailure {
  arn?: string;
  reason?: string;
  detail?: string;
}

interface EcsService {
  serviceArn?: string;
  serviceName?: string;
  clusterArn?: string;
  status?: string;
  taskDefinition?: string;
  desiredCount?: number;
  runningCount?: number;
  pendingCount?: number;
  launchType?: string;
  platformVersion?: string;
}

interface DescribeServiceOutput {
  service?: EcsService;
  failures?: EcsFailure[];
}

export const describeServiceMapper: ComponentBaseMapper = {
  props(context) {
    return buildEcsComponentProps(context, describeServiceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as DescribeServiceOutput | undefined;
    const service = data?.service;
    const timestamp = context.execution.updatedAt
      ? new Date(context.execution.updatedAt).toLocaleString()
      : context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : "-";

    const details: Record<string, string> = {
      "Retrieved At": timestamp,
    };
    if (service) {
      details["Service"] = stringOrDash(service.serviceName);
      details["Status"] = stringOrDash(service.status);
      details["Cluster"] = stringOrDash(service.clusterArn);
      details["Task Definition"] = stringOrDash(service.taskDefinition);
      const region = service.clusterArn?.split(":")[2] ?? "";
      const cluster = service.clusterArn?.split("/").pop() ?? "";
      if (region && cluster && service.serviceName) {
        details["ECS Console"] = ecsConsoleUrl(region, cluster, service.serviceName);
      }
    }
    return details;
  },

  subtitle(context: SubtitleContext): string {
    return ecsSubtitle(context);
  },
};

function describeServiceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as DescribeServiceConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.cluster) {
    items.push({ icon: "server", label: config.cluster });
  }
  if (config?.service) {
    items.push({ icon: "package", label: config.service });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
