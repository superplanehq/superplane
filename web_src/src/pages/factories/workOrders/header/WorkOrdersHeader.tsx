import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
import type { ReactNode } from "react";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import type { WorkOrderListState } from "../../lib/useWorkOrderListState";
import { useWorkOrdersHeaderShortcuts } from "../../lib/useWorkOrdersHeaderShortcuts";
import { factorySectionHeaderClassName } from "../../pages/factoryPageLayoutStyles";
import { buildAssigneeFilterOptions, buildLineFilterOptions } from "../../lib/workOrderFilterOptions";
import { WORK_ORDER_SCOPES, type WorkOrderListEntry } from "../../lib/workOrderListModel";
import { DisplayMenu } from "./DisplayMenu";
import { FilterChips } from "./FilterChips";
import { FilterMenu } from "./FilterMenu";
import { ScopePills } from "./ScopePills";
import { SearchField } from "./SearchField";

interface WorkOrdersHeaderProps {
  state: WorkOrderListState;
  /** Every entry before scope/filters, used to build the assignee options. */
  entries: WorkOrderListEntry[];
  factoryLines: FactoriesFactoryLine[];
  onCreateWorkOrder: () => void;
  canCreate: boolean;
  permissionsLoading: boolean;
  hostedCreditEmptyBanner?: ReactNode;
  brokenIntegrationsBanner?: ReactNode;
}

/**
 * Compact title bar for the Tasks page. Title, scope, and Filter
 * stay on the left. Search, Display, and New sit on the right.
 *
 * The Filter menu and the chip row read from one shared set of options, so
 * both always show the same labels.
 */
export function WorkOrdersHeader({
  state,
  entries,
  factoryLines,
  onCreateWorkOrder,
  canCreate,
  permissionsLoading,
  hostedCreditEmptyBanner,
  brokenIntegrationsBanner,
}: WorkOrdersHeaderProps) {
  const searchRef = useWorkOrdersHeaderShortcuts(state);
  const lineOptions = buildLineFilterOptions(factoryLines);
  const assigneeOptions = buildAssigneeFilterOptions(entries);

  return (
    <WorkspacePageHeader
      className={factorySectionHeaderClassName}
      data-testid="work-orders-header"
      title="Tasks"
      leading={
        <>
          <ScopePills
            value={state.scope}
            onChange={state.setScope}
            options={WORK_ORDER_SCOPES}
            testIdPrefix="work-orders-scope"
          />
          <FilterMenu state={state} lineOptions={lineOptions} assigneeOptions={assigneeOptions} />
        </>
      }
      actions={
        <>
          <SearchField
            inputRef={searchRef}
            open={state.searchOpen}
            value={state.search}
            onOpen={state.openSearch}
            onChange={state.setSearch}
            onClose={state.closeSearch}
          />
          <DisplayMenu state={state} />
          <PermissionTooltip
            allowed={canCreate || permissionsLoading}
            message="You do not have permission to create tasks."
          >
            <Button
              type="button"
              size="sm"
              disabled={!canCreate}
              onClick={onCreateWorkOrder}
              data-testid="work-order-list-create-button"
            >
              <Plus className="size-3.5" aria-hidden />
              New task
            </Button>
          </PermissionTooltip>
        </>
      }
      belowRow={
        hostedCreditEmptyBanner || brokenIntegrationsBanner || state.filterCount > 0 ? (
          <>
            {hostedCreditEmptyBanner}
            {brokenIntegrationsBanner}
            {state.filterCount > 0 ? (
              <FilterChips state={state} lineOptions={lineOptions} assigneeOptions={assigneeOptions} />
            ) : null}
          </>
        ) : undefined
      }
    />
  );
}
