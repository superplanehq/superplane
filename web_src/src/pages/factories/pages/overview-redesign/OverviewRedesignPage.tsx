import {
  CircleCheck,
  CircleX,
  GitMerge,
  GitPullRequestArrow,
  Loader2,
  MessageCircleQuestion,
  Plus,
  Timer,
  UserPlus,
} from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";

import { Avatar } from "@/components/Avatar/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import {
  createWorkOrderPath,
  factoryVelocityPath,
  workOrderDetailPath,
  workOrdersPath,
} from "../../lib/factoryPagePaths";
import { factorySectionBodyClassName, factorySectionHeaderClassName } from "../factoryPageLayoutStyles";
import { CardEmptyState, CardViewAllLink, OverviewCard } from "./overviewRedesignCardParts";
import { HealthScorecards } from "./HealthScorecards";
import { ImprovementsCard, SuggestionsCard } from "./OverviewRedesignRail";
import { OverviewScopeToggle, type OverviewScope } from "./OverviewScopeToggle";
import type {
  AttentionItem,
  AttentionReason,
  BriefingCounts,
  InFlightItem,
  OverviewRedesignData,
  ShippedItem,
  ShippedOutcome,
  WorkOrderOwner,
} from "./overviewRedesignMocks";

/**
 * Workspace Overview redesign (Storybook-only baseline).
 *
 * Layout: briefing header, a horizontal row of health scorecards, then the
 * work stream — Needs attention and In flight full width, Recently shipped
 * and Suggested work orders paired in one row, and Workspace improvements
 * as a wide card at the bottom. Every table caps at three rows; totals stay
 * visible in the header counts. Future-capability cards carry a "Preview"
 * badge.
 *
 * The All/My toggle in the header scopes the three work order tables to
 * the viewer's assignments. Health metrics and the proposal cards always
 * stay workspace-wide.
 */
export function OverviewRedesignPage({ data }: { data: OverviewRedesignData }) {
  const { organizationId, factoryKey } = useFactoriesLayout();
  const [scope, setScope] = useState<OverviewScope>("all");

  const attention = scopeItems(data.attention, scope);
  const inFlight = scopeItems(data.inFlight, scope);
  const shipped = scopeItems(data.shipped, scope);

  return (
    <>
      <WorkspacePageHeader
        className={factorySectionHeaderClassName}
        title="Overview"
        subtitle={
          data.briefing ? (
            <BriefingLine
              counts={{ attention: attention.length, inFlight: inFlight.length }}
              organizationId={organizationId}
              factoryKey={factoryKey}
            />
          ) : (
            "Your workspace at a glance."
          )
        }
        actions={<OverviewScopeToggle value={scope} onChange={setScope} />}
      />

      <div className={factorySectionBodyClassName}>
        <HealthScorecards metrics={data.health} velocityHref={factoryVelocityPath(organizationId, factoryKey)} />

        <div className="mt-6 flex min-w-0 flex-col gap-6">
          <NeedsAttentionCard
            items={attention}
            organizationId={organizationId}
            factoryKey={factoryKey}
            viewAllHref={workOrdersPath(organizationId, factoryKey)}
          />
          <InFlightCard
            items={inFlight}
            organizationId={organizationId}
            factoryKey={factoryKey}
            viewAllHref={workOrdersPath(organizationId, factoryKey)}
            newWorkOrderHref={createWorkOrderPath(organizationId, factoryKey)}
          />
          <div className="grid gap-6 lg:grid-cols-2">
            <SuggestionsCard suggestions={data.suggestions} />
            <RecentlyShippedCard
              items={shipped}
              organizationId={organizationId}
              factoryKey={factoryKey}
              viewAllHref={workOrdersPath(organizationId, factoryKey)}
            />
          </div>
          <ImprovementsCard proposals={data.improvements} readiness={data.readiness} />
        </div>
      </div>
    </>
  );
}

function scopeItems<T extends { mine?: boolean }>(items: T[], scope: OverviewScope): T[] {
  return scope === "my" ? items.filter((item) => item.mine) : items;
}

/* ------------------------------- Header ------------------------------- */

