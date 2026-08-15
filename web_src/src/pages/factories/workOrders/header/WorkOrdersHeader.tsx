import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import type { WorkOrderListState } from "../../lib/useWorkOrderListState";
import { useWorkOrdersHeaderShortcuts } from "../../lib/useWorkOrdersHeaderShortcuts";
import { buildAssigneeFilterOptions, buildLineFilterOptions } from "../../lib/workOrderFilterOptions";
import type { WorkOrderListEntry } from "../../lib/workOrderListModel";
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
}

/**
 * Title bar for the Work Orders page. Uses the shared workspace page header,
 * with scope pills next to the title and the Filter menu, collapsible
 * search, Display menu, and New button in the trailing actions slot. Layout
 * and ordering sit inside Display rather than on the bar so the row stays
 * quiet.
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
}: WorkOrdersHeaderProps) {
  const searchRef = useWorkOrdersHeaderShortcuts(state);
  const lineOptions = buildLineFilterOptions(factoryLines);
  const assigneeOptions = buildAssigneeFilterOptions(entries);

  return (
    <WorkspacePageHeader
      data-testid="work-orders-header"
      title="Work Orders"
      leading={<ScopePills value={state.scope} onChange={state.setScope} />}
      actions={
        <>
          <FilterMenu state={state} lineOptions={lineOptions} assigneeOptions={assigneeOptions} />
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
            message="You don't have permission to create work orders."
          >
            <Button
              type="button"
              size="sm"
              disabled={!canCreate}
              onClick={onCreateWorkOrder}
              data-testid="work-order-list-create-button"
            >
              <Plus className="size-3.5" aria-hidden />
              New work order
            </Button>
          </PermissionTooltip>
        </>
      }
      belowRow={
        state.filterCount > 0 ? (
          <FilterChips state={state} lineOptions={lineOptions} assigneeOptions={assigneeOptions} />
        ) : undefined
      }
    />
  );
}
