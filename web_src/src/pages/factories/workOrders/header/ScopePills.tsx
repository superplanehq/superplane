import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

export interface ScopePillOption<T extends string> {
  id: T;
  label: string;
  tooltip: string;
}

interface ScopePillsProps<T extends string> {
  value: T;
  onChange: (scope: T) => void;
  options: ReadonlyArray<ScopePillOption<T>>;
  /** Prefix for each pill's data-testid, e.g. "work-orders-scope" → "work-orders-scope-all". */
  testIdPrefix: string;
}

/** Pill-style scope selector shared by the Tasks header and the Overview scope toggle. */
export function ScopePills<T extends string>({ value, onChange, options, testIdPrefix }: ScopePillsProps<T>) {
  return (
    <div className="flex items-center rounded-md border border-border p-0.5" role="group" aria-label="Scope">
      {options.map((scope) => {
        const active = scope.id === value;
        return (
          <Tooltip key={scope.id}>
            <TooltipTrigger asChild>
              <button
                type="button"
                aria-pressed={active}
                onClick={() => onChange(scope.id)}
                data-testid={`${testIdPrefix}-${scope.id}`}
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
