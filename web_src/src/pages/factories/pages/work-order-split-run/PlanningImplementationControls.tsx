import { MessageSquare, TriangleAlert } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { SPLIT_RUN_CHAT_COLUMN_CLASSNAME } from "./splitRunPopupModel";

export function PlanningImplementationControls({
  inline = false,
  startDiscouraged = false,
  modelSelect,
  actions,
  canSend,
  showSuggestChanges,
  onSuggestChanges,
}: {
  inline?: boolean;
  startDiscouraged?: boolean;
  modelSelect?: ReactNode;
  actions?: ReactNode;
  canSend: boolean;
  showSuggestChanges: boolean;
  onSuggestChanges: () => void;
}) {
  if (!modelSelect && !actions) return null;
  return (
    <section aria-label="Implementation" className={cn("shrink-0", !inline && "border-t border-border py-3")}>
      <div
        className={cn(!inline && SPLIT_RUN_CHAT_COLUMN_CLASSNAME, "flex flex-wrap items-center justify-between gap-2")}
      >
        {showSuggestChanges ? (
          <Button type="button" variant="outline" size="sm" disabled={!canSend} onClick={onSuggestChanges}>
            <MessageSquare aria-hidden className="size-4" />
            Suggest changes
          </Button>
        ) : startDiscouraged ? (
          <span className="flex items-center gap-1.5 text-[12px] text-amber-800 dark:text-amber-300">
            <TriangleAlert aria-hidden className="size-4 shrink-0" />
            Starting not recommended
          </span>
        ) : (
          <span className="text-[12px] text-muted-foreground">Implementation</span>
        )}
        <div className="ml-auto flex flex-wrap items-center justify-end gap-2" data-testid="split-run-intent-settings">
          {modelSelect}
          {actions}
        </div>
      </div>
    </section>
  );
}
