import { Check, Circle, Loader2, X as XIcon } from "lucide-react";
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
  if (status === "running") {
    return (
      <Loader2
        className={cn("size-3.5 shrink-0 animate-spin text-[#2563eb] dark:text-blue-300", className)}
        aria-hidden
      />
    );
  }
  return (
    <Circle className={cn("size-3.5 shrink-0 text-[#a3a3a3] dark:text-muted-foreground", className)} strokeWidth={1.75} aria-hidden />
  );
}
