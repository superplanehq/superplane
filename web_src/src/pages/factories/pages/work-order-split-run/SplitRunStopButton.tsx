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

import type { WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import {
  availableSplitRunStopChoices,
  DEFAULT_SPLIT_RUN_STOP_CHOICE,
  defaultSplitRunStopChoice,
  type SplitRunFooterKind,
  type SplitRunStopChoice,
} from "./splitRunFooter";

const SEGMENT_CLASSNAME =
  "inline-flex h-7 items-center justify-center text-[13px] font-medium text-primary-foreground hover:bg-primary-foreground/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring";

/**
 * Running-footer Stop control. One shell, two hit targets, same pattern
 * as GitHub Close issue.
 */
export function SplitRunStopButton({
  onStop,
  busy = false,
  status,
  kind,
}: {
  onStop?: (choice: SplitRunStopChoice) => void | Promise<void>;
  busy?: boolean;
  status?: WorkOrderDisplayStatus;
  kind?: SplitRunFooterKind;
}) {
  const choices = availableSplitRunStopChoices(status);
  const [choice, setChoice] = useState<SplitRunStopChoice>(
    () => defaultSplitRunStopChoice(status, kind) ?? DEFAULT_SPLIT_RUN_STOP_CHOICE,
  );
  const selectedId = choices.some((item) => item.id === choice) ? choice : choices[0]?.id;
  const selected = choices.find((item) => item.id === selectedId);

  if (!selected) {
    return null;
  }

  return (
    <div
      className="inline-flex items-stretch overflow-hidden rounded-md bg-primary shadow-sm"
      data-testid="split-run-stop"
    >
      <button
        type="button"
        className={cn(SEGMENT_CLASSNAME, "px-3")}
        disabled={busy}
        onClick={() => void onStop?.(selected.id)}
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
          {choices.map((item, index) => {
            const selected = item.id === selectedId;
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
