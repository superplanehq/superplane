import { Check, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import type { StepItem } from "./parser";

interface StepsWidgetProps {
  items: StepItem[];
}

export function StepsWidget({ items }: StepsWidgetProps) {
  const firstPending = items.findIndex((i) => !i.done);

  return (
    <div className="my-4 space-y-1 rounded-lg border border-slate-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-800">
      {items.map((item, i) => {
        const isActive = i === firstPending;
        return (
          <div
            key={i}
            className={cn(
              "flex items-center gap-2 text-xs",
              !item.done && !isActive && "text-slate-400 dark:text-gray-500",
            )}
          >
            {item.done ? (
              <Check className="size-3.5 text-green-600 shrink-0" />
            ) : isActive ? (
              <Loader2 className="size-3.5 shrink-0 animate-spin text-slate-600 dark:text-gray-300" />
            ) : (
              <div className="size-3.5 shrink-0 rounded-full border border-slate-300 dark:border-gray-600" />
            )}
            <span
              className={cn(
                item.done && "text-slate-600 dark:text-gray-300",
                isActive && "font-medium text-slate-900 dark:text-gray-100",
              )}
            >
              {item.text}
            </span>
          </div>
        );
      })}
    </div>
  );
}
