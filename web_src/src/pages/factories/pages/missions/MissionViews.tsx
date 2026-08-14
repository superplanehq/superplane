import { cn } from "@/lib/utils";
import { useState } from "react";
import { Link } from "react-router";

import type { WorkOrderLayoutId, WorkOrderScope } from "../../lib/workOrderListModel";
import { MissionAssignee } from "./MissionAssignee";
import { useMissionAssignment } from "./MissionAssignmentContext";
import { applyMissionScope, visibleMissionItems, type MissionRailItem } from "./missionListModel";
import { factoryMissionDetailPath } from "./missionPaths";
import { MissionStatusValue } from "./MissionStatusValue";

export function MissionViews({
  items,
  layout,
  scope,
  currentUserId,
  organizationId,
  factoryKey,
}: {
  items: MissionRailItem[];
  layout: WorkOrderLayoutId;
  scope: WorkOrderScope;
  currentUserId?: string;
  organizationId: string;
  factoryKey: string;
}) {
  const { closedByMissionId } = useMissionAssignment();
  const [expanded, setExpanded] = useState(false);
  const filtered = applyMissionScope(items, scope, closedByMissionId, currentUserId);
  const { visible, hiddenCount } = visibleMissionItems(filtered, expanded);
  const moreLabel = expanded ? "Show fewer" : `Show ${hiddenCount} more`;
  const showMore = hiddenCount > 0 || expanded;

  if (filtered.length === 0) {
    return null;
  }

  const shared = { items: visible, organizationId, factoryKey };

  return (
    <section aria-label="Missions" data-testid="mission-views">
      {layout === "board" ? (
        <MissionBoardLane
          {...shared}
          totalCount={filtered.length}
          showMore={showMore}
          moreLabel={moreLabel}
          onToggleMore={() => setExpanded((current) => !current)}
        />
      ) : null}
      {layout === "list" ? <MissionListView {...shared} totalCount={filtered.length} /> : null}
      {layout === "table" ? <MissionTableView {...shared} /> : null}
      {layout !== "board" && showMore ? (
        <button
          type="button"
          className="mt-2 text-[12px] text-muted-foreground hover:text-foreground"
          data-testid="mission-show-more"
          onClick={() => setExpanded((current) => !current)}
        >
          {moreLabel}
        </button>
      ) : null}
    </section>
  );
}

function MissionBoardLane({
  items,
  organizationId,
  factoryKey,
  totalCount,
  showMore,
  moreLabel,
  onToggleMore,
}: {
  items: MissionRailItem[];
  organizationId: string;
  factoryKey: string;
  totalCount: number;
  showMore: boolean;
  moreLabel: string;
  onToggleMore: () => void;
}) {
  return (
    <section
      aria-label="Missions"
      className="flex flex-col rounded-lg border border-border/70 bg-card/60 p-2"
      data-testid="mission-board"
    >
      <header className="flex items-center justify-between gap-2 px-2 pb-2">
        <div className="inline-flex items-center gap-2">
          <h2 className="text-[12px] font-semibold uppercase tracking-[0.06em] text-foreground/80">Missions</h2>
          <span className="rounded-full bg-background/70 px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground">
            {totalCount}
          </span>
        </div>
        {showMore ? (
          <button
            type="button"
            className="text-[12px] text-muted-foreground hover:text-foreground"
            data-testid="mission-show-more"
            onClick={onToggleMore}
          >
            {moreLabel}
          </button>
        ) : null}
      </header>

      <ul className="flex gap-2 overflow-x-auto pb-1">
        {items.map((item) => (
          <li key={item.mission.id} className="w-64 shrink-0">
            <MissionBoardCard item={item} organizationId={organizationId} factoryKey={factoryKey} />
          </li>
        ))}
      </ul>
    </section>
  );
}

function MissionBoardCard({
  item,
  organizationId,
  factoryKey,
}: {
  item: MissionRailItem;
  organizationId: string;
  factoryKey: string;
}) {
  return (
    <article className="group relative rounded-md border border-border bg-card p-2.5 shadow-sm transition hover:border-foreground/20 hover:shadow">
      <Link
        to={factoryMissionDetailPath(organizationId, factoryKey, item.mission.id)}
        className="absolute inset-0 z-0 rounded-md"
        data-testid={`mission-card-${item.mission.id}`}
        aria-label={`Open ${item.mission.name}`}
      />
      <div className="relative z-10 pointer-events-none">
        <MissionStatusValue status={item.status} variant="card" />
        <h3 className="mt-1.5 line-clamp-2 text-[13px] font-medium leading-snug text-foreground">
          {item.mission.name}
        </h3>
        <p className="mt-1 text-[11px] text-muted-foreground">{item.progress.label}</p>
        <MissionProgressBar percent={item.progress.percent} className="mt-1.5" />
        <div className="mt-2 flex items-center justify-end">
          <MissionAssignee organizationId={organizationId} assignee={item.mission.assignee} />
        </div>
      </div>
    </article>
  );
}

