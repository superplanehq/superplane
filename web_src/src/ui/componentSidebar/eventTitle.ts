import { getEventTitleFallback } from "@/pages/workflowv2/utils";
import type { SidebarEvent } from "./types";

export function getSidebarEventTitle(event: Pick<SidebarEvent, "title" | "receivedAt">): string {
  return event.title?.trim() || getEventTitleFallback(event.receivedAt?.toISOString());
}
