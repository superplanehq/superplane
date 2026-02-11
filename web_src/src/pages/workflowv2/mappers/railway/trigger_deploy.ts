import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { ComponentBaseMapper, ComponentBaseContext, SubtitleContext, ExecutionDetailsContext } from "../types";
import RailwayLogo from "@/assets/icons/integrations/railway.svg";
import { formatTimeAgo } from "@/utils/date";

interface TriggerDeployMetadata {
  project?: { id?: string; name?: string };
  service?: { id?: string; name?: string };
  environment?: { id?: string; name?: string };
}

/**
 * Mapper for the "railway.triggerDeploy" component type
 */
export const triggerDeployMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const { node, componentDefinition } = context;
    const metadata = node.metadata as unknown as TriggerDeployMetadata;
    const metadataItems = [];

    // Show service name
    if (metadata?.service?.name) {
      metadataItems.push({
        icon: "box",
        label: metadata.service.name,
      });
    }

    // Show environment name
    if (metadata?.environment?.name) {
      metadataItems.push({
        icon: "globe",
        label: metadata.environment.name,
      });
    }

    return {
      iconSrc: RailwayLogo,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      title: node.name || componentDefinition.label || "Trigger Deploy",
      metadata: metadataItems,
    };
  },

  subtitle(context: SubtitleContext): string {
    const { execution } = context;

    if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
      const updatedAt = execution.updatedAt ? new Date(execution.updatedAt) : null;
      return updatedAt ? `Triggered ${formatTimeAgo(updatedAt)}` : "Triggered";
    }

    if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_FAILED") {
      const updatedAt = execution.updatedAt ? new Date(execution.updatedAt) : null;
      return updatedAt ? `Failed ${formatTimeAgo(updatedAt)}` : "Failed";
    }

    if (execution.state === "STATE_STARTED") {
      return "Deploying...";
    }

    const createdAt = execution.createdAt ? new Date(execution.createdAt) : null;
    return createdAt ? formatTimeAgo(createdAt) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext) {
    const { execution, node } = context;
    const details: Record<string, string> = {};

    if (execution.createdAt) {
      details["Started At"] = new Date(execution.createdAt).toLocaleString();
    }

    if (execution.updatedAt && execution.state === "STATE_FINISHED") {
      details["Finished At"] = new Date(execution.updatedAt).toLocaleString();
    }

    // Add metadata info
    const metadata = node.metadata as unknown as TriggerDeployMetadata;
    if (metadata?.project?.name) {
      details["Project"] = metadata.project.name;
    }
    if (metadata?.service?.name) {
      details["Service"] = metadata.service.name;
    }
    if (metadata?.environment?.name) {
      details["Environment"] = metadata.environment.name;
    }

    return details;
  },
};
