import { Button } from "@/components/ui/button";
import { logoDarkInvertClass } from "@/lib/logoDarkMode";
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
      className={cn(
        "size-3.5 shrink-0 object-contain",
        (source.iconAlt === "GitHub" || source.iconAlt === "SuperPlane") && "dark:brightness-0 dark:invert",
        logoDarkInvertClass(source.iconSrc),
      )}
    />
  );
  const testId = `work-order-card-source-${entryId}`;

  return (
    <div className="pointer-events-auto relative z-10" onClick={(event) => event.stopPropagation()}>
      {href ? (
        <Button asChild variant="ghost" size="icon-xs" className="size-6 rounded-md">
          <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            title={label}
            aria-label={label}
            data-testid={testId}
          >
            {icon}
          </a>
        </Button>
      ) : (
        <span
          title={label}
          aria-label={label}
          data-testid={testId}
          className="inline-flex size-6 shrink-0 items-center justify-center"
        >
          {icon}
        </span>
      )}
    </div>
  );
}
