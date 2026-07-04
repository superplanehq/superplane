import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export function WidgetEmptyState({
  icon: Icon,
  message,
  testId,
  className,
}: {
  icon: LucideIcon;
  message: ReactNode;
  testId?: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex h-full min-h-[6rem] flex-col items-center justify-center gap-1.5 p-4 text-center text-[13px] text-gray-500",
        className,
      )}
      data-testid={testId}
    >
      <Icon className="size-4" aria-hidden />
      <p>{message}</p>
    </div>
  );
}
