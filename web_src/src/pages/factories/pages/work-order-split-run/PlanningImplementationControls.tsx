import { ArrowRight, MessageSquare } from "lucide-react";
import { useId, useState, type ReactNode } from "react";
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
  const options = (
    <div className="ml-auto flex flex-wrap items-center justify-end gap-2" data-testid="split-run-intent-settings">
      {modelSelect}
      {actions}
    </div>
  );
  return (
    <section aria-label="Implementation" className={cn("shrink-0", !inline && "border-t border-border py-3")}>
      <div
        className={cn(!inline && SPLIT_RUN_CHAT_COLUMN_CLASSNAME, "flex flex-wrap items-center justify-between gap-2")}
      >
        {startDiscouraged ? (
          <DiscouragedImplementationOptions>{options}</DiscouragedImplementationOptions>
        ) : (
          <>
            {showSuggestChanges ? (
              <Button type="button" variant="outline" size="sm" disabled={!canSend} onClick={onSuggestChanges}>
                <MessageSquare aria-hidden className="size-4" />
                Suggest changes
              </Button>
            ) : (
              <span className="text-[12px] text-muted-foreground">Implementation</span>
            )}
            {options}
          </>
        )}
      </div>
    </section>
  );
}

function DiscouragedImplementationOptions({ children }: { children: ReactNode }) {
  const [expanded, setExpanded] = useState(false);
  const optionsId = useId();
  return (
    <>
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-[12px] text-muted-foreground">
        <span>Starting not recommended.</span>
        <Button
          type="button"
          variant="link"
          size="sm"
          className="h-auto gap-1 p-0 text-[12px] text-muted-foreground underline underline-offset-4 hover:text-foreground"
          aria-expanded={expanded}
          aria-controls={optionsId}
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? "Hide options" : "Override"}
          {!expanded ? <ArrowRight aria-hidden className="size-3" /> : null}
        </Button>
      </div>
      <div id={optionsId} hidden={!expanded} className="ml-auto">
        {children}
      </div>
    </>
  );
}
