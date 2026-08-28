import { Icon } from "@/components/Icon";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { AlertCircle, ClipboardList, Loader2 } from "lucide-react";

const CARD_CLASSES =
  "flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border bg-card px-6 py-16 text-center";

export function WorkOrdersLoadingState() {
  return (
    <div className={CARD_CLASSES} data-testid="work-orders-loading-state">
      <Loader2 className="size-5 animate-spin text-muted-foreground" aria-hidden />
      <p className="text-[13px] text-muted-foreground">Loading work orders…</p>
    </div>
  );
}

export function WorkOrdersErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className={CARD_CLASSES} data-testid="work-orders-error-state">
      <AlertCircle className="size-5 text-destructive" aria-hidden />
      <div className="space-y-1">
        <p className="text-[13px] font-medium text-foreground">We could not load work orders.</p>
        <p className="text-[12px] text-muted-foreground">Check your network and try again.</p>
      </div>
      <Button type="button" variant="outline" size="sm" onClick={onRetry} data-testid="work-orders-error-retry">
        Retry
      </Button>
    </div>
  );
}

interface WorkOrdersTrueEmptyStateProps {
  onCreateWorkOrder: () => void;
  canCreate: boolean;
  permissionsLoading: boolean;
}

export function WorkOrdersTrueEmptyState({
  onCreateWorkOrder,
  canCreate,
  permissionsLoading,
}: WorkOrdersTrueEmptyStateProps) {
  return (
    <div className={CARD_CLASSES} data-testid="work-orders-empty-state">
      <ClipboardList className="size-6 text-muted-foreground" aria-hidden />
      <div className="space-y-1">
        <p className="text-[14px] font-medium text-foreground">No work orders yet</p>
        <p className="text-[12px] text-muted-foreground">
          Create your first work order to plan and dispatch work across factory lines.
        </p>
      </div>
      <PermissionTooltip
        allowed={canCreate || permissionsLoading}
        message="You do not have permission to create tasks."
      >
        <Button type="button" disabled={!canCreate} onClick={onCreateWorkOrder} data-testid="work-orders-empty-create">
          <Icon name="plus" />
          New task
        </Button>
      </PermissionTooltip>
    </div>
  );
}

interface WorkOrdersScopedEmptyStateProps {
  scopeLabel: string;
  onResetScope: () => void;
}

export function WorkOrdersScopedEmptyState({ scopeLabel, onResetScope }: WorkOrdersScopedEmptyStateProps) {
  return (
    <div className={CARD_CLASSES} data-testid="work-orders-scope-empty-state">
      <ClipboardList className="size-5 text-muted-foreground" aria-hidden />
      <div className="space-y-1">
        <p className="text-[14px] font-medium text-foreground">Nothing in {scopeLabel}</p>
        <p className="text-[12px] text-muted-foreground">Switch scope to see other work orders in this workspace.</p>
      </div>
      <Button type="button" variant="outline" size="sm" onClick={onResetScope}>
        Show all work orders
      </Button>
    </div>
  );
}

interface WorkOrdersFilteredEmptyStateProps {
  onClearFilters: () => void;
}

export function WorkOrdersFilteredEmptyState({ onClearFilters }: WorkOrdersFilteredEmptyStateProps) {
  return (
    <div className={CARD_CLASSES} data-testid="work-orders-filtered-empty-state">
      <ClipboardList className="size-5 text-muted-foreground" aria-hidden />
      <div className="space-y-1">
        <p className="text-[14px] font-medium text-foreground">No work orders match these filters</p>
        <p className="text-[12px] text-muted-foreground">
          Try a different search, remove a filter, or widen the scope.
        </p>
      </div>
      <Button type="button" variant="outline" size="sm" onClick={onClearFilters}>
        Clear filters
      </Button>
    </div>
  );
}
