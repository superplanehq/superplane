import { cn } from "@/lib/utils";

/** Status color plus the running ping used on work-order cards. */
export function WorkOrderStatusDot({
  colorClassName,
  pulsing = false,
  title,
  className,
  "aria-label": ariaLabel,
  "aria-hidden": ariaHidden,
  "data-testid": testId,
}: {
  colorClassName: string;
  pulsing?: boolean;
  title?: string;
  className?: string;
  "aria-label"?: string;
  "aria-hidden"?: boolean;
  "data-testid"?: string;
}) {
  return (
    <span
      className={cn("relative size-2 shrink-0", className)}
      title={title}
      aria-label={ariaLabel}
      aria-hidden={ariaHidden}
      data-testid={testId}
    >
      {pulsing ? (
        <span className={cn("absolute inset-0 rounded-full opacity-60 animate-ping", colorClassName)} aria-hidden />
      ) : null}
      <span className={cn("relative block size-full rounded-full", colorClassName)} />
    </span>
  );
}
