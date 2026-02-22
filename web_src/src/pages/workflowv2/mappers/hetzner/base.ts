import { ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { noopMapper } from "../noop";
import { formatTimeAgo } from "@/utils/date";

function getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};
  const metadata = context.execution.metadata as Record<string, unknown> | undefined;

  const serverId = metadata?.serverId ?? (metadata?.server as Record<string, unknown> | undefined)?.id;
  const loadBalancerId =
    metadata?.loadBalancerId ?? (metadata?.loadBalancer as Record<string, unknown> | undefined)?.id;

  if (serverId !== undefined) {
    details["Server ID"] = String(serverId);
  }

  if (loadBalancerId !== undefined) {
    details["Load Balancer ID"] = String(loadBalancerId);
  }

  if (context.execution.createdAt) {
    details["Started at"] = new Date(context.execution.createdAt).toLocaleString();
  }
  if (context.execution.updatedAt && context.execution.state === "STATE_FINISHED") {
    details["Finished at"] = new Date(context.execution.updatedAt).toLocaleString();
  }

  if (context.execution.resultMessage) {
    details["Error"] = context.execution.resultMessage;
  }

  return details;
}

function subtitle(context: SubtitleContext): string {
  if (!context.execution.createdAt) return "";
  return formatTimeAgo(new Date(context.execution.createdAt));
}

export const hetznerBaseMapper: ComponentBaseMapper = {
  ...noopMapper,
  getExecutionDetails: getExecutionDetails,
  subtitle: subtitle,
};
