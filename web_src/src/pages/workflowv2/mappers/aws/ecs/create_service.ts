import { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEcsComponentProps, ecsSubtitle } from "./common";

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

    if (!service) {
      return {};
    }

    return {
      "Created At": stringOrDash(
        context.execution.updatedAt ? new Date(context.execution.updatedAt).toLocaleString() : "-",
      ),
      Service: stringOrDash(service.serviceName),
      "Service ARN": stringOrDash(service.serviceArn),
      Status: stringOrDash(service.status),
      Cluster: stringOrDash(service.clusterArn),
      "Task Definition": stringOrDash(service.taskDefinition),
      "Desired Count": stringOrDash(service.desiredCount),
      "Running Count": stringOrDash(service.runningCount),
      "Pending Count": stringOrDash(service.pendingCount),
      "Launch Type": stringOrDash(service.launchType),
      "Platform Version": stringOrDash(service.platformVersion),
      "Scheduling Strategy": stringOrDash(service.schedulingStrategy),
    };
  },

  subtitle(context: SubtitleContext): string {
    return ecsSubtitle(context);
  },
};

function createServiceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as CreateServiceConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }
  if (config?.cluster) {
    items.push({ icon: "server", label: config.cluster });
  }
  if (config?.serviceName) {
    items.push({ icon: "package", label: config.serviceName });
  }
  if (config?.taskDefinition) {
    items.push({ icon: "list", label: config.taskDefinition });
  }
  if (config?.launchType && config.launchType !== "AUTO") {
    items.push({ icon: "rocket", label: config.launchType });
  }

  return items;
}
