import { Check, Circle, CircleDashed, Loader2, TriangleAlert, X as XIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import type { FactoryNodeStatus } from "./types";

export function FactoryNodeStatusGlyph({ status, className }: { status: FactoryNodeStatus; className?: string }) {
  if (status === "passed") {
    return (
      <span
        className={cn(
          "flex size-4 shrink-0 items-center justify-center rounded-full bg-[#16a34a] text-white",
          className,
        )}
      >
        <Check className="size-2.5" strokeWidth={3} />
      </span>
    );
  }

  if (status === "failed") {
    return (
      <span
        className={cn(
          "flex size-4 shrink-0 items-center justify-center rounded-full bg-[#dc2626] text-white",
          className,
        )}
      >
        <XIcon className="size-2.5" strokeWidth={3} />
      </span>
    );
  }

  if (status === "error") {
    return (
      <TriangleAlert className={cn("size-3.5 shrink-0 text-[#dc2626] dark:text-red-300", className)} aria-hidden />
    );
  }

  if (status === "running") {
    return (
      <Loader2
        className={cn("size-3.5 shrink-0 animate-spin text-[#2563eb] dark:text-blue-300", className)}
        aria-hidden
      />
    );
  }

  if (status === "cancelling") {
    return (
      <Loader2
        className={cn("size-3.5 shrink-0 animate-spin text-[#d97706] dark:text-amber-300", className)}
        aria-hidden
      />
    );
  }

  if (status === "queued") {
    return (
      <CircleDashed
        className={cn("size-3.5 shrink-0 text-[#ea580c] dark:text-orange-300", className)}
        strokeWidth={1.75}
        aria-hidden
      />
    );
  }

  if (status === "triggered") {
    return (
      <Circle
        className={cn(
          "size-3.5 shrink-0 fill-[#8b5cf6] text-[#8b5cf6] dark:fill-violet-400 dark:text-violet-400",
          className,
        )}
        strokeWidth={0}
        aria-hidden
      />
    );
  }

  if (status === "cancelled") {
    return (
      <span
        className={cn(
          "flex size-4 shrink-0 items-center justify-center rounded-full bg-[#737373] text-white",
          className,
        )}
      >
        <XIcon className="size-2.5" strokeWidth={3} />
      </span>
    );
  }

  // pending + did_not_run share muted hollow circle; label carries the distinction.
  return (
    <Circle
      className={cn("size-3.5 shrink-0 text-[#a3a3a3] dark:text-muted-foreground", className)}
      strokeWidth={1.75}
      aria-hidden
    />
  );
}
