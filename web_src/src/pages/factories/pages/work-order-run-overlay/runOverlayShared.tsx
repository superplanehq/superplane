import type { FactoriesWorkOrderArtifact } from "@/api-client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { XIcon } from "lucide-react";
import type { ReactNode } from "react";

import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { WorkOrderArtifactInline } from "../../WorkOrderArtifactInline";
import { WorkOrderCheckCard } from "../../WorkOrderChecksSection";
import { PhaseGlyph } from "../linePhaseGlyph";

import {
  continueActionLabel,
  nextPhaseId,
  overlayStatusGlyph,
  overlayStatusLabel,
  type RunOverlayFixture,
  type RunOverlayPhase,
  type RunOverlayPhaseId,
} from "./workOrderRunOverlayMocks";

/** Dimmed line board behind the overlay, so the modal still sits on the kanban. */
export function RunOverlayBoardBackdrop() {
  return (
    <div className="flex min-h-svh gap-3 bg-muted/40 p-4" aria-hidden>
      {["Plan", "Implement", "Verify"].map((lane) => (
        <div key={lane} className="flex min-h-[36rem] min-w-72 flex-1 flex-col rounded-xl bg-card/70 p-3">
          <p className="mb-3 text-[13px] font-medium text-muted-foreground">{lane}</p>
          <div className="space-y-2">
            <div className="rounded-lg border border-border bg-card px-3 py-2.5 text-[13px] text-foreground">
              {lane === "Implement" ? "Ship idempotent refund retries" : "Queued work order"}
            </div>
            <div className="rounded-lg border border-border bg-card px-3 py-2.5 text-[13px] text-muted-foreground">
              {lane === "Plan" ? "Draft retry telemetry" : "Follow-up work"}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

export function RunOverlayFrame({
  children,
  testId,
  wide = false,
  canvas = false,
  fixed = false,
  onDismiss,
}: {
  children: ReactNode;
  testId: string;
  wide?: boolean;
  canvas?: boolean;
  /** Cover the viewport. Use on the live line board so overflow does not clip the dialog. */
  fixed?: boolean;
  onDismiss?: () => void;
}) {
  return (
    <div
      className={cn(
        "inset-0 z-50 flex items-center justify-center bg-black/50 p-3 sm:p-6",
        fixed ? "fixed" : "absolute",
      )}
      onClick={onDismiss}
    >
      <div
        className={cn(
          "flex max-h-[min(52rem,calc(100vh-2.5rem))] flex-col overflow-hidden rounded-lg border border-border bg-background shadow-lg dark:bg-gray-900",
          canvas
            ? "h-[min(52rem,calc(100vh-2.5rem))] w-[min(90vw,84rem)]"
            : wide
              ? "h-[min(48rem,calc(100vh-2.5rem))] w-[min(72rem,calc(100vw-2rem))]"
              : "h-[min(46rem,calc(100vh-2.5rem))] w-[min(56rem,calc(100vw-2rem))]",
        )}
        data-testid={testId}
        role="dialog"
        aria-modal="true"
        onClick={(event) => event.stopPropagation()}
      >
        {children}
      </div>
    </div>
  );
}

export function RunOverlayHeader({
  fixture,
  phase,
  onContinue,
  onClose,
  trailing,
}: {
  fixture: RunOverlayFixture;
  phase: RunOverlayPhase;
  onContinue?: () => void;
  onClose?: () => void;
  trailing?: ReactNode;
}) {
  const next = nextPhaseId(phase.id);

  return (
    <header className="relative flex shrink-0 items-start gap-3 border-b border-border px-5 py-3 pr-12">
      <div className="min-w-0 flex-1">
        <p className="text-[12px] text-muted-foreground">
          {fixture.runLabel}
          {" · "}
          {fixture.elapsed}
          {" · "}
          {phase.name}
        </p>
        <h2 className="mt-0.5 truncate text-[16px] font-semibold tracking-[-0.02em] text-foreground">
          {fixture.title}
        </h2>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {trailing}
        {onContinue ? (
          <Button type="button" size="sm" onClick={onContinue} disabled={!next && phase.id !== "verify"}>
            {continueActionLabel(phase.id)}
          </Button>
        ) : null}
      </div>
      {onClose ? (
        <button
          type="button"
          onClick={onClose}
          className="absolute top-2 right-2 flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10"
          aria-label="Close run"
        >
          <XIcon className="h-4 w-4" />
        </button>
      ) : null}
    </header>
  );
}

export function RunCheckGrid({
  checks,
  emptyLabel,
  className,
}: {
  checks: WorkOrderCheckPresentation[];
  emptyLabel: string;
  className?: string;
}) {
  if (checks.length === 0) {
    return <p className="text-[13px] text-muted-foreground">{emptyLabel}</p>;
  }

  return (
    <div className={cn("grid gap-3 sm:grid-cols-2 xl:grid-cols-4", className)}>
      {checks.map((check) => (
        <WorkOrderCheckCard key={check.id} check={check} />
      ))}
    </div>
  );
}

export function RunAttachedList({
  artifacts,
  emptyLabel,
}: {
  artifacts: FactoriesWorkOrderArtifact[];
  emptyLabel: string;
}) {
  if (artifacts.length === 0) {
    return <p className="text-[13px] text-muted-foreground">{emptyLabel}</p>;
  }

  return (
    <ul className="flex flex-wrap gap-2">
      {artifacts.map((artifact) => (
        <li
          key={artifact.id ?? `${artifact.type}-${artifact.createdAt}`}
          className="rounded-md border border-border bg-card px-2.5 py-1.5"
        >
          <WorkOrderArtifactInline
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

export function HorizontalPhaseStepper({
  fixture,
  selectedId,
  onSelect,
}: {
  fixture: RunOverlayFixture;
  selectedId: RunOverlayPhaseId;
  onSelect: (id: RunOverlayPhaseId) => void;
}) {
  return (
    <ol className="flex items-stretch gap-0">
      {fixture.phases.map((phase, index) => {
        const selected = phase.id === selectedId;
        return (
          <li key={phase.id} className="flex min-w-0 flex-1 items-center">
            {index > 0 ? <span aria-hidden className="h-px w-3 shrink-0 bg-border sm:w-6" /> : null}
            <button
              type="button"
              onClick={() => onSelect(phase.id)}
              className={cn(
                "flex min-w-0 flex-1 items-center gap-2 rounded-lg border px-3 py-2 text-left transition-colors",
                selected
                  ? "border-foreground/20 bg-accent/50"
                  : "border-border bg-card hover:border-foreground/15 hover:bg-accent/30",
              )}
              aria-current={selected ? "step" : undefined}
            >
              <PhaseGlyph kind={overlayStatusGlyph(phase.status)} />
              <span className="min-w-0">
                <span className="block truncate text-[13px] font-medium text-foreground">{phase.name}</span>
                <span className="block truncate text-[11px] text-muted-foreground">
                  {overlayStatusLabel(phase.status)}
                  {phase.duration ? ` · ${phase.duration}` : ""}
                </span>
              </span>
            </button>
          </li>
        );
      })}
    </ol>
  );
}

export function SectionLabel({ children }: { children: ReactNode }) {
  return <h3 className="workspace-section-label mb-2">{children}</h3>;
}
