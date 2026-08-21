import { Flame } from "lucide-react";

import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";
import type { Streak } from "@/pages/factories/verification/types";

interface StreakIndicatorProps {
  streaks: Streak[];
}

/** Current and best streaks without new blocking findings. */
export function StreakIndicator({ streaks }: StreakIndicatorProps) {
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {streaks.map((streak) => (
        <div key={streak.label} className={cn(factoryCardClassName, "flex flex-col gap-2 p-4")}>
          <div className="flex items-center gap-2">
            <Flame
              className={cn(
                "size-4 shrink-0",
                streak.current > 0 ? "text-amber-500 dark:text-amber-400" : "text-muted-foreground",
              )}
              aria-hidden
            />
            <span className="text-[12px] font-medium uppercase tracking-wide text-muted-foreground">
              {streak.label}
            </span>
          </div>
          <div className="flex items-baseline gap-2">
            <span className="text-4xl font-medium text-slate-900 dark:text-gray-100">{streak.current}</span>
            <span className="text-[13px] text-muted-foreground">{streak.unit}</span>
          </div>
          <p className="text-xs text-slate-500 dark:text-gray-400">
            Best: {streak.best} {streak.unit}
          </p>
        </div>
      ))}
    </div>
  );
}
