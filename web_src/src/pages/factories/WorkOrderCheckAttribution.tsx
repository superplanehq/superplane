import { formatRelative } from "@/lib/datetime";

import type { WorkOrderCheckPresentation } from "./lib/workOrderChecks";

/** Who reported the check and when, e.g. "PR Risk Review · 2 hours ago". */
export function WorkOrderCheckAttribution({ check }: { check: WorkOrderCheckPresentation }) {
  const parts = [check.sourceName, check.updatedAt ? formatRelative(check.updatedAt) : undefined].filter(Boolean);
  return <>{parts.join(" · ")}</>;
}
