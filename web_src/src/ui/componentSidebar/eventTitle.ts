import { formatTimestamp } from "@/pages/workflowv2/mappers/utils";
import type { SidebarEvent } from "./types";

export function getSidebarEventTitle(event: Pick<SidebarEvent, "title" | "receivedAt">): string {
  return event.title?.trim() || "Event received at " + formatTimestamp(event.receivedAt?.toISOString() || "");
}
