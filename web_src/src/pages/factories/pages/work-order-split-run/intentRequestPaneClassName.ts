import { cn } from "@/lib/utils";

const REQUEST_PANE_BASE_CLASS = "flex min-h-0 min-w-0 w-full flex-1 flex-col";
const REQUEST_PANE_SPLIT_CLASS =
  "flex min-h-0 min-w-0 w-full flex-1 flex-col border-b border-border lg:w-[var(--intent-left)] lg:min-w-[14rem] lg:flex-none lg:border-r lg:border-b-0";
export const REFINE_SPLIT_EASE =
  "lg:duration-300 lg:ease-[cubic-bezier(0.32,0.72,0,1)] motion-reduce:lg:transition-none";

export function requestPaneClassName(
  refineOpen: boolean,
  showPlanPane: boolean,
  isResizing: boolean,
  planningReviewEnabled = false,
) {
  if (!refineOpen) {
    return showPlanPane ? REQUEST_PANE_SPLIT_CLASS : REQUEST_PANE_BASE_CLASS;
  }
  return cn(
    REQUEST_PANE_BASE_CLASS,
    "border-b border-border lg:w-[var(--intent-left)] lg:flex-none lg:border-r lg:border-b-0",
    showPlanPane && "lg:min-w-[14rem]",
    showPlanPane && planningReviewEnabled && "max-lg:hidden",
    !isResizing && `lg:transition-[width] ${REFINE_SPLIT_EASE}`,
  );
}