function MissionListView({
  items,
  organizationId,
  factoryKey,
  totalCount,
}: {
  items: MissionRailItem[];
  organizationId: string;
  factoryKey: string;
  totalCount: number;
}) {
  return (
    <section aria-label="Missions" data-testid="mission-list">
      <header className="mb-1.5 flex items-center gap-2 px-1">
        <h2 className="text-[12px] font-semibold uppercase tracking-[0.06em] text-foreground/80">Missions</h2>
        <span className="rounded-full bg-muted px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground">
          {totalCount}
        </span>
      </header>
      <ul className="divide-y divide-border overflow-hidden rounded-md border border-border bg-card">
        {items.map((item) => (
          <li key={item.mission.id}>
            <article className="group relative flex items-center gap-3 px-3 py-2 transition-colors hover:bg-accent/40">
              <Link
                to={factoryMissionDetailPath(organizationId, factoryKey, item.mission.id)}
                className="absolute inset-0 z-0"
                data-testid={`mission-list-row-${item.mission.id}`}
                aria-label={`Open ${item.mission.name}`}
              />
              <span className="relative z-10 pointer-events-none">
                <MissionStatusValue status={item.status} variant="row" />
              </span>
              <span className="relative z-10 pointer-events-none min-w-0 flex-1 truncate text-[13px] font-medium text-foreground">
                {item.mission.name}
              </span>
              <span className="relative z-10 pointer-events-none hidden shrink-0 text-[11px] text-muted-foreground sm:inline">
                {item.progress.label}
              </span>
              <div className="relative z-10 pointer-events-none hidden w-24 shrink-0 sm:block">
                <MissionProgressBar percent={item.progress.percent} />
              </div>
              <div className="relative z-10 pointer-events-none">
                <MissionAssignee organizationId={organizationId} assignee={item.mission.assignee} />
              </div>
            </article>
          </li>
        ))}
      </ul>
    </section>
  );
}

function MissionTableView({
  items,
  organizationId,
  factoryKey,
}: {
  items: MissionRailItem[];
  organizationId: string;
  factoryKey: string;
}) {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-card" data-testid="mission-table">
      <div className="grid grid-cols-[110px_1fr_auto] items-center gap-3 border-b border-border bg-muted/30 px-3 py-2 text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground md:grid-cols-[110px_1fr_160px_120px_auto]">
        <span>Status</span>
        <span>Mission</span>
        <span className="hidden md:inline">Progress</span>
        <span className="hidden md:inline">Work orders</span>
        <span className="text-right">Assignee</span>
      </div>
      <ul className="divide-y divide-border">
        {items.map((item) => (
          <li key={item.mission.id}>
            <article className="group relative grid grid-cols-[110px_1fr_auto] items-center gap-3 px-3 py-2 transition-colors hover:bg-accent/40 md:grid-cols-[110px_1fr_160px_120px_auto]">
              <Link
                to={factoryMissionDetailPath(organizationId, factoryKey, item.mission.id)}
                className="absolute inset-0 z-0"
                data-testid={`mission-table-row-${item.mission.id}`}
                aria-label={`Open ${item.mission.name}`}
              />
              <span className="relative z-10 pointer-events-none">
                <MissionStatusValue status={item.status} variant="row" />
              </span>
              <span className="relative z-10 pointer-events-none min-w-0 truncate text-[13px] font-medium text-foreground">
                {item.mission.name}
              </span>
              <span className="relative z-10 pointer-events-none hidden min-w-0 md:block">
                <span className="sr-only">{item.progress.label}</span>
                <MissionProgressBar percent={item.progress.percent} />
              </span>
              <span className="relative z-10 pointer-events-none hidden tabular-nums text-[12px] text-muted-foreground md:inline">
                {item.progress.total}
              </span>
              <span className="relative z-10 pointer-events-none justify-self-end">
                <MissionAssignee organizationId={organizationId} assignee={item.mission.assignee} />
              </span>
            </article>
          </li>
        ))}
      </ul>
    </div>
  );
}

function MissionProgressBar({ percent, className }: { percent: number; className?: string }) {
  return (
    <div className={cn("h-1.5 overflow-hidden rounded-full bg-muted", className)} aria-hidden>
      <div className="h-full rounded-full bg-foreground/70" style={{ width: `${percent}%` }} />
    </div>
  );
}
