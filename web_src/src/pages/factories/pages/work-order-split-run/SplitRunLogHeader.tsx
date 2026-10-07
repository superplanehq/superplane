import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";
import { Maximize2 } from "lucide-react";

import { SectionTitle } from "../work-order-popup-redesign/popupShared";

export function SplitRunLogHeader({ expandHref, className }: { expandHref?: string | null; className?: string }) {
  return (
    <div className={cn("flex items-center justify-between gap-2", className)}>
      <SectionTitle>Automations</SectionTitle>
      {expandHref ? (
        <Link
          href={expandHref}
          aria-label="Open automation run"
          className="flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          data-testid="split-run-log-expand"
        >
          <Maximize2 className="size-3.5" aria-hidden />
        </Link>
      ) : null}
    </div>
  );
}
