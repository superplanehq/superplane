import type { FactoriesFactoryLine } from "@/api-client";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
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
  createHref: string;
  canCreate: boolean;
  permissionsLoading: boolean;
}

/**
 * Title bar for the Work Orders page. Everything lives on one row: the page
 * title and scope pills on the left, and the Filter menu, collapsible
 * search, Display menu, and New button on the right. Layout and ordering sit
 * inside Display rather than on the bar so the row stays quiet.
 *
 * The Filter menu and the chip row read from one shared set of options, so
 * both always show the same labels.
 */
export function WorkOrdersHeader({
  state,
  entries,
  factoryLines,
  createHref,
  canCreate,
  permissionsLoading,
}: WorkOrdersHeaderProps) {
  const searchRef = useWorkOrdersHeaderShortcuts(state);
  const lineOptions = buildLineFilterOptions(factoryLines);
  const assigneeOptions = buildAssigneeFilterOptions(entries);

  return (
    <div className="flex w-full flex-col" data-testid="work-orders-header">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 flex-wrap items-center gap-3">
          <h1 className="text-[22px] font-semibold tracking-[-0.02em] text-foreground">Work Orders</h1>
          <ScopePills value={state.scope} onChange={state.setScope} />
        </div>

        <div className="flex items-center gap-1">
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
              asChild
              disabled={!canCreate}
              className="h-8 shrink-0 gap-1.5 px-2.5"
              data-testid="work-order-list-create-button"
            >
              <Link href={canCreate ? createHref : "#"}>
                <Plus className="size-3.5" aria-hidden />
                New
              </Link>
            </Button>
          </PermissionTooltip>
        </div>
      </div>

      <FilterChips state={state} lineOptions={lineOptions} assigneeOptions={assigneeOptions} />
    </div>
  );
}
