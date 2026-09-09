import { useState } from "react";
import { CalendarDays, ChevronDown } from "lucide-react";
import type { DateRange } from "react-day-picker";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Calendar } from "@/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";

import {
  rangeFromCustomDays,
  spendingPeriodTriggerLabel,
  SPENDING_PERIOD_PRESETS,
  type SpendingDateRange,
  type SpendingPeriodPreset,
} from "./spendingRedesignLib";

export function SpendingPeriodControls({
  period,
  customRange,
  customOpen,
  label = "Spending period",
  testId = "spending-period",
  pickerTestId = "spending-period-picker",
  onPeriodChange,
  onCustomOpenChange,
  onCustomRangeChange,
}: {
  period: SpendingPeriodPreset;
  customRange: SpendingDateRange;
  customOpen: boolean;
  label?: string;
  testId?: string;
  pickerTestId?: string;
  onPeriodChange: (value: string) => void;
  onCustomOpenChange: (open: boolean) => void;
  onCustomRangeChange: (range: SpendingDateRange) => void;
}) {
  const committed: DateRange = {
    from: customRange.start,
    to: new Date(customRange.end.getTime() - 1),
  };
  const [draftRange, setDraftRange] = useState<DateRange | undefined>();
  const selected = draftRange ?? committed;
  const triggerLabel = spendingPeriodTriggerLabel(period, customRange);

  return (
    <Popover
      open={customOpen}
      onOpenChange={(open) => {
        if (!open) {
          setDraftRange(undefined);
        }
        onCustomOpenChange(open);
      }}
    >
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          aria-expanded={customOpen}
          aria-haspopup="dialog"
          aria-label={`${label}, ${triggerLabel}`}
          data-testid={testId}
        >
          <CalendarDays className="size-3.5" aria-hidden />
          {triggerLabel}
          <ChevronDown className="size-3.5" aria-hidden />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-auto overflow-visible p-0" data-testid={pickerTestId}>
        <div className="flex flex-col sm:flex-row">
          <div
            role="radiogroup"
            aria-label={label}
            className="flex flex-col gap-0.5 border-b p-2 sm:w-44 sm:border-r sm:border-b-0"
          >
            {SPENDING_PERIOD_PRESETS.map((option) => {
              const isActive = period === option.value;
              return (
                <button
                  key={option.value}
                  type="button"
                  role="radio"
                  aria-checked={isActive}
                  className={cn(
                    "rounded-md px-2.5 py-1.5 text-left text-[13px]",
                    isActive
                      ? "bg-accent font-medium text-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-foreground",
                  )}
                  onClick={() => onPeriodChange(option.value)}
                >
                  {option.label}
                </button>
              );
            })}
          </div>
          <Calendar
            className="p-2 [--cell-size:2rem]"
            classNames={{ root: "rdp-root w-[16.5rem]" }}
            defaultMonth={selected.from}
            mode="range"
            selected={selected}
            onSelect={(next) => {
              if (!next?.from) {
                return;
              }
              setDraftRange(next);
              if (!next.to) {
                return;
              }
              onCustomRangeChange(rangeFromCustomDays(next.from, next.to));
            }}
          />
        </div>
      </PopoverContent>
    </Popover>
  );
}
