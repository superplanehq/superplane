import type { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { ExecutionInfo, NodeInfo, OutputPayload } from "../types";
import type { DropletData } from "./types";

export function findPublicIP(droplet: DropletData): string | undefined {
  return droplet.networks?.v4?.find((n) => n.type === "public")?.ip_address;
}

export function buildBaseDropletDetails(droplet: DropletData): Record<string, string> {
  return {
    "Droplet ID": droplet.id?.toString() || "-",
    Name: droplet.name || "-",
    Status: droplet.status || "-",
    Region: droplet.region?.name || droplet.region?.slug || "-",
    "GPU Size": droplet.size_slug || "-",
    Memory: droplet.memory ? `${droplet.memory} MB` : "-",
    vCPUs: droplet.vcpus?.toString() || "-",
    Disk: droplet.disk ? `${droplet.disk} GB` : "-",
  };
}

export function getDropletFromOutputs(outputs: unknown): DropletData | undefined {
  const typed = outputs as { default?: OutputPayload[] } | undefined;
  return typed?.default?.[0]?.data as DropletData | undefined;
}

export function gpuBaseEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const rootEvent = execution.rootEvent;
  const { title } = rootEvent ? rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent }) : { title: "" };
  const createdAt = execution.createdAt ?? new Date().toISOString();
  const eventId = rootEvent?.id ?? "";

  return [
    {
      receivedAt: new Date(createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(createdAt)),
      eventState: getState(componentName)(execution),
      eventId,
    },
  ];
}
