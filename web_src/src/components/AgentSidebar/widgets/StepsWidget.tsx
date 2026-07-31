import { Check, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import type { StepItem } from "./parser";

interface StepsWidgetProps {
  items: StepItem[];
}

export function StepsWidget({ items }: StepsWidgetProps) {
  const firstPending = items.findIndex((i) => !i.done);

  return (
    <div className="my-4 space-y-1 rounded-lg border border-edge-default bg-surface-raised p-3">
      {items.map((item, i) => {
        const isActive = i === firstPending;
        return (
          <div
            key={i}
            className={cn("flex items-center gap-2 text-xs", !item.done && !isActive && "text-content-muted")}
          >
            {item.done ? (
              <Check className="size-3.5 text-green-600 shrink-0" />
            ) : isActive ? (
              <Loader2 className="size-3.5 shrink-0 animate-spin text-content-secondary" />
            ) : (
              <div className="size-3.5 shrink-0 rounded-full border border-edge-default" />
            )}
            <span className={cn(item.done && "text-content-secondary", isActive && "font-medium text-content-primary")}>
              {item.text}
            </span>
          </div>
        );
      })}
    </div>
  );
}
