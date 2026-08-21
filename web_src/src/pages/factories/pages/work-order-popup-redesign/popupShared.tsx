import type { FactoriesWorkOrderArtifact } from "@/api-client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { CircleDollarSign, Clock, FileText, XIcon } from "lucide-react";
import type { ReactNode } from "react";

import { FACTORIES_ORGANIZATION_ID } from "../../__fixtures__/factoryPageResponses";
import { extractArtifactMarkdownBody, toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { OrgUserReference } from "../../OrgUserReference";
import { WorkOrderArtifactInline } from "../../WorkOrderArtifactInline";
import { WorkOrderStatusNote } from "../../WorkOrderStatusNote";
import { RunOverlayBoardBackdrop, RunOverlayFrame } from "../work-order-run-overlay/runOverlayShared";
import type { PopupFixture, PopupLogEntry, PopupLogState } from "./workOrderPopupMocks";

export { RunOverlayBoardBackdrop };

export function PopupShell({
  testId,
  children,
  fixed = false,
  wide = false,
  onDismiss,
}: {
  testId: string;
  children: ReactNode;
  fixed?: boolean;
  wide?: boolean;
  onDismiss?: () => void;
}) {
  return (
    <RunOverlayFrame testId={testId} fixed={fixed} wide={wide} onDismiss={onDismiss}>
      {children}
    </RunOverlayFrame>
  );
}

export function PopupHeader({
  title,
  children,
  onClose,
}: {
  title: string;
  children?: ReactNode;
  onClose?: () => void;
}) {
  return (
    <header className="relative shrink-0 border-b border-border px-5 py-3 pr-12">
      <h2 className="truncate text-[16px] font-semibold tracking-[-0.02em] text-foreground">{title}</h2>
      {children}
      <button
        type="button"
        onClick={onClose}
        className="absolute top-2 right-2 flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10"
        aria-label="Close"
      >
        <XIcon className="h-4 w-4" />
      </button>
    </header>
  );
}

/** Owner, elapsed time, and spend. No status, author, or ticket key. */
export function OwnerTimeCostRow({ fixture, className }: { fixture: PopupFixture; className?: string }) {
  return (
    <div
      className={cn("mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-[13px] text-foreground", className)}
      data-testid="popup-owner-time-cost"
    >
      <span className="inline-flex min-w-0 items-center gap-1.5">
        <OrgUserReference display={fixture.owner} size="xs" nameClassName="truncate text-[13px]" />
      </span>
      <span className="inline-flex items-center gap-1.5 text-muted-foreground" title={fixture.startedLabel}>
        <Clock className="size-3.5 shrink-0" aria-hidden />
        <span className="text-foreground">{fixture.elapsed}</span>
      </span>
      <span className="inline-flex items-center gap-1.5 text-muted-foreground">
        <CircleDollarSign className="size-3.5 shrink-0" aria-hidden />
        <span className="text-foreground">
          {fixture.costUsd} <span className="text-muted-foreground">·</span> {fixture.tokensLabel}
        </span>
      </span>
    </div>
  );
}

export function DescriptionMarkdown({ artifact }: { artifact: FactoriesWorkOrderArtifact }) {
  const body = extractArtifactMarkdownBody(toArtifactDataRecord(artifact.data)) ?? "";
  return <MarkdownContent content={body} variant="workspace" />;
}

export function DescriptionFileCard({ artifact }: { artifact: FactoriesWorkOrderArtifact }) {
  return (
    <article className="overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex items-center gap-2 border-b border-border bg-muted/40 px-3 py-2">
        <FileText className="size-3.5 text-muted-foreground" aria-hidden />
        <span className="text-[13px] font-medium tracking-[-0.01em] text-foreground">description.md</span>
      </div>
      <div className="px-4 py-3">
        <DescriptionMarkdown artifact={artifact} />
      </div>
    </article>
  );
}

export function OutputList({ artifacts }: { artifacts: FactoriesWorkOrderArtifact[] }) {
  return (
    <ul className="flex flex-col">
      {artifacts.map((artifact) => (
        <li className="flex items-center py-1.5" key={artifact.id ?? `${artifact.type}-${artifact.createdAt}`}>
          <WorkOrderArtifactInline
            className="w-full justify-start"
            artifact={{
              id: artifact.id,
              type: artifact.type ?? "TYPE_UNSPECIFIED",
              data: toArtifactDataRecord(artifact.data),
            }}
          />
        </li>
      ))}
    </ul>
  );
}

export function WaitingNotes({ notes }: { notes: PopupFixture["waitingNotes"] }) {
  if (notes.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-3">
      {notes.map((note) => (
        <WorkOrderStatusNote
          key={note.key}
          note={note}
          organizationId={FACTORIES_ORGANIZATION_ID}
          canClose={false}
          canManage={false}
          isBusy={false}
          statusActions={[]}
          onClose={() => undefined}
          onStatusChange={async () => undefined}
        />
      ))}
    </div>
  );
}

const LOG_DOT: Record<PopupLogState, string> = {
  passed: "bg-[color:var(--status-completed-dot)]",
  running: "bg-[color:var(--status-running-dot)]",
  waiting: "bg-[color:var(--status-waiting-dot)]",
  failed: "bg-[color:var(--status-failed-dot)]",
};

export function AgentLogList({ entries }: { entries: PopupLogEntry[] }) {
  return (
    <ol className="flex flex-col">
      {entries.map((entry) => (
        <li key={entry.id} className="flex items-start gap-3 py-2">
          <span className={cn("mt-1.5 size-1.5 shrink-0 rounded-full", LOG_DOT[entry.state])} aria-hidden />
          <div className="min-w-0 flex-1">
            <p className="text-[13px] text-foreground">
              <span className="font-medium">{entry.title}</span>
              <span className="text-muted-foreground"> · {entry.actor}</span>
            </p>
            {entry.detail ? <p className="mt-0.5 text-[12px] text-muted-foreground">{entry.detail}</p> : null}
          </div>
          <span className="shrink-0 text-[12px] tabular-nums text-muted-foreground">{entry.duration}</span>
        </li>
      ))}
    </ol>
  );
}

export function PopupBody({ children }: { children: ReactNode }) {
  return <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">{children}</div>;
}

export function SectionTitle({ children }: { children: ReactNode }) {
  return <h3 className="workspace-section-title">{children}</h3>;
}

export function PrototypeSwitcher({
  value,
  onChange,
  options,
}: {
  value: string;
  onChange: (id: string) => void;
  options: { id: string; label: string; pattern: string }[];
}) {
  return (
    <div className="pointer-events-none absolute inset-x-0 top-0 z-20 flex justify-center p-3">
      <div className="pointer-events-auto flex items-center gap-2 rounded-full border border-border bg-background/95 px-2 py-1.5 shadow-sm">
        {options.map((entry) => (
          <Button
            key={entry.id}
            type="button"
            size="xs"
            variant={value === entry.id ? "default" : "ghost"}
            className={cn(value !== entry.id && "text-muted-foreground")}
            onClick={() => onChange(entry.id)}
            aria-pressed={value === entry.id}
          >
            {entry.label} · {entry.pattern}
          </Button>
        ))}
      </div>
    </div>
  );
}
