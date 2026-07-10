import type { MouseEvent, ReactNode } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

export function HeaderIconButton({
  label,
  icon,
  onClick,
  active,
}: {
  label: string;
  icon: ReactNode;
  onClick?: (event: MouseEvent<HTMLButtonElement>) => void;
  active?: boolean;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          aria-pressed={active}
          onClick={(event) => {
            event.stopPropagation();
            onClick?.(event);
          }}
          className={cn(
            "flex h-6 w-6 items-center justify-center rounded transition-colors",
            active
              ? "bg-blue-100 text-blue-700 hover:bg-blue-200"
              : "text-slate-400 hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-gray-800 dark:hover:text-gray-100",
          )}
        >
          {icon}
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">{label}</TooltipContent>
    </Tooltip>
  );
}
