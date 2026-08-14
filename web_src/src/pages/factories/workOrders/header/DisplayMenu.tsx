import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Check, Columns3, List, Settings2, Table as TableIcon } from "lucide-react";
import type { WorkOrderListState } from "../../lib/useWorkOrderListState";
import { WORK_ORDER_LAYOUTS, WORK_ORDER_ORDERINGS, type WorkOrderLayoutId } from "../../lib/workOrderListModel";
import { MENU_ITEM_CLASSNAME, MENU_LABEL_CLASSNAME, MENU_TRIGGER_CLASSNAME } from "./menuStyles";

const LAYOUT_ICONS: Record<WorkOrderLayoutId, typeof Columns3> = {
  board: Columns3,
  list: List,
  table: TableIcon,
};

/**
 * Layout and ordering. Both live behind one trigger because users set them
 * once and then leave them alone, unlike scope and filters.
 */
export function DisplayMenu({ state }: { state: WorkOrderListState }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={MENU_TRIGGER_CLASSNAME}
          data-testid="work-orders-display-trigger"
        >
          <Settings2 className="size-3.5" aria-hidden />
          Display
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Layout</DropdownMenuLabel>
        {WORK_ORDER_LAYOUTS.map((layout) => {
          const Icon = LAYOUT_ICONS[layout.id];
          return (
            <DropdownMenuItem
              key={layout.id}
              className={MENU_ITEM_CLASSNAME}
              onSelect={() => state.setLayout(layout.id)}
              data-testid={`work-orders-layout-${layout.id}`}
            >
              <Icon className="size-3.5" aria-hidden />
              <span className="flex-1">{layout.label}</span>
              {state.layout === layout.id ? <Check className="size-3.5" aria-hidden /> : null}
            </DropdownMenuItem>
          );
        })}

        <DropdownMenuSeparator />

        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Ordering</DropdownMenuLabel>
        {WORK_ORDER_ORDERINGS.map((ordering) => (
          <DropdownMenuItem
            key={ordering.id}
            className={MENU_ITEM_CLASSNAME}
            onSelect={() => state.setOrdering(ordering.id)}
            data-testid={`work-orders-ordering-${ordering.id}`}
          >
            <span className="flex-1">{ordering.label}</span>
            {state.ordering === ordering.id ? <Check className="size-3.5" aria-hidden /> : null}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
