import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { findIntakePresentation } from "@/lib/intakePresentation";
import { cn } from "@/lib/utils";

import { intakeStatusInfo } from "./intakeCatalogModel";

interface IntakeStatusPillProps {
  status: string;
  /** Show the one-line summary on hover. */
  withTooltip?: boolean;
  className?: string;
}

export function IntakeStatusPill({ status, withTooltip = true, className }: IntakeStatusPillProps) {
  const info = intakeStatusInfo(status);
  const pill = (
    <span
      data-testid="intake-status-pill"
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium whitespace-nowrap",
        info.pillClassName,
        className,
      )}
    >
      <span className={cn("size-1.5 rounded-full", info.dotClassName)} aria-hidden />
      {info.label}
    </span>
  );

  if (!withTooltip) {
    return pill;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{pill}</TooltipTrigger>
      <TooltipContent className="max-w-64">{info.summary}</TooltipContent>
    </Tooltip>
  );
}

interface IntakeIconProps {
  intakeKey: string;
  name: string;
  size?: "sm" | "md" | "lg";
}

const ICON_SIZES = {
  sm: "size-6 text-[10px]",
  md: "size-8 text-xs",
  lg: "size-10 text-sm",
};

export function IntakeIcon({ intakeKey, name, size = "md" }: IntakeIconProps) {
  const iconSrc = findIntakePresentation(intakeKey)?.iconSrc;
  return (
    <span
      className={cn(
        "flex shrink-0 items-center justify-center rounded-md border border-slate-200 bg-white font-semibold text-slate-600 uppercase dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300",
        ICON_SIZES[size],
      )}
      aria-hidden
    >
      {iconSrc ? <img src={iconSrc} alt="" className="size-[60%] object-contain" /> : name.slice(0, 1) || "?"}
    </span>
  );
}
