import { TriangleAlert } from "lucide-react";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";
import { SPLIT_RUN_CHAT_COLUMN_CLASSNAME } from "./splitRunPopupModel";

export function PlanningImplementationControls({
  startDiscouraged = false,
  modelSelect,
  actions,
}: {
  startDiscouraged?: boolean;
  modelSelect?: ReactNode;
  actions?: ReactNode;
}) {
  if (!modelSelect && !actions) return null;

  return (
    <section aria-label="Implementation" className="shrink-0 border-t border-border py-2.5">
      <div className={cn(SPLIT_RUN_CHAT_COLUMN_CLASSNAME, "flex flex-wrap items-center justify-between gap-2")}>
        {startDiscouraged ? (
          <div className="flex min-w-0 flex-1 items-center gap-2 sm:min-w-52">
            <TriangleAlert aria-hidden className="size-4 shrink-0 text-amber-700 dark:text-amber-400" />
            <p className="text-sm text-foreground/80">Review the plan before you start</p>
          </div>
        ) : (
          <span className="text-sm text-foreground/75">Implementation</span>
        )}
        <div
          className={cn(
            "ml-auto flex flex-wrap items-center justify-end gap-2",
            startDiscouraged && "w-full justify-between sm:w-auto sm:justify-end",
          )}
          data-testid="split-run-intent-settings"
        >
          {modelSelect}
          {actions}
        </div>
      </div>
    </section>
  );
}
