import { useState, type ReactNode } from "react";

import { cn } from "@/lib/utils";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/ui/hoverCard";

import {
  CLARITY_CHECK_NAME,
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  type ConfidenceBand,
} from "../../lib/confidenceScore";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import {
  CONFIDENCE_ANALYZING_TOOLTIP,
  ConfidenceAnalyzingIndicator,
  ConfidenceMeter,
  SCORE_PAIR_LABEL,
} from "../../workOrders/ConfidenceMeter";
import { SCORE_FALLBACK_WHY } from "./composerScoreSummary";

const CHIP_TONE: Record<ConfidenceBand, string> = {
  High: "border-emerald-500/30 bg-emerald-500/10",
  Medium: "border-orange-500/30 bg-orange-500/10",
  Low: "border-red-500/30 bg-red-500/10",
};

type ScoreKind = "clarity" | "confidence";

const SCORE_COPY: Record<ScoreKind, { name: string; fallbackWhy: string; testId: string }> = {
  clarity: {
    name: CLARITY_CHECK_NAME,
    fallbackWhy: SCORE_FALLBACK_WHY.clarity,
    testId: "split-run-intent-clarity",
  },
  confidence: {
    name: CONFIDENCE_CHECK_NAME,
    fallbackWhy: SCORE_FALLBACK_WHY.confidence,
    testId: "split-run-intent-confidence",
  },
};

type WorkOrderIntentConfidenceFooterProps = {
  clarity?: WorkOrderCheckPresentation;
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing: boolean;
  flush?: boolean;
};

/** Plan-pane footer: one why-chip per published score. */
export function WorkOrderIntentConfidenceFooter({
  clarity,
  confidence,
  isAnalyzing,
  flush = false,
}: WorkOrderIntentConfidenceFooterProps) {
  const hasScore = Boolean(clarity || confidence);

  if (isAnalyzing && !hasScore) {
    return (
      <ScoreFooterShell flush={flush}>
        <ScoreWhyChip
          key="analyzing"
          name={SCORE_PAIR_LABEL}
          label={`${SCORE_PAIR_LABEL}. Analyzing`}
          why={CONFIDENCE_ANALYZING_TOOLTIP}
          testId="split-run-intent-confidence"
        >
          <ConfidenceAnalyzingIndicator
            testId="split-run-intent-confidence-meter"
            showTooltip={false}
            decorative
            className="text-[13px]"
          />
        </ScoreWhyChip>
      </ScoreFooterShell>
    );
  }

  if (!hasScore) {
    return (
      <ScoreFooterShell flush={flush}>
        <p className="text-[13px] text-muted-foreground">No scores yet.</p>
      </ScoreFooterShell>
    );
  }

  return (
    <ScoreFooterShell flush={flush}>
      {clarity ? <ScoreChip kind="clarity" check={clarity} /> : null}
      {confidence ? <ScoreChip kind="confidence" check={confidence} /> : null}
    </ScoreFooterShell>
  );
}

function ScoreChip({ kind, check }: { kind: ScoreKind; check: WorkOrderCheckPresentation }) {
  const copy = SCORE_COPY[kind];
  const maxScore = check.maxScore || CONFIDENCE_SCORE_MAX;
  const scoreLabel = `${check.score}/${maxScore}`;
  return (
    <ScoreWhyChip
      key={`${check.score}-${check.summary ?? ""}`}
      name={copy.name}
      label={`${copy.name} ${scoreLabel}`}
      why={check.summary?.trim() || copy.fallbackWhy}
      tone={confidenceBandForScore(check.score)}
      testId={copy.testId}
    >
      <span className="flex flex-col leading-none">
        <span className="text-[11px] text-muted-foreground">{copy.name.replace(/ score$/, "")}</span>
        <span className="flex items-baseline gap-0.5">
          <span className="text-[22px] font-semibold tabular-nums text-foreground">{check.score}</span>
          <span className="text-[12px] text-muted-foreground">/{maxScore}</span>
        </span>
      </span>
      <ConfidenceMeter
        score={check.score}
        label={copy.name}
        testId={`${copy.testId}-meter`}
        showTooltip={false}
        decorative
        size="lg"
      />
    </ScoreWhyChip>
  );
}

function ScoreWhyChip({
  name,
  label,
  why,
  tone,
  testId,
  children,
}: {
  name: string;
  label: string;
  why: string;
  tone?: ConfidenceBand;
  testId: string;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const [pinned, setPinned] = useState(false);

  const setPanel = (next: boolean, pin = pinned) => {
    if (pin && !next) {
      return;
    }
    setOpen(next);
    if (!next) {
      setPinned(false);
    }
  };

  const togglePinned = () => {
    setPinned((current) => {
      const next = !current;
      setOpen(next);
      return next;
    });
  };

  return (
    <HoverCard open={open} onOpenChange={(next) => setPanel(next)} openDelay={0} closeDelay={80}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          aria-expanded={open}
          aria-label={label}
          data-testid={`${testId}-chip`}
          className={cn(
            "sp-stream-text inline-flex min-h-[3.25rem] items-center gap-2.5 rounded-xl border px-3 py-2 text-left",
            tone ? CHIP_TONE[tone] : "border-border bg-muted/40",
          )}
          onClick={(event) => {
            event.preventDefault();
            togglePinned();
          }}
        >
          {children}
        </button>
      </HoverCardTrigger>
      <HoverCardContent
        side="top"
        align="start"
        sideOffset={8}
        avoidCollisions={false}
        className="sp-confidence-why z-[80] w-80 p-3"
      >
        <p className="text-[12px] text-muted-foreground">{name}</p>
        <p className="mt-1 text-[13px] leading-5 text-foreground" data-testid={`${testId}-copy`}>
          {why}
        </p>
      </HoverCardContent>
    </HoverCard>
  );
}

function ScoreFooterShell({ children, flush }: { children: ReactNode; flush?: boolean }) {
  return (
    <footer
      className={cn(
        "flex flex-wrap items-center gap-2",
        flush ? undefined : "shrink-0 border-t border-border px-5 py-3",
      )}
      data-testid="split-run-overview-checks"
      aria-label={SCORE_PAIR_LABEL}
    >
      {children}
    </footer>
  );
}
