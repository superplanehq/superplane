import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

export type OverviewScope = "all" | "my";

const OVERVIEW_SCOPES: Array<{ id: OverviewScope; label: string; tooltip: string }> = [
  { id: "all", label: "All", tooltip: "Everything in this workspace." },
  { id: "my", label: "My", tooltip: "Only work orders assigned to you." },
];

/**
 * All / My scope toggle in the page header actions. Same pill styling as
 * the Work Orders scope pills so the control reads as one pattern across
 * pages. Scopes the three work order tables; health metrics and workspace
 * proposals always stay workspace-wide.
 */
export function OverviewScopeToggle({
  value,
  onChange,
}: {
  value: OverviewScope;
  onChange: (scope: OverviewScope) => void;
}) {
  return (
    <div className="flex items-center rounded-md border border-border p-0.5" role="group" aria-label="Scope">
      {OVERVIEW_SCOPES.map((scope) => {
        const active = scope.id === value;
        return (
          <Tooltip key={scope.id}>
            <TooltipTrigger asChild>
              <button
                type="button"
                aria-pressed={active}
                onClick={() => onChange(scope.id)}
                data-testid={`overview-scope-${scope.id}`}
                className={cn(
                  "inline-flex h-7 items-center rounded-[5px] px-2.5 text-[12px] font-medium transition-colors",
                  active ? "bg-accent text-foreground" : "text-muted-foreground hover:text-foreground",
                )}
              >
                {scope.label}
              </button>
            </TooltipTrigger>
            <TooltipContent side="bottom">{scope.tooltip}</TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}
