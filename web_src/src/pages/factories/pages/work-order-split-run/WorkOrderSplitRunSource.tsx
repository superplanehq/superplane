import { ExternalLink } from "lucide-react";

import { safeExternalUrl } from "@/lib/safeExternalUrl";

import { OrgUserReference } from "../../OrgUserReference";
import type { SplitRunSource } from "./splitRunSource";

export function WorkOrderSplitRunSource({ source }: { source: SplitRunSource }) {
  return (
    <div className="mt-2 flex flex-col gap-1.5 text-[13px] tracking-[-0.01em]" data-testid="split-run-source">
      {source.kind === "intake" ? <IntakeSource source={source} /> : <ManualSource source={source} />}
    </div>
  );
}

function IntakeSource({ source }: { source: Extract<SplitRunSource, { kind: "intake" }> }) {
  const href = safeExternalUrl(source.ticket.href);
  return (
    <>
      <p className="flex min-w-0 items-center gap-1.5 text-foreground">
        <img src={source.iconSrc} alt={source.iconAlt} className="size-4 shrink-0" />
        <span className="truncate">{source.name}</span>
      </p>
      {href ? (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex min-w-0 items-center gap-1.5 font-medium text-foreground hover:underline"
          data-testid="split-run-source-ticket"
        >
          <span className="truncate">{source.ticket.label}</span>
          <ExternalLink className="size-3 shrink-0 text-muted-foreground" aria-hidden />
        </a>
      ) : (
        <p className="truncate font-medium text-foreground">{source.ticket.label}</p>
      )}
    </>
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
