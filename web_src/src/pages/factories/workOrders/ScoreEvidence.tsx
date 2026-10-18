import { useState, type ComponentPropsWithRef, type ReactNode } from "react";
import { ChevronDown } from "lucide-react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/ui/hoverCard";

import {
  CLARITY_CHECK_NAME,
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  type ConfidenceBand,
} from "../lib/confidenceScore";
import { workOrderCheckStatus, type WorkOrderCheckPresentation } from "../lib/workOrderChecks";
import { SCORE_KINDS, scoreSummaryText, type ScoreKind, type ScoreSummaryValue } from "../lib/scoreSummary";
import {
  CheckStatusBars,
  CONFIDENCE_ANALYZING_LABEL,
  CONFIDENCE_ANALYZING_TOOLTIP,
  ConfidenceMeter,
} from "./ConfidenceMeter";

const ITEM_CLASS = "inline-flex h-7 items-center gap-1.5 px-1.5 text-[12px] leading-none";

const CHIP_BUTTON_CLASS =
  "rounded-md transition-colors hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:outline-none";

export type ScoreEvidenceValue = ScoreSummaryValue & {
  /** Top of the scale: 5 for legacy scores, 3 for review scores. */
  maxScore?: number;
};

/**
 * External control for inline confidence checks in the plan card.
 */
export type ScoreEvidenceDetails = {
  open: boolean;
  controlsId: string;
  onToggle: () => void;
  checks: readonly WorkOrderCheckPresentation[];
};

const SCORE_LABEL: Record<ScoreKind, { short: string; name: string }> = {
  clarity: { short: "Clarity", name: CLARITY_CHECK_NAME },
  confidence: { short: "Confidence", name: CONFIDENCE_CHECK_NAME },
};

const NUMBER_TONE: Record<ConfidenceBand, string> = {
  High: "text-success",
  Medium: "text-amber-700 dark:text-amber-400",
  Low: "text-destructive",
};

/**
 * Two labeled score items: name, bars, number. The plan card can show
 * confidence checks inline; other surfaces keep the existing score summary.
 */
export function ScoreEvidenceRow({
  clarity,
  confidence,
  isAnalyzing = false,
  showClarity = true,
  showConfidence = true,
  confidenceDetails,
  testIds,
  className,
}: {
  clarity?: ScoreEvidenceValue;
  confidence?: ScoreEvidenceValue;
  isAnalyzing?: boolean;
  showClarity?: boolean;
  showConfidence?: boolean;
  /** Hands the confidence chip over to inline checks in the plan card. */
  confidenceDetails?: ScoreEvidenceDetails;
  testIds?: Partial<Record<ScoreKind, string>>;
  className?: string;
}) {
  const values: Record<ScoreKind, ScoreEvidenceValue | undefined> = { clarity, confidence };
  return (
    <div
      className={cn("flex min-w-0 flex-wrap items-center gap-x-1 gap-y-1", className)}
      data-testid="score-evidence-row"
    >
      {SCORE_KINDS.map((kind) => {
        if (kind === "clarity" && !showClarity) {
          return null;
        }
        if (kind === "confidence" && !showConfidence) {
          return null;
        }
        return (
          <ScoreEvidence
            key={kind}
            kind={kind}
            value={values[kind]}
            isAnalyzing={isAnalyzing}
            testId={testIds?.[kind]}
            details={kind === "confidence" ? confidenceDetails : undefined}
          />
        );
      })}
    </div>
  );
}

