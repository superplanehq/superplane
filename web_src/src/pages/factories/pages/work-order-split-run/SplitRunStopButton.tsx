import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Check, ChevronDown } from "lucide-react";
import { useState } from "react";

import { DEFAULT_SPLIT_RUN_STOP_CHOICE, SPLIT_RUN_STOP_CHOICES, type SplitRunStopChoice } from "./splitRunFooter";

const SEGMENT_CLASSNAME =
  "inline-flex h-7 items-center justify-center text-[13px] font-medium text-primary-foreground hover:bg-primary-foreground/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring";

/**
 * Running-footer Stop control. One shell, two hit targets, same pattern
 * as GitHub Close issue.
 */
export function SplitRunStopButton({
  onStop,
  busy = false,
}: {
  onStop?: (choice: SplitRunStopChoice) => void | Promise<void>;
  busy?: boolean;
}) {
  const [choice, setChoice] = useState<SplitRunStopChoice>(DEFAULT_SPLIT_RUN_STOP_CHOICE);
  const selected = SPLIT_RUN_STOP_CHOICES.find((item) => item.id === choice) ?? SPLIT_RUN_STOP_CHOICES[0];

  return (
    <div
      className="inline-flex items-stretch overflow-hidden rounded-md bg-primary shadow-sm"
      data-testid="split-run-stop"
    >
      <button
        type="button"
        className={cn(SEGMENT_CLASSNAME, "px-3")}
        disabled={busy}
        onClick={() => void onStop?.(choice)}
        data-testid="split-run-footer-stop"
      >
        {selected.actionLabel}
      </button>
      <span className="w-px self-stretch bg-primary-foreground/25" aria-hidden />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className={cn(SEGMENT_CLASSNAME, "px-2")}
            aria-label="Choose how to stop"
            disabled={busy}
            data-testid="split-run-stop-menu"
          >
            <ChevronDown className="size-3.5" aria-hidden />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" side="top" className="w-72">
          {SPLIT_RUN_STOP_CHOICES.map((item, index) => {
            const selected = item.id === choice;
            return (
              <div key={item.id}>
                {index > 0 ? <DropdownMenuSeparator /> : null}
                <DropdownMenuItem
                  className="items-start gap-2 py-2"
                  data-selected={selected}
                  data-testid={`split-run-stop-${item.id}`}
                  onSelect={() => setChoice(item.id)}
                >
                  <span className="mt-0.5 flex w-3.5 shrink-0 justify-center" aria-hidden>
                    {selected ? <Check className="size-3.5" /> : null}
                  </span>
                  <span className="flex min-w-0 flex-col gap-0.5">
                    <span className="text-[13px] font-medium text-foreground">{item.label}</span>
                    <span className="text-[12px] leading-4 text-muted-foreground">{item.description}</span>
                  </span>
                </DropdownMenuItem>
              </div>
            );
          })}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
