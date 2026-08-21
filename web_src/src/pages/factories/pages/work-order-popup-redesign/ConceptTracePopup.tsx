import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

import { toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { WorkOrderArtifactInline } from "../../WorkOrderArtifactInline";
import { WorkOrderCheckCard } from "../../WorkOrderChecksSection";
import { DescriptionMarkdown, OwnerTimeCostRow, PopupBody, PopupHeader, PopupShell, WaitingNotes } from "./popupShared";
import type { PopupFixture, PopupLogEntry, PopupLogState } from "./workOrderPopupMocks";

/**
 * Trace inspector. Pattern: LangSmith / agent observability.
 *
 * The work is a tree of spans. description.md is the input span.
 * Artifacts hang off the span that produced them. Checks are evals.
 */
export function ConceptTracePopup({ fixture }: { fixture: PopupFixture }) {
  return (
    <PopupShell testId="work-order-popup-trace">
      <PopupHeader title={fixture.title}>
        <OwnerTimeCostRow fixture={fixture} />
      </PopupHeader>
      <PopupBody>
        <ol className="flex flex-col gap-1 border-l border-border pl-4">
          <TraceSpan name="Input" actor="description.md" duration="—" state="passed" open>
            <DescriptionMarkdown artifact={fixture.description} />
          </TraceSpan>

          {fixture.log.map((entry) => (
            <TraceSpan
              key={entry.id}
              name={entry.title}
              actor={entry.actor}
              duration={entry.duration}
              state={entry.state}
              detail={entry.detail}
              open={entry.state === "running" || entry.state === "waiting"}
            >
              <SpanArtifact fixture={fixture} entry={entry} />
            </TraceSpan>
          ))}

          <TraceSpan name="Evals" actor="Automations" duration="—" state="passed" open>
            <div className="grid gap-3 sm:grid-cols-2">
              {fixture.checks.slice(0, 4).map((check) => (
                <WorkOrderCheckCard key={check.id} check={check} />
              ))}
            </div>
          </TraceSpan>
        </ol>

        <div className="mt-8">
          <WaitingNotes notes={fixture.waitingNotes} />
        </div>
      </PopupBody>
    </PopupShell>
  );
}

const STATE_DOT: Record<PopupLogState, string> = {
  passed: "bg-[color:var(--status-completed-dot)]",
  running: "bg-[color:var(--status-running-dot)]",
  waiting: "bg-[color:var(--status-waiting-dot)]",
};

function TraceSpan({
  name,
  actor,
  duration,
  state,
  detail,
  open,
  children,
}: {
  name: string;
  actor: string;
  duration: string;
  state: PopupLogState;
  detail?: string;
  open?: boolean;
  children?: ReactNode;
}) {
  return (
    <li className="relative py-2">
      <span className={cn("absolute top-3.5 -left-[1.15rem] size-2 rounded-full", STATE_DOT[state])} aria-hidden />
      <div className="flex items-baseline justify-between gap-3">
        <p className="min-w-0 text-[13px] text-foreground">
          <span className="font-medium">{name}</span>
          <span className="text-muted-foreground"> · {actor}</span>
        </p>
        <span className="shrink-0 text-[12px] tabular-nums text-muted-foreground">{duration}</span>
      </div>
      {detail ? <p className="mt-0.5 text-[12px] text-muted-foreground">{detail}</p> : null}
      {open && children ? <div className="mt-2">{children}</div> : null}
    </li>
  );
}

function SpanArtifact({ fixture, entry }: { fixture: PopupFixture; entry: PopupLogEntry }) {
  if (!entry.artifactId) {
    return null;
  }
  const artifact =
    entry.artifactId === fixture.description.id
      ? fixture.description
      : fixture.outputs.find((item) => item.id === entry.artifactId);
  if (!artifact) {
    return null;
  }
  return (
    <div className="rounded-md border border-border bg-card px-2.5 py-1.5">
      <WorkOrderArtifactInline
        artifact={{
          id: artifact.id,
          type: artifact.type ?? "TYPE_UNSPECIFIED",
          data: toArtifactDataRecord(artifact.data),
        }}
      />
    </div>
  );
}
