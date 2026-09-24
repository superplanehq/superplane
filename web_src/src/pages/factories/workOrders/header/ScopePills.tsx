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

/** Scope selector using the same rounded tab chrome as factory settings tabs. */
export function ScopePills<T extends string>({ value, onChange, options, testIdPrefix }: ScopePillsProps<T>) {
  return (
    <div
      className="inline-flex h-7 w-fit items-center justify-center rounded-full bg-slate-100 p-0.5 text-muted-foreground dark:bg-gray-800"
      role="group"
      aria-label="Scope"
    >
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
                  "inline-flex h-[calc(100%-1px)] items-center justify-center rounded-full border border-transparent px-2.5 py-1 text-[13px] font-medium whitespace-nowrap transition-[color,box-shadow]",
                  active
                    ? "bg-background text-foreground shadow-sm dark:border-input dark:bg-input/30 dark:text-foreground"
                    : "text-foreground hover:text-foreground dark:text-muted-foreground",
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
