import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Check, Funnel } from "lucide-react";
import type { WorkOrderFilterDimension, WorkOrderListState } from "../../lib/useWorkOrderListState";
import { buildStatusFilterOptions, type WorkOrderFilterOption } from "../../lib/workOrderFilterOptions";
import { MENU_ITEM_CLASSNAME, MENU_LABEL_CLASSNAME } from "./menuStyles";

interface FilterMenuProps {
  state: WorkOrderListState;
  /** Omit on a line board: the page is already scoped to one line. */
  lineOptions?: WorkOrderFilterOption[];
  assigneeOptions: WorkOrderFilterOption[];
}

/** Filter trigger plus one submenu per dimension. Selections are additive. */
export function FilterMenu({ state, lineOptions, assigneeOptions }: FilterMenuProps) {
  const filterCount = lineOptions ? state.filterCount : state.filterCount - state.filters.lineIds.length;
  return (
    <DropdownMenu open={state.filterMenuOpen} onOpenChange={state.setFilterMenuOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label="Filter"
          className="relative size-8 shrink-0 text-muted-foreground"
          data-testid="work-orders-filter-trigger"
        >
          <Funnel className="size-3.5" aria-hidden />
          {filterCount > 0 ? (
            <span className="absolute -top-0.5 -right-0.5 flex size-3.5 items-center justify-center rounded-full bg-accent text-[9px] font-medium text-foreground">
              {filterCount}
            </span>
          ) : null}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-52">
        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Add filter</DropdownMenuLabel>

        <FilterSubMenu
          label="Status"
          resetLabel="Any status"
          dimension="statuses"
          state={state}
          options={buildStatusFilterOptions()}
        />

        {lineOptions ? (
          <FilterSubMenu
            label="Line"
            resetLabel="Any line"
            dimension="lineIds"
            state={state}
            options={lineOptions}
            emptyLabel="No lines yet"
          />
        ) : null}

        <FilterSubMenu
          label="Owner"
          resetLabel="Anyone"
          dimension="assigneeIds"
          state={state}
          options={assigneeOptions}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

interface FilterSubMenuProps {
  label: string;
  resetLabel: string;
  dimension: WorkOrderFilterDimension;
  state: WorkOrderListState;
  options: WorkOrderFilterOption[];
  emptyLabel?: string;
}

function FilterSubMenu({ label, resetLabel, dimension, state, options, emptyLabel }: FilterSubMenuProps) {
  const selected: readonly string[] = state.filters[dimension];
  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger className={MENU_ITEM_CLASSNAME} data-testid={`work-orders-filter-${dimension}`}>
        <span className="flex-1">{label}</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent className="w-48">
          <DropdownMenuItem
            className={MENU_ITEM_CLASSNAME}
            onSelect={(event) => {
              event.preventDefault();
              state.clearFilterDimension(dimension);
            }}
          >
            <span className="flex-1">{resetLabel}</span>
            {selected.length === 0 ? <Check className="size-3.5" aria-hidden /> : null}
          </DropdownMenuItem>

          {options.length === 0 ? (
            <div className="px-2 py-1.5 text-[12px] text-muted-foreground">{emptyLabel ?? "Nothing to filter"}</div>
          ) : null}

          {options.map((option) => (
            <DropdownMenuItem
              key={option.value}
              className={MENU_ITEM_CLASSNAME}
              onSelect={(event) => {
                event.preventDefault();
                state.toggleFilter(dimension, option.value);
              }}
            >
              {option.dot ? <span className={cn("size-1.5 rounded-full", option.dot)} aria-hidden /> : null}
              <span className="flex-1 truncate">{option.label}</span>
              {selected.includes(option.value) ? <Check className="size-3.5" aria-hidden /> : null}
            </DropdownMenuItem>
          ))}
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}
