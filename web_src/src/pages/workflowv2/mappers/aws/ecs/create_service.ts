import { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEcsComponentProps, ecsConsoleUrl, ecsSubtitle, MAX_METADATA_ITEMS } from "./common";

interface CreateServiceConfiguration {
  region?: string;
  cluster?: string;
  serviceName?: string;
  taskDefinition?: string;
  desiredCount?: number;
  launchType?: string;
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
  schedulingStrategy?: string;
}

interface CreateServiceOutput {
  service?: EcsService;
}

export const createServiceMapper: ComponentBaseMapper = {
  props(context) {
    return buildEcsComponentProps(context, createServiceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as CreateServiceOutput | undefined;
    const service = data?.service;
    const timestamp = context.execution.updatedAt
      ? new Date(context.execution.updatedAt).toLocaleString()
      : context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : "-";

    const details: Record<string, string> = {
      "Created At": timestamp,
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

function createServiceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as CreateServiceConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.cluster) {
    items.push({ icon: "server", label: config.cluster });
  }
  if (config?.serviceName) {
    items.push({ icon: "package", label: config.serviceName });
  }
  if (config?.taskDefinition) {
    items.push({ icon: "list", label: config.taskDefinition });
  }
  if (items.length < MAX_METADATA_ITEMS && config?.launchType && config.launchType !== "AUTO") {
    items.push({ icon: "rocket", label: config.launchType });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