function BriefingLine({
  counts,
  organizationId,
  factoryKey,
}: {
  counts: BriefingCounts;
  organizationId: string;
  factoryKey: string;
}) {
  const attentionFragment =
    counts.attention === 0 ? (
      <span>Nothing needs attention</span>
    ) : (
      <span className="font-medium text-foreground">
        {counts.attention} {counts.attention === 1 ? "work order needs" : "work orders need"} attention
      </span>
    );

  return (
    <span className="flex flex-wrap items-center gap-x-2 gap-y-1" data-testid="overview-briefing">
      {attentionFragment}
      <BriefingSeparator />
      <Link to={workOrdersPath(organizationId, factoryKey)} className="hover:text-foreground">
        {counts.inFlight} in flight
      </Link>
    </span>
  );
}

function BriefingSeparator() {
  return (
    <span aria-hidden className="text-border">
      ·
    </span>
  );
}

/* ----------------------------- Row helpers ----------------------------- */

/** Overview tables show at most this many rows; header counts keep the total. */
const MAX_OVERVIEW_ROWS = 3;

/**
 * Recently shipped rows are single line, so five of them roughly match the
 * height of three two-line suggestion rows in the paired columns.
 */
const MAX_SHIPPED_ROWS = 5;

/** Detail route from a workspace-scoped key like "SP-61" (mock-only parsing). */
function workOrderHref(organizationId: string, factoryKey: string, workOrderKey: string) {
  const orderNumber = workOrderKey.split("-")[1] ?? "";
  return workOrderDetailPath(organizationId, factoryKey, orderNumber);
}

/**
 * Stretched link that makes the whole row open the work order detail.
 * Row content sits above it with `pointer-events-none`; inline buttons
 * re-enable pointer events. Same pattern as the Work Orders list rows.
 */
function RowLink({ href, label }: { href: string; label: string }) {
  return <Link to={href} className="absolute inset-0 z-0" aria-label={label} />;
}

const rowClassName =
  "group relative flex items-center gap-3 border-b border-border/60 px-4 py-3 transition-colors last:border-b-0 hover:bg-accent/40";

const rowContentClassName = "relative z-10 pointer-events-none";

/* --------------------------- Needs attention --------------------------- */

const ATTENTION_META: Record<
  AttentionReason,
  { label: string; actionLabel: string; icon: typeof CircleCheck; chipClassName: string }
> = {
  approval: {
    label: "Approval needed",
    actionLabel: "Approve",
    icon: CircleCheck,
    chipClassName: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400",
  },
  question: {
    label: "Agent question",
    actionLabel: "Answer",
    icon: MessageCircleQuestion,
    chipClassName: "border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-400",
  },
  failed: {
    label: "Run failed",
    actionLabel: "Retry",
    icon: CircleX,
    chipClassName: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400",
  },
  stalled: {
    label: "No progress",
    actionLabel: "Open",
    icon: Timer,
    chipClassName: "border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-400",
  },
};

/**
 * Owner of the work order: avatar + name (name hides on narrow screens).
 * No owner renders a dashed "Unassigned" chip — in a team-wide queue that
 * marks work anyone can pick up, matching the work orders list pattern.
 */
function OwnerReference({ owner, className }: { owner?: WorkOrderOwner; className?: string }) {
  if (!owner) {
    return (
      <span
        className={cn(
          "inline-flex shrink-0 items-center gap-1 rounded-full border border-dashed border-border px-2 py-0.5 text-[11px] text-muted-foreground",
          className,
        )}
      >
        <UserPlus className="size-3" aria-hidden />
        Unassigned
      </span>
    );
  }
  // The name stays visible on every breakpoint — hover-only reveals fail on
  // touch and behind `pointer-events-none` row content. Pointer events come
  // back on so the title tooltip can expand a truncated name.
  return (
    <span className={cn("flex shrink-0 items-center gap-1.5", className, "pointer-events-auto")} title={owner.name}>
      <Avatar initials={owner.initials} alt={owner.name} className="size-5" />
      <span className="max-w-[16ch] truncate text-[12px] text-muted-foreground">{owner.name}</span>
    </span>
  );
}

