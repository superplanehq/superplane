import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { formatTimeAgo } from "@/lib/date";
import type { OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";
import { Link } from "react-router";

import { factoryHomePath } from "../../lib/factoryPagePaths";
import type { WorkOrderListEntry } from "../../lib/workOrderListModel";
import { getWorkOrderDisplayStatusMeta } from "../../lib/workOrderProgress";
import { OrgUserReference } from "../../OrgUserReference";

export function MissionWorkOrderList({
  entries,
  organizationId,
  factoryKey,
}: {
  entries: WorkOrderListEntry[];
  organizationId: string;
  factoryKey: string;
}) {
  const { resolveUser } = useOrgUserLookup(organizationId);

  return (
    <ul data-testid="mission-work-order-list">
      {entries.map((entry) => (
        <MissionWorkOrderRow
          key={entry.id}
          entry={entry}
          organizationId={organizationId}
          factoryKey={factoryKey}
          resolveUser={resolveUser}
        />
      ))}
    </ul>
  );
}

function MissionWorkOrderRow({
  entry,
  organizationId,
  factoryKey,
  resolveUser,
}: {
  entry: WorkOrderListEntry;
  organizationId: string;
  factoryKey: string;
  resolveUser: OrgUserDisplayLookup;
}) {
  const meta = getWorkOrderDisplayStatusMeta(entry.displayStatus);
  const href = factoryHomePath(organizationId, factoryKey);
  const timeLabel = entry.updatedAtMs > 0 ? formatTimeAgo(new Date(entry.updatedAtMs)) : "—";
  const assignee = entry.order.assignees?.[0];

  return (
    <li>
      <Link
        to={href}
        className="flex items-center gap-3 py-2 hover:bg-accent/30"
        data-testid={`mission-work-order-row-${entry.id}`}
        aria-label={`Open ${entry.title}`}
      >
        <span className={cn("size-1.5 shrink-0 rounded-full", meta.dotClassName)} aria-hidden />
        <span className="w-[54px] shrink-0 font-mono text-[11px] text-muted-foreground">{entry.displayKey}</span>
        <span className="min-w-0 flex-1 truncate text-[13px] text-foreground">{entry.title}</span>
        <span className="hidden shrink-0 text-[11px] text-muted-foreground sm:inline">{timeLabel}</span>
        {assignee ? (
          <OrgUserReference
            display={resolveUser(assignee.id, assignee.name)}
            size="xs"
            showName={false}
            className="shrink-0"
          />
        ) : null}
      </Link>
    </li>
  );
}
