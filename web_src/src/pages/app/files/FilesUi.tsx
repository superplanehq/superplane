import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { GitCompareArrows } from "lucide-react";
import type { ReactNode } from "react";

export function DiffHeaderAction({
  hasPendingChanges,
  onDiffOpen,
}: {
  hasPendingChanges: boolean;
  onDiffOpen: () => void;
}) {
  if (!hasPendingChanges) {
    return null;
  }

  return (
    <Button type="button" variant="outline" size="sm" onClick={onDiffOpen}>
      <GitCompareArrows className="h-4 w-4" />
      Diff
    </Button>
  );
}

export function IconButton({
  label,
  disabled,
  onClick,
  className,
  children,
}: {
  label: string;
  disabled?: boolean;
  onClick: () => void;
  className?: string;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={label}
          disabled={disabled}
          onClick={onClick}
          className={`text-slate-600 hover:bg-slate-100 hover:text-slate-950 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100 ${className ?? ""}`}
        >
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom">{label}</TooltipContent>
    </Tooltip>
  );
}
