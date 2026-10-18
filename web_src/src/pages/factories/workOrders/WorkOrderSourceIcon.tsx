import { logoDarkInvertClass } from "@/lib/logoDarkMode";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";

import { workOrderCardSourceLabel, type WorkOrderCardSource } from "../lib/workOrderCardSource";

/** Same footprint as `WorkOrderStatusIcon` (size-3.5) so the footer lines up with the title row. */
const SOURCE_MARK_CLASS =
  "inline-flex size-3.5 shrink-0 items-center justify-center rounded-md transition-colors hover:bg-muted/60";

export function WorkOrderSourceIcon({ entryId, source }: { entryId: string; source: WorkOrderCardSource }) {
  const href = source.ticket ? safeExternalUrl(source.ticket.href) : null;
  const label = workOrderCardSourceLabel(source);
  const icon = (
    <img
      src={source.iconSrc}
      alt=""
      className={cn(
        "size-3 shrink-0 object-contain",
        (source.iconAlt === "GitHub" || source.iconAlt === "SuperPlane") && "dark:brightness-0 dark:invert",
        logoDarkInvertClass(source.iconSrc),
      )}
    />
  );
  const testId = `work-order-card-source-${entryId}`;

  return (
    <span
      className="pointer-events-auto relative z-10 inline-flex shrink-0 items-center justify-center self-center"
      onClick={(event) => event.stopPropagation()}
    >
      {href ? (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          title={label}
          aria-label={label}
          data-testid={testId}
          className={SOURCE_MARK_CLASS}
        >
          {icon}
        </a>
      ) : (
        <span title={label} aria-label={label} data-testid={testId} className={SOURCE_MARK_CLASS}>
          {icon}
        </span>
      )}
    </span>
  );
}
