import { useState, type ReactNode } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/ui/hoverCard";

import {
  CLARITY_CHECK_NAME,
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  type ConfidenceBand,
} from "../lib/confidenceScore";
import { SCORE_KINDS, scoreSummaryText, type ScoreKind, type ScoreSummaryValue } from "../lib/scoreSummary";
import {
  CONFIDENCE_ANALYZING_LABEL,
  CONFIDENCE_ANALYZING_TOOLTIP,
  ConfidenceAnalyzingIndicator,
  ConfidenceMeter,
} from "./ConfidenceMeter";

const ITEM_CLASS = "inline-flex h-7 items-center gap-1.5 px-1.5 text-[12px] leading-none";

export type ScoreEvidenceValue = ScoreSummaryValue;

const SCORE_LABEL: Record<ScoreKind, { short: string; name: string }> = {
  clarity: { short: "Clarity", name: CLARITY_CHECK_NAME },
  confidence: { short: "Confidence", name: CONFIDENCE_CHECK_NAME },
};

const NUMBER_TONE: Record<ConfidenceBand, string> = {
  High: "text-success",
  Medium: "text-warning",
  Low: "text-destructive",
};

/**
 * Two labeled score items: name, five bars, number. Hover peeks the summary,
 * click pins it. The refine strip, the classic draft footer, and previews
 * share this row so the scores read the same everywhere.
 */
export function ScoreEvidenceRow({
  clarity,
  confidence,
  isAnalyzing = false,
  testIds,
  className,
}: {
  clarity?: ScoreEvidenceValue;
  confidence?: ScoreEvidenceValue;
  isAnalyzing?: boolean;
  testIds?: Partial<Record<ScoreKind, string>>;
  className?: string;
}) {
  const values: Record<ScoreKind, ScoreEvidenceValue | undefined> = { clarity, confidence };
  return (
    <div
      className={cn("flex min-w-0 flex-wrap items-center gap-x-1 gap-y-1", className)}
      data-testid="score-evidence-row"
    >
      {SCORE_KINDS.map((kind) => (
        <ScoreEvidence key={kind} kind={kind} value={values[kind]} isAnalyzing={isAnalyzing} testId={testIds?.[kind]} />
      ))}
    </div>
  );
}

export function ScoreEvidence({
  kind,
  value,
  isAnalyzing = false,
  testId,
}: {
  kind: ScoreKind;
  value?: ScoreEvidenceValue;
  isAnalyzing?: boolean;
  testId?: string;
}) {
  const label = SCORE_LABEL[kind];
  const score = value?.score;
  const summary = scoreSummaryText(kind, value);

  if (isAnalyzing) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className={ITEM_CLASS} aria-label={`${label.short}. ${CONFIDENCE_ANALYZING_LABEL}`} role="status">
            <ScoreEvidenceLabel>{label.short}</ScoreEvidenceLabel>
            <ConfidenceAnalyzingIndicator
              testId={testId ? `${testId}-analyzing` : undefined}
              showTooltip={false}
              decorative
              className="shrink-0"
            />
          </span>
        </TooltipTrigger>
        <TooltipContent>{CONFIDENCE_ANALYZING_TOOLTIP}</TooltipContent>
      </Tooltip>
    );
  }

  if (score == null || !summary) {
    return (
      <span className={ITEM_CLASS} aria-label={`${label.short}. No score yet`} data-testid={testId}>
        <ScoreEvidenceLabel>{label.short}</ScoreEvidenceLabel>
        <span className="tabular-nums text-muted-foreground">–</span>
      </span>
    );
  }

  return (
    <ScoreEvidenceCard
      name={label.name}
      body={summary}
      label={`${label.short} ${score}/${CONFIDENCE_SCORE_MAX}`}
      testId={testId}
    >
      <ScoreEvidenceLabel>{label.short}</ScoreEvidenceLabel>
      <ConfidenceMeter
        score={score}
        label={label.name}
        showTooltip={false}
        decorative
        testId={testId ? `${testId}-meter` : undefined}
      />
      <span className={cn("tabular-nums font-semibold", NUMBER_TONE[confidenceBandForScore(score)])}>{score}</span>
    </ScoreEvidenceCard>
  );
}

function ScoreEvidenceLabel({ children }: { children: ReactNode }) {
  return <span className="text-muted-foreground">{children}</span>;
}

/** Hover peeks, click pins. A pinned card stays until the next click. */
function ScoreEvidenceCard({
  name,
  body,
  label,
  testId,
  children,
}: {
  name: string;
  body: string;
  label: string;
  testId?: string;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const [pinned, setPinned] = useState(false);

  const onOpenChange = (next: boolean) => {
    if (pinned && !next) {
      return;
    }
    setOpen(next);
  };

  const togglePinned = () => {
    setPinned((current) => {
      const next = !current;
      setOpen(next);
      return next;
    });
  };

  return (
    <HoverCard open={open} onOpenChange={onOpenChange} openDelay={0} closeDelay={80}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          aria-label={label}
          aria-expanded={open}
          aria-pressed={pinned}
          data-testid={testId}
          data-state={open ? "open" : "closed"}
          className={cn(
            ITEM_CLASS,
            "rounded-md transition-colors hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:outline-none",
            pinned && "bg-muted/70",
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
        <p className="mt-1 text-[13px] leading-5 text-foreground" data-testid={testId ? `${testId}-copy` : undefined}>
          {body}
        </p>
      </HoverCardContent>
    </HoverCard>
  );
}
