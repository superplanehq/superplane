import { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildEcsComponentProps, ecsSubtitle } from "./common";

interface UpdateServiceConfiguration {
  region?: string;
  cluster?: string;
  service?: string;
  taskDefinition?: string;
  desiredCount?: number;
  forceNewDeployment?: boolean;
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

interface UpdateServiceOutput {
  service?: EcsService;
}

export const updateServiceMapper: ComponentBaseMapper = {
  props(context) {
    return buildEcsComponentProps(context, updateServiceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as UpdateServiceOutput | undefined;
    const service = data?.service;

    if (!service) {
      return {};
    }

    return {
      "Updated At": stringOrDash(
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

function updateServiceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as UpdateServiceConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }
  if (config?.cluster) {
    items.push({ icon: "server", label: config.cluster });
  }
  if (config?.service) {
    items.push({ icon: "package", label: config.service });
  }
  if (config?.taskDefinition) {
    items.push({ icon: "list", label: config.taskDefinition });
  }
  if (config?.forceNewDeployment) {
    items.push({ icon: "refresh-cw", label: "force new deployment" });
  }

  return items;
}
