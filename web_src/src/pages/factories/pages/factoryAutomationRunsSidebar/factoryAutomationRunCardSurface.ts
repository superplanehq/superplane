import { cn } from "@/lib/utils";

import { WORK_ORDER_CARD_HOVER_SURFACE_CLASS } from "../../workOrders/workOrderCardSurface";

/** Selected fill lives on the card, not on the sidebar column. */
export function factoryAutomationRunCardSurfaceClass(isSelected: boolean): string {
  return cn(
    "shadow-none hover:shadow-none hover:border-border",
    isSelected
      ? "border-foreground/15 bg-slate-200/80 hover:bg-slate-200/80 dark:border-foreground/20 dark:bg-slate-800 dark:hover:bg-slate-800"
      : WORK_ORDER_CARD_HOVER_SURFACE_CLASS,
  );
}