function NeedsAttentionCard({
  items,
  organizationId,
  factoryKey,
  viewAllHref,
}: {
  items: AttentionItem[];
  organizationId: string;
  factoryKey: string;
  viewAllHref: string;
}) {
  return (
    <OverviewCard
      title="Needs attention"
      subtitle="Work orders that wait for a human decision."
      count={items.length}
      headerAction={<CardViewAllLink href={viewAllHref} label="View all" />}
      testId="overview-attention-card"
    >
      {items.length === 0 ? (
        <CardEmptyState
          icon={<CircleCheck className="size-5 text-emerald-600 dark:text-emerald-400" aria-hidden />}
          title="Nothing waits on the team"
          hint="Work orders that need a decision appear here."
        />
      ) : (
        <ul>
          {items.slice(0, MAX_OVERVIEW_ROWS).map((item) => {
            const meta = ATTENTION_META[item.reason];
            const Icon = meta.icon;
            return (
              <li key={item.id} className={rowClassName} data-testid={`overview-attention-row-${item.id}`}>
                <RowLink
                  href={workOrderHref(organizationId, factoryKey, item.workOrderKey)}
                  label={`Open ${item.workOrderKey} ${item.title}`}
                />
                <Badge
                  variant="outline"
                  className={cn(
                    "inline-flex shrink-0 items-center gap-1 px-2 py-0.5 text-[11px] font-medium",
                    rowContentClassName,
                    meta.chipClassName,
                  )}
                >
                  <Icon className="size-3" aria-hidden />
                  {meta.label}
                </Badge>
                <div className={cn("flex min-w-0 flex-1 items-baseline gap-2", rowContentClassName)}>
                  <span className="shrink-0 text-[12px] font-medium tabular-nums text-muted-foreground">
                    {item.workOrderKey}
                  </span>
                  <p className="min-w-0 truncate text-[13px] font-medium text-foreground">{item.title}</p>
                </div>
                <span
                  className={cn("hidden shrink-0 text-[12px] text-muted-foreground lg:inline", rowContentClassName)}
                >
                  {item.lineName} · {item.stepName}
                </span>
                <OwnerReference owner={item.owner} className={rowContentClassName} />
                <span className={cn("shrink-0 text-[12px] tabular-nums text-muted-foreground", rowContentClassName)}>
                  {item.waitingFor}
                </span>
                {/* The action happens on the detail page, so the button links there too. */}
                <Button asChild size="xs" variant="outline" className="relative z-10 shrink-0">
                  <Link to={workOrderHref(organizationId, factoryKey, item.workOrderKey)}>{meta.actionLabel}</Link>
                </Button>
              </li>
            );
          })}
        </ul>
      )}
    </OverviewCard>
  );
}

/* ------------------------------ In flight ------------------------------ */

function StepProgress({
  stepIndex,
  stepCount,
  className,
}: {
  stepIndex: number;
  stepCount: number;
  className?: string;
}) {
  return (
    <span className={cn("flex items-center gap-0.5", className)} aria-hidden>
      {Array.from({ length: stepCount }, (_, index) => (
        <span
          key={index}
          className={cn("h-1 w-3 rounded-full", index < stepIndex ? "bg-blue-500 dark:bg-blue-400" : "bg-muted")}
        />
      ))}
    </span>
  );
}

