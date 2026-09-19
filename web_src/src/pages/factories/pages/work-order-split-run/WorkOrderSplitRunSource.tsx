import { ExternalLink } from "lucide-react";

import { logoDarkInvertClass } from "@/lib/logoDarkMode";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";

import { OrgUserReference } from "../../OrgUserReference";
import { isMonochromeSourceLogo, type SplitRunSource } from "./splitRunSource";

export function WorkOrderSplitRunSource({ source, compact = false }: { source: SplitRunSource; compact?: boolean }) {
  if (compact) {
    return (
      <div
        className="flex min-w-0 items-center gap-1.5 text-[11px] font-medium leading-none tracking-[-0.01em]"
        data-testid="split-run-source"
      >
        {source.kind === "intake" ? <CompactIntakeSource source={source} /> : <CompactManualSource source={source} />}
      </div>
    );
  }

  return (
    <div className="mt-2 flex flex-col gap-1.5 text-[13px] tracking-[-0.01em]" data-testid="split-run-source">
      {source.kind === "intake" ? <IntakeSource source={source} /> : <ManualSource source={source} />}
    </div>
  );
}

function IntakeSource({ source }: { source: Extract<SplitRunSource, { kind: "intake" }> }) {
  return (
    <>
      <p className="flex min-w-0 items-center gap-1.5 text-foreground">
        <IntakeSourceLogo source={source} className="dark:brightness-0 dark:invert" />
        <span className="truncate">{source.name}</span>
      </p>
      <IntakeTicket ticket={source.ticket} />
    </>
  );
}

function IntakeTicket({ ticket }: { ticket?: { label: string; href: string } }) {
  if (!ticket) {
    return null;
  }

  const href = safeExternalUrl(ticket.href);
  if (!href) {
    return <p className="truncate font-medium text-foreground">{ticket.label}</p>;
  }

  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex min-w-0 items-center gap-1.5 font-medium text-foreground hover:underline"
      data-testid="split-run-source-ticket"
    >
      <span className="truncate">{ticket.label}</span>
      <ExternalLink className="size-3 shrink-0 text-muted-foreground" aria-hidden />
    </a>
  );
}

function ManualSource({ source }: { source: Extract<SplitRunSource, { kind: "manual" }> }) {
  return (
    <>
      <OrgUserReference display={source.person} size="xs" nameClassName="truncate text-[13px]" />
      <p className="text-muted-foreground">{source.detail}</p>
    </>
  );
}

function CompactIntakeSource({ source }: { source: Extract<SplitRunSource, { kind: "intake" }> }) {
  return (
    <>
      <IntakeSourceLogo source={source} className="brightness-0 invert dark:invert-0" />
      {source.ticket ? <IntakeTicket ticket={source.ticket} /> : <span className="truncate">{source.name}</span>}
    </>
  );
}

function IntakeSourceLogo({
  source,
  className,
}: {
  source: Extract<SplitRunSource, { kind: "intake" }>;
  className: string;
}) {
  return (
    <img
      src={source.iconSrc}
      alt={source.iconAlt}
      className={cn(
        "size-4 shrink-0",
        isMonochromeSourceLogo(source.iconAlt) && className,
        logoDarkInvertClass(source.iconSrc),
      )}
    />
  );
}

function CompactManualSource({ source }: { source: Extract<SplitRunSource, { kind: "manual" }> }) {
  return <OrgUserReference display={source.person} size="xs" nameClassName="truncate text-[11px]" />;
}
