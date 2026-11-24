import { ArrowRight, ExternalLink } from "lucide-react";
import { useParams } from "react-router";

import { Link } from "@/components/Link/link";
import { logoDarkInvertClass } from "@/lib/logoDarkMode";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";

import { OrgUserReference } from "../../OrgUserReference";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import { isMonochromeSourceLogo, type SplitRunRunRef, type SplitRunSource } from "./splitRunSource";

export function WorkOrderSplitRunSource({ source, compact = false }: { source: SplitRunSource; compact?: boolean }) {
  if (compact) {
    return (
      <div
        className="flex min-w-0 items-center gap-1.5 text-[11px] font-medium leading-none tracking-[-0.01em]"
        data-testid="split-run-source"
      >
        {source.kind === "intake" ? (
          <CompactIntakeSource source={source} />
        ) : source.kind === "mcp" || source.kind === "automation" ? (
          <CompactNamedSource source={source} />
        ) : (
          <CompactManualSource source={source} />
        )}
      </div>
    );
  }

  return (
    <div className="mt-2 flex flex-col gap-1.5 text-[13px] tracking-[-0.01em]" data-testid="split-run-source">
      {source.kind === "intake" ? (
        <IntakeSource source={source} />
      ) : source.kind === "mcp" || source.kind === "automation" ? (
        <NamedSource source={source} />
      ) : (
        <ManualSource source={source} />
      )}
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
      <IntakeSourceLogo source={source} className="dark:brightness-0 dark:invert" />
      {source.ticket ? <IntakeTicket ticket={source.ticket} /> : <span className="truncate">{source.name}</span>}
    </>
  );
}

function IntakeSourceLogo({
  source,
  className,
}: {
  source: Extract<SplitRunSource, { kind: "intake" | "mcp" | "automation" }>;
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

/** Client and automation sources show a name and a logo. */
type NamedSplitRunSource = Extract<SplitRunSource, { kind: "mcp" | "automation" }>;

function NamedSource({ source }: { source: NamedSplitRunSource }) {
  return (
    <p className="flex min-w-0 items-center gap-1.5 text-foreground">
      <IntakeSourceLogo source={source} className="dark:brightness-0 dark:invert" />
      <SourceRunName name={source.name} run={source.run} />
    </p>
  );
}

/**
 * Names the automation that created the task, then links to that run.
 * Needs the org and factory route params.
 */
function SourceRunName({ name, run }: { name: string; run?: SplitRunRunRef }) {
  const href = useSourceRunHref(run);
  return (
    <>
      <span className="truncate">{name}</span>
      {href ? (
        <Link
          href={href}
          className="inline-flex shrink-0 items-center gap-1 font-medium text-foreground hover:text-muted-foreground"
          data-testid="split-run-source-run"
        >
          {/* Decoration does not paint on a flex container's own text, so the underline sits on the span. */}
          <span className="underline underline-offset-2">View run</span>
          <ArrowRight className="size-3 shrink-0" aria-hidden />
        </Link>
      ) : null}
    </>
  );
}

function useSourceRunHref(run?: SplitRunRunRef): string | undefined {
  const { organizationId, factoryKey } = useParams();
  if (!run || !organizationId || !factoryKey) {
    return undefined;
  }
  return getWorkOrderRunHref(organizationId, factoryKey, run.appId, run.runId) || undefined;
}

function CompactNamedSource({ source }: { source: NamedSplitRunSource }) {
  return (
    <>
      <IntakeSourceLogo source={source} className="dark:brightness-0 dark:invert" />
      <SourceRunName name={source.name} run={source.run} />
    </>
  );
}

function CompactManualSource({ source }: { source: Extract<SplitRunSource, { kind: "manual" }> }) {
  return <OrgUserReference display={source.person} size="xs" nameClassName="truncate text-[11px]" />;
}
