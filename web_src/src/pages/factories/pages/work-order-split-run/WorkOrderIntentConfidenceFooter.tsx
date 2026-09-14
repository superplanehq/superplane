import { useState, type ReactNode } from "react";

import { cn } from "@/lib/utils";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/ui/hoverCard";

import {
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
} from "../../workOrders/ConfidenceMeter";

const CHIP_TONE: Record<ConfidenceBand, string> = {
  High: "border-emerald-500/30 bg-emerald-500/10",
  Medium: "border-orange-500/30 bg-orange-500/10",
  Low: "border-red-500/30 bg-red-500/10",
};

const FALLBACK_WHY = "The analysis scored how clear this work is.";

type WorkOrderIntentConfidenceFooterProps = {
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing: boolean;
  flush?: boolean;
};

export function WorkOrderIntentConfidenceFooter({
  confidence,
  isAnalyzing,
  flush = false,
}: WorkOrderIntentConfidenceFooterProps) {
  if (isAnalyzing && !confidence) {
    return (
      <ConfidenceFooterShell flush={flush}>
        <ConfidenceWhyChip
          key="analyzing"
          label={`${CONFIDENCE_CHECK_NAME}. Analyzing`}
          why={CONFIDENCE_ANALYZING_TOOLTIP}
        >
          <ConfidenceAnalyzingIndicator
            testId="split-run-intent-confidence-meter"
            showTooltip={false}
            decorative
            className="text-[13px]"
          />
        </ConfidenceWhyChip>
      </ConfidenceFooterShell>
    );
  }

  if (!confidence) {
    return (
      <ConfidenceFooterShell flush={flush}>
        <p className="text-[13px] text-muted-foreground">No confidence score yet.</p>
      </ConfidenceFooterShell>
    );
  }

  const maxScore = confidence.maxScore || CONFIDENCE_SCORE_MAX;
  const scoreLabel = `${confidence.score}/${maxScore}`;

  return (
    <ConfidenceFooterShell flush={flush}>
      <ConfidenceWhyChip
        key={`${confidence.score}-${confidence.summary ?? ""}`}
        label={`${CONFIDENCE_CHECK_NAME} ${scoreLabel}`}
        why={confidence.summary?.trim() || FALLBACK_WHY}
        tone={confidenceBandForScore(confidence.score)}
      >
        <span className="flex items-baseline gap-0.5 leading-none">
          <span className="text-[22px] font-semibold tabular-nums text-foreground">{confidence.score}</span>
          <span className="text-[12px] text-muted-foreground">/{maxScore}</span>
        </span>
        <ConfidenceMeter
          score={confidence.score}
          testId="split-run-intent-confidence-meter"
          showTooltip={false}
          decorative
          size="lg"
        />
      </ConfidenceWhyChip>
    </ConfidenceFooterShell>
  );
}

function ConfidenceWhyChip({
  label,
  why,
  tone,
  children,
}: {
  label: string;
  why: string;
  tone?: ConfidenceBand;
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
          data-testid="split-run-intent-confidence-chip"
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
        <p className="text-[12px] text-muted-foreground">{CONFIDENCE_CHECK_NAME}</p>
        <p className="mt-1 text-[13px] leading-5 text-foreground" data-testid="split-run-intent-confidence-copy">
          {why}
        </p>
      </HoverCardContent>
    </HoverCard>
  );
}

function ConfidenceFooterShell({ children, flush }: { children: ReactNode; flush?: boolean }) {
  return (
    <footer
      className={flush ? undefined : "shrink-0 border-t border-border px-5 py-3"}
      data-testid="split-run-overview-checks"
      aria-label={CONFIDENCE_CHECK_NAME}
    >
      {children}
    </footer>
  );
}
