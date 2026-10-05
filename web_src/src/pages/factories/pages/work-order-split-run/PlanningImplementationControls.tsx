import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { SPLIT_RUN_CHAT_COLUMN_CLASSNAME } from "./splitRunPopupModel";

export function PlanningImplementationControls({
  modelSelect,
  actions,
  canSend,
  showSuggestChanges,
  onSuggestChanges,
}: {
  modelSelect?: ReactNode;
  actions?: ReactNode;
  canSend: boolean;
  showSuggestChanges: boolean;
  onSuggestChanges: () => void;
}) {
  if (!modelSelect && !actions) return null;
  return (
    <section aria-label="Implementation" className="shrink-0 border-t border-border py-3">
      <div className={cn(SPLIT_RUN_CHAT_COLUMN_CLASSNAME, "flex flex-wrap items-center justify-between gap-2")}>
        {showSuggestChanges ? (
          <Button type="button" variant="ghost" size="sm" disabled={!canSend} onClick={onSuggestChanges}>
            Suggest changes
          </Button>
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
