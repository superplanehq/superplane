import type { FactoryApp } from "@/api-client";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { LoadingButton } from "@/components/ui/loading-button";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Copy, MoreHorizontal, Pencil, Plus, Trash2, Workflow } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type MouseEvent } from "react";
import { useNavigate } from "react-router";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import {
  listFactoryAutomationRuns,
  resolveFactoryAutomationStatus,
  resolveFactoryAutomationStatusFromCanvasRuns,
  type FactoryAutomationRunCard,
  type FactoryAutomationTick,
} from "../lib/factoryAutomationStatus";
import { automationDetailPath, automationsPath, factoryAppRunPath } from "../lib/factoryPagePaths";
import { type AutomationCardActions } from "./automationCardActions";
import { LineVelocityPanel } from "./LineVelocityPanel";

export function AutomationDetail({
  organizationId,
  factoryKey,
  app,
  actions,
}: {
  organizationId: string;
  factoryKey: string;
  app: FactoryApp;
  actions: AutomationCardActions;
}) {
  const canvasId = app.id ?? "";
  const {
    data: runsPages,
    isLoading: runsLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteCanvasRuns(canvasId, {}, Boolean(canvasId));
  const canvasRuns = useMemo(() => runsPages?.pages.flatMap((page) => page?.runs ?? []) ?? [], [runsPages]);
  const status = resolveFactoryAutomationStatusFromCanvasRuns(canvasRuns);
  const runs = useMemo(() => listFactoryAutomationRuns(canvasRuns), [canvasRuns]);

  const runsScrollRef = useRef<HTMLUListElement>(null);
  const loadMoreRuns = useCallback(() => {
    if (!hasNextPage || isFetchingNextPage) {
      return;
    }
    void fetchNextPage();
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);
  const loadMoreRunsIfNeeded = useAutoLoadMoreOnScroll({
    hasMore: Boolean(hasNextPage),
    isLoading: isFetchingNextPage,
    onLoadMore: loadMoreRuns,
  });

  useEffect(() => {
    loadMoreRunsIfNeeded(runsScrollRef.current);
  }, [runs.length, loadMoreRunsIfNeeded]);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const automationName = app.name?.trim() || "Unnamed automation";
  const description = app.description?.trim();
  const subtitleParts = [status.label, description].filter(Boolean);
  const subtitle = subtitleParts.length > 0 ? subtitleParts.join(" · ") : undefined;

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="automations-detail">
      <WorkspacePageHeader
        variant="entity"
        backHref={automationsPath(organizationId, factoryKey)}
        backLabel="Automations"
        backTestId="automations-detail-back"
        title={automationName}
        subtitle={subtitle}
        actions={
          <AutomationHeaderActions
            name={automationName}
            actions={actions}
            onRequestDelete={() => setDeleteOpen(true)}
          />
        }
      />

      <DeleteAutomationDialog
        open={deleteOpen}
        name={automationName}
        canDelete={actions.canDelete}
        isDeleting={Boolean(actions.isDeleting)}
        onClose={() => setDeleteOpen(false)}
        onConfirm={actions.onDelete}
      />

      <Tabs
        defaultValue="runs"
        className="mt-6 flex min-h-0 min-w-0 flex-1 flex-col gap-3 px-[var(--workspace-page-gutter)]"
      >
        <TabsList>
          <TabsTrigger value="runs" data-testid="automations-tab-runs">
            Runs
          </TabsTrigger>
          <TabsTrigger value="velocity" data-testid="automations-tab-velocity">
            Velocity
          </TabsTrigger>
        </TabsList>
        <TabsContent value="runs" className="mt-0 flex min-h-0 min-w-0 flex-1 flex-col outline-none">
          <ul
            ref={runsScrollRef}
            className="flex min-h-[12rem] min-w-0 max-h-[calc(100dvh-16rem)] flex-1 flex-col gap-2 overflow-y-auto overscroll-contain p-0.5 [scrollbar-width:thin]"
            onScroll={(event) => loadMoreRunsIfNeeded(event.currentTarget)}
            data-testid="automations-runs-scroll"
          >
            {runsLoading && runs.length === 0 ? (
              <li className="px-2 py-4 text-[12px] text-muted-foreground">Loading runs…</li>
            ) : runs.length === 0 ? (
              <li className="px-2 py-4 text-[12px] text-muted-foreground">No runs</li>
            ) : (
              <>
                {runs.map((run) => (
                  <li key={run.runId}>
                    <AutomationRunRow
                      organizationId={organizationId}
                      factoryKey={factoryKey}
                      appId={canvasId}
                      run={run}
                    />
                  </li>
                ))}
                {isFetchingNextPage ? (
                  <li className="py-2 text-center text-[12px] text-muted-foreground">Loading more…</li>
                ) : null}
              </>
            )}
          </ul>
        </TabsContent>
        <TabsContent
          value="velocity"
          className="mt-0 min-h-0 max-h-[calc(100dvh-16rem)] flex-1 overflow-y-auto outline-none [scrollbar-width:thin]"
        >
          <LineVelocityPanel intro="Run throughput for this automation." />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function AutomationRunRow({
  organizationId,
  factoryKey,
  appId,
  run,
}: {
  organizationId: string;
  factoryKey: string;
  appId: string;
  run: FactoryAutomationRunCard;
}) {
  const href = factoryAppRunPath(organizationId, factoryKey, appId, run.runId, { from: "automations" });
  const timestamp = run.updatedAt ?? run.finishedAt ?? run.createdAt;
  const timeLabel =
    run.tick === "queued" && !timestamp ? "next" : timestamp ? formatTimeAgo(new Date(timestamp), false) : null;

  return (
    <Link
      href={href}
      className="block w-full rounded-md border border-border bg-background px-3 py-2.5 text-left transition-colors hover:border-foreground/25 hover:bg-accent/40"
      data-testid={`automations-run-${run.runId}`}
    >
      <div className="truncate text-[13px] font-medium tracking-[-0.01em] text-foreground">{run.title}</div>
      <div className="mt-1.5 flex items-center gap-1.5 text-[12px] text-muted-foreground">
        <StatusTick tick={run.tick} size="sm" />
        <span>
          {run.label}
          {timeLabel ? ` · ${timeLabel}` : ""}
        </span>
      </div>
    </Link>
  );
}

function resolveAutomationCardNavigation(href: string | undefined, navigate: ReturnType<typeof useNavigate>) {
  if (!href) {
    return {
      interactive: false,
      onClick: undefined as (() => void) | undefined,
      onKeyDown: undefined as ((event: { key: string; preventDefault: () => void }) => void) | undefined,
    };
  }
  return {
    interactive: true,
    onClick: () => navigate(href),
    onKeyDown: (event: { key: string; preventDefault: () => void }) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        navigate(href);
      }
    },
  };
}

export function AutomationCard({
  app,
  href,
  tick,
  statusLabel,
  emphasized,
  actions,
}: {
  app: FactoryApp;
  href?: string;
  tick: FactoryAutomationTick;
  statusLabel: string;
  emphasized?: boolean;
  actions?: AutomationCardActions;
}) {
  const navigate = useNavigate();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const description = app.description?.trim() || "Factory automation canvas.";
  const automationName = app.name?.trim() || "Unnamed automation";
  const { interactive, onClick, onKeyDown } = resolveAutomationCardNavigation(href, navigate);

  return (
    <div
      role={interactive ? "button" : undefined}
      tabIndex={interactive ? 0 : undefined}
      className={cn(
        "group/automation w-full rounded-lg border px-3.5 py-3 text-left transition-colors",
        emphasized ? "border-foreground bg-background" : "border-border bg-background",
        interactive && "cursor-pointer hover:border-foreground/25 hover:bg-accent/40",
      )}
      onClick={onClick}
      onKeyDown={onKeyDown}
      data-testid={emphasized ? `automations-detail-card-${app.id}` : `automations-app-${app.id}`}
    >
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <Workflow className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={1.75} aria-hidden />
            <span className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{automationName}</span>
          </div>
          <p className="mt-1 text-[12px] leading-snug text-muted-foreground">{description}</p>
        </div>
        {actions ? (
          <AutomationCardMenu
            name={automationName}
            actions={actions}
            hoverReveal={!emphasized}
            onRequestDelete={() => setDeleteOpen(true)}
          />
        ) : null}
      </div>
      <div className="mt-3.5 flex items-center gap-1.5">
        <StatusTick tick={tick} />
        <span className="text-[12px] leading-tight text-muted-foreground">{statusLabel}</span>
      </div>
      {actions ? (
        <DeleteAutomationDialog
          open={deleteOpen}
          name={automationName}
          canDelete={actions.canDelete}
          isDeleting={Boolean(actions.isDeleting)}
          onClose={() => setDeleteOpen(false)}
          onConfirm={actions.onDelete}
        />
      ) : null}
    </div>
  );
}

const overflowMenuContentClassName = "min-w-[12rem] rounded-xl border-border p-1 shadow-lg";
const overflowMenuItemClassName = "cursor-pointer rounded-md px-2 py-1.5 text-[13px] [&_svg]:size-3.5";
const overflowMenuDestructiveItemClassName =
  "text-destructive focus:bg-destructive/10 focus:text-destructive dark:focus:bg-destructive/15";

export function AutomationHeaderActions({
  name,
  actions,
  onRequestDelete,
}: {
  name: string;
  actions: AutomationCardActions;
  onRequestDelete: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className="text-muted-foreground hover:bg-accent hover:text-foreground"
          disabled={actions.isDuplicating || actions.isDeleting}
          aria-label={`${name} menu`}
          data-testid="automations-detail-menu"
        >
          <MoreHorizontal className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <AutomationMenuItems actions={actions} onRequestDelete={onRequestDelete} testIdPrefix="automations-detail" />
    </DropdownMenu>
  );
}

function AutomationCardMenu({
  name,
  actions,
  hoverReveal,
  onRequestDelete,
}: {
  name: string;
  actions: AutomationCardActions;
  hoverReveal: boolean;
  onRequestDelete: () => void;
}) {
  const stopCardClick = (event: MouseEvent<HTMLElement>) => {
    event.stopPropagation();
  };

  return (
    <div className="shrink-0" onClick={stopCardClick}>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label={`${name} menu`}
            className={cn(
              "flex size-7 items-center justify-center rounded-md text-muted-foreground transition hover:bg-accent hover:text-foreground",
              hoverReveal &&
                "opacity-0 group-hover/automation:opacity-100 group-focus-within/automation:opacity-100 data-[state=open]:opacity-100",
            )}
            disabled={actions.isDuplicating || actions.isDeleting}
            data-testid="automations-card-menu"
          >
            <MoreHorizontal className="size-3.5" aria-hidden />
          </button>
        </DropdownMenuTrigger>
        <AutomationMenuItems actions={actions} onRequestDelete={onRequestDelete} testIdPrefix="automations-card" />
      </DropdownMenu>
    </div>
  );
}

function AutomationMenuItems({
  actions,
  onRequestDelete,
  testIdPrefix,
}: {
  actions: AutomationCardActions;
  onRequestDelete: () => void;
  testIdPrefix: string;
}) {
  return (
    <DropdownMenuContent align="end" sideOffset={6} className={overflowMenuContentClassName}>
      <PermissionTooltip allowed={actions.canEdit} message="You don't have permission to edit this automation.">
        <DropdownMenuItem
          className={overflowMenuItemClassName}
          onClick={actions.onEdit}
          disabled={!actions.canEdit}
          data-testid={`${testIdPrefix}-edit`}
        >
          <Pencil aria-hidden />
          Edit
        </DropdownMenuItem>
      </PermissionTooltip>
      <PermissionTooltip
        allowed={actions.canDuplicate}
        message="You don't have permission to duplicate this automation."
      >
        <DropdownMenuItem
          className={overflowMenuItemClassName}
          onClick={() => {
            void actions.onDuplicate();
          }}
          disabled={!actions.canDuplicate || actions.isDuplicating}
          data-testid={`${testIdPrefix}-duplicate`}
        >
          <Copy aria-hidden />
          Duplicate
        </DropdownMenuItem>
      </PermissionTooltip>
      <DropdownMenuSeparator />
      <PermissionTooltip allowed={actions.canDelete} message="You don't have permission to delete this automation.">
        <DropdownMenuItem
          className={cn(overflowMenuItemClassName, overflowMenuDestructiveItemClassName)}
          onClick={onRequestDelete}
          disabled={!actions.canDelete || actions.isDeleting}
          data-testid={`${testIdPrefix}-delete`}
        >
          <Trash2 aria-hidden />
          Delete
        </DropdownMenuItem>
      </PermissionTooltip>
    </DropdownMenuContent>
  );
}

function DeleteAutomationDialog({
  open,
  name,
  canDelete,
  isDeleting,
  onClose,
  onConfirm,
}: {
  open: boolean;
  name: string;
  canDelete: boolean;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: () => void | Promise<void>;
}) {
  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen && !isDeleting) {
          onClose();
        }
      }}
    >
      <DialogContent showCloseButton={false} className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete "{name}"?</DialogTitle>
        </DialogHeader>
        <p className="text-[13px] text-muted-foreground">This cannot be undone.</p>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={isDeleting}>
            Cancel
          </Button>
          <LoadingButton
            type="button"
            variant="destructive"
            onClick={() => {
              void (async () => {
                try {
                  await onConfirm();
                  onClose();
                } catch {
                  // Toast is handled by the caller; keep the dialog open for retry.
                }
              })();
            }}
            disabled={!canDelete}
            loading={isDeleting}
            loadingText="Deleting..."
            data-testid="automations-delete-confirm"
          >
            <Trash2 className="h-3.5 w-3.5" aria-hidden />
            Delete
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function StatusTick({ tick, size = "md" }: { tick: FactoryAutomationTick; size?: "sm" | "md" }) {
  return (
    <span
      className={cn(
        size === "sm" ? "size-1.5" : "size-2",
        "shrink-0 rounded-full bg-[#c4c4c4]",
        tick === "running" && "bg-[#3b82f6] animate-pulse",
        tick === "waiting" && "bg-[#f59e0b]",
        tick === "failed" && "bg-[#ef4444]",
        tick === "queued" && "bg-[#a3a3a3]",
        tick === "passed" && "bg-[#22c55e]",
        tick === "cancelled" && "bg-[#a3a3a3]",
      )}
      aria-hidden
    />
  );
}

