import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { WORK_ORDER_SCOPES, type WorkOrderScope } from "../../lib/workOrderListModel";

interface ScopePillsProps {
  value: WorkOrderScope;
  onChange: (scope: WorkOrderScope) => void;
}

/** All / Active / My selector next to the page title. */
export function ScopePills({ value, onChange }: ScopePillsProps) {
  return (
    <div className="flex items-center rounded-md border border-border p-0.5" role="group" aria-label="Scope">
      {WORK_ORDER_SCOPES.map((scope) => {
        const active = scope.id === value;
        return (
          <Tooltip key={scope.id}>
            <TooltipTrigger asChild>
              <button
                type="button"
                aria-pressed={active}
                onClick={() => onChange(scope.id)}
                data-testid={`work-orders-scope-${scope.id}`}
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
