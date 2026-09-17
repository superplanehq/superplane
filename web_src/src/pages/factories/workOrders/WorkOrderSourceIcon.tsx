import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";

import { workOrderCardSourceLabel, type WorkOrderCardSource } from "../lib/workOrderCardSource";

export function WorkOrderSourceIcon({ entryId, source }: { entryId: string; source: WorkOrderCardSource }) {
  const href = source.ticket ? safeExternalUrl(source.ticket.href) : null;
  const label = workOrderCardSourceLabel(source);
  const icon = (
    <img
      src={source.iconSrc}
      alt=""
      className={cn("size-3.5 shrink-0 object-contain", source.iconAlt === "GitHub" && "dark:brightness-0 dark:invert")}
    />
  );
  const frameClassName = "inline-flex size-6 shrink-0 items-center justify-center rounded-md";

  return (
    <div className="pointer-events-auto relative z-10" onClick={(event) => event.stopPropagation()}>
      {href ? (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          title={label}
          aria-label={label}
          data-testid={`work-order-card-source-${entryId}`}
          className={cn(
            frameClassName,
            "outline-none transition-colors hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring",
          )}
        >
          {icon}
        </a>
      ) : (
        <span
          title={label}
          aria-label={label}
          data-testid={`work-order-card-source-${entryId}`}
          className={frameClassName}
        >
          {icon}
        </span>
      )}
    </div>
  );
}