export function EmptyAutomationsState({ canCreate, onCreate }: { canCreate: boolean; onCreate: () => void }) {
  return (
    <div
      className="flex flex-col items-center rounded-lg border border-dashed border-border bg-card px-6 py-12 text-center"
      data-testid="automations-empty-state"
    >
      <Workflow className="h-8 w-8 text-muted-foreground" aria-hidden />
      <p className="mt-3 text-[15px] font-medium text-foreground">No automations yet</p>
      <p className="mt-1 max-w-md text-[13px] text-muted-foreground">
        Automations listen for triggers and run canvases that power line phases.
      </p>
      <Button type="button" size="sm" className="mt-6" disabled={!canCreate} onClick={onCreate}>
        <Plus className="size-3.5" aria-hidden />
        New automation
      </Button>
    </div>
  );
}

export function AutomationsPageList({
  organizationId,
  factoryKey,
  apps,
  workOrders,
  actionsForApp,
}: {
  organizationId: string;
  factoryKey: string;
  apps: FactoryApp[];
  workOrders: Parameters<typeof resolveFactoryAutomationStatus>[1];
  actionsForApp: (app: FactoryApp) => AutomationCardActions;
}) {
  return (
    <ul className="flex flex-col gap-2" data-testid="automations-list">
      {apps.map((app) => {
        if (!app.id) {
          return null;
        }
        const status = resolveFactoryAutomationStatus(app.id, workOrders);
        return (
          <li key={app.id}>
            <AutomationCard
              app={app}
              href={automationDetailPath(organizationId, factoryKey, app.id)}
              tick={status.tick}
              statusLabel={status.label}
              actions={actionsForApp(app)}
            />
          </li>
        );
      })}
    </ul>
  );
}