function InFlightCard({
  items,
  organizationId,
  factoryKey,
  viewAllHref,
  newWorkOrderHref,
}: {
  items: InFlightItem[];
  organizationId: string;
  factoryKey: string;
  viewAllHref: string;
  newWorkOrderHref: string;
}) {
  return (
    <OverviewCard
      title="In flight"
      subtitle="Work orders that run on lines now."
      count={items.length}
      headerAction={<CardViewAllLink href={viewAllHref} label="View all" />}
      testId="overview-in-flight-card"
    >
      {items.length === 0 ? (
        <CardEmptyState
          title="No work orders run now"
          hint="Create a work order to send work through the workspace."
          action={
            <Button asChild size="xs" variant="outline">
              <Link to={newWorkOrderHref}>
                <Plus aria-hidden />
                New work order
              </Link>
            </Button>
          }
        />
      ) : (
        <ul>
          {items.slice(0, MAX_OVERVIEW_ROWS).map((item) => (
            <li key={item.id} className={rowClassName} data-testid={`overview-in-flight-row-${item.id}`}>
              <RowLink
                href={workOrderHref(organizationId, factoryKey, item.workOrderKey)}
                label={`Open ${item.workOrderKey} ${item.title}`}
              />
              <Loader2
                className={cn("size-3.5 shrink-0 animate-spin text-blue-500 dark:text-blue-400", rowContentClassName)}
                aria-hidden
              />
              <div className={cn("flex min-w-0 flex-1 items-baseline gap-2", rowContentClassName)}>
                <span className="shrink-0 text-[12px] font-medium tabular-nums text-muted-foreground">
                  {item.workOrderKey}
                </span>
                <p className="min-w-0 truncate text-[13px] font-medium text-foreground">{item.title}</p>
              </div>
              <span className={cn("hidden shrink-0 text-[12px] text-muted-foreground lg:inline", rowContentClassName)}>
                {item.lineName} · {item.stepName}
              </span>
              <StepProgress stepIndex={item.stepIndex} stepCount={item.stepCount} className={rowContentClassName} />
              <span
                className={cn(
                  "w-8 shrink-0 text-right text-[12px] tabular-nums text-muted-foreground",
                  rowContentClassName,
                )}
              >
                {item.elapsed}
              </span>
            </li>
          ))}
        </ul>
      )}
    </OverviewCard>
  );
}

/* --------------------------- Recently shipped --------------------------- */

const SHIPPED_META: Record<ShippedOutcome, { label: string; icon: typeof GitMerge; iconClassName: string }> = {
  merged: { label: "Merged", icon: GitMerge, iconClassName: "text-emerald-600 dark:text-emerald-400" },
  "in-review": { label: "In review", icon: GitPullRequestArrow, iconClassName: "text-blue-600 dark:text-blue-400" },
  unsuccessful: { label: "Unsuccessful", icon: CircleX, iconClassName: "text-red-600 dark:text-red-400" },
};

function RecentlyShippedCard({
  items,
  organizationId,
  factoryKey,
  viewAllHref,
}: {
  items: ShippedItem[];
  organizationId: string;
  factoryKey: string;
  viewAllHref: string;
}) {
  return (
    <OverviewCard
      title="Recently shipped"
      subtitle="Outcomes from the last 7 days."
      headerAction={<CardViewAllLink href={viewAllHref} label="View all" />}
      testId="overview-shipped-card"
    >
      {items.length === 0 ? (
        <CardEmptyState title="No completed work in the last 7 days" hint="Finished work orders appear here." />
      ) : (
        <ul>
          {items.slice(0, MAX_SHIPPED_ROWS).map((item) => {
            const meta = SHIPPED_META[item.outcome];
            const Icon = meta.icon;
            return (
              <li key={item.id} className={rowClassName} data-testid={`overview-shipped-row-${item.id}`}>
                <RowLink
                  href={workOrderHref(organizationId, factoryKey, item.workOrderKey)}
                  label={`Open ${item.workOrderKey} ${item.title}`}
                />
                <Icon className={cn("size-3.5 shrink-0", meta.iconClassName, rowContentClassName)} aria-hidden />
                <div className={cn("flex min-w-0 flex-1 items-baseline gap-2", rowContentClassName)}>
                  <span className="shrink-0 text-[12px] font-medium tabular-nums text-muted-foreground">
                    {item.workOrderKey}
                  </span>
                  <p className="min-w-0 truncate text-[13px] font-medium text-foreground">{item.title}</p>
                </div>
                <span
                  className={cn("hidden shrink-0 text-[12px] text-muted-foreground lg:inline", rowContentClassName)}
                >
                  {item.detail}
                </span>
                <span className={cn("shrink-0 text-[12px] text-muted-foreground", rowContentClassName)}>
                  {item.when}
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </OverviewCard>
  );
}