export function ScoreEvidence({
  kind,
  value,
  isAnalyzing = false,
  testId,
  details,
}: {
  kind: ScoreKind;
  value?: ScoreEvidenceValue;
  isAnalyzing?: boolean;
  testId?: string;
  details?: ScoreEvidenceDetails;
}) {
  const label = SCORE_LABEL[kind];
  const score = value?.score;
  const summary = scoreSummaryText(kind, value);

  // The verdict above already animates while the agent works. The score
  // slots stay quiet and show the same dash as a missing score.
  if (isAnalyzing) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <ScoreEvidencePlaceholder
            label={label.short}
            ariaLabel={`${label.short}. ${CONFIDENCE_ANALYZING_LABEL}`}
            role="status"
            testId={testId ? `${testId}-analyzing` : undefined}
          />
        </TooltipTrigger>
        <TooltipContent>{CONFIDENCE_ANALYZING_TOOLTIP}</TooltipContent>
      </Tooltip>
    );
  }

  if (score == null || !summary) {
    return <ScoreEvidencePlaceholder label={label.short} ariaLabel={`${label.short}. No score yet`} testId={testId} />;
  }

  const max = value?.maxScore || CONFIDENCE_SCORE_MAX;
  if (details) {
    return <ScoreEvidenceDetailsChip testId={testId} details={details} />;
  }

  const chipLabel = `${label.short} ${score}/${max}`;

  const content = (
    <>
      <ScoreEvidenceLabel>{label.short}</ScoreEvidenceLabel>
      <ConfidenceMeter
        score={score}
        max={max}
        label={label.name}
        showTooltip={false}
        decorative
        testId={testId ? `${testId}-meter` : undefined}
      />
      <span className={cn("tabular-nums font-semibold", NUMBER_TONE[confidenceBandForScore(score, max)])}>{score}</span>
    </>
  );

  return (
    <ScoreEvidenceCard name={label.name} body={summary} label={chipLabel} testId={testId}>
      {content}
    </ScoreEvidenceCard>
  );
}

/**
 * The plan card score is a disclosure control with an explicit action.
 */
function ScoreEvidenceDetailsChip({ testId, details }: { testId?: string; details: ScoreEvidenceDetails }) {
  const checkLabels = details.checks.map((check) => `${check.name} ${workOrderCheckStatus(check).label.toLowerCase()}`);
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      aria-label={`Confidence checks: ${checkLabels.join("; ")}. ${details.open ? "Hide checks" : "Show checks"}`}
      aria-expanded={details.open}
      aria-controls={details.open ? details.controlsId : undefined}
      data-testid={testId}
      data-state={details.open ? "open" : "closed"}
      className={cn(
        CHIP_BUTTON_CLASS,
        "group h-auto min-h-6 min-w-0 shrink flex-wrap justify-start gap-x-2 gap-y-0.5 whitespace-normal px-2 py-0.5 text-left",
      )}
      onClick={(event) => {
        event.preventDefault();
        details.onToggle();
      }}
    >
      <span className="inline-flex items-center gap-1.5 text-[12px] leading-4 font-medium text-muted-foreground">
        Confidence
        <CheckStatusBars checks={details.checks} testId={testId ? `${testId}-meter` : undefined} />
      </span>
      <span className="inline-flex items-center gap-1 border-l border-border/70 pl-2 text-[11px] font-medium text-muted-foreground group-hover:text-foreground group-focus-visible:text-foreground">
        {details.open ? "Hide checks" : "Show checks"}
        <ChevronDown className={cn("size-3 transition-transform", details.open && "rotate-180")} aria-hidden="true" />
      </span>
    </Button>
  );
}

function ScoreEvidenceLabel({ children }: { children: ReactNode }) {
  return <span className="text-muted-foreground">{children}</span>;
}

/** Spreads the rest so a TooltipTrigger can attach its ref and handlers. */
function ScoreEvidencePlaceholder({
  label,
  ariaLabel,
  testId,
  ...rest
}: {
  label: string;
  ariaLabel: string;
  testId?: string;
} & ComponentPropsWithRef<"span">) {
  return (
    <span {...rest} className={ITEM_CLASS} aria-label={ariaLabel} data-testid={testId}>
      <ScoreEvidenceLabel>{label}</ScoreEvidenceLabel>
      <span className="tabular-nums text-muted-foreground">–</span>
    </span>
  );
}

/**
 * Hover peeks, click pins. A pinned card stays until the next click.
 * The card floats above the trigger, so it never pushes the composer.
 */
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
          className={cn(ITEM_CLASS, CHIP_BUTTON_CLASS, pinned && "bg-muted/70")}
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
        align="end"
        sideOffset={8}
        collisionPadding={12}
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
