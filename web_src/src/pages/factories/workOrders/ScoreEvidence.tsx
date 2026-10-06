import { useState, type ComponentPropsWithRef, type ReactNode } from "react";

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
import { CONFIDENCE_ANALYZING_LABEL, CONFIDENCE_ANALYZING_TOOLTIP, ConfidenceMeter } from "./ConfidenceMeter";

const ITEM_CLASS = "inline-flex h-7 items-center gap-1.5 px-1.5 text-[12px] leading-none";

const CHIP_BUTTON_CLASS =
  "rounded-md transition-colors hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:outline-none";

export type ScoreEvidenceValue = ScoreSummaryValue;

/**
 * External control for the confidence drawer. The chip reports hover and
 * click; the owner renders the drawer wherever it belongs (for example
 * above the plan card).
 */
export type ScoreEvidenceDrawer = {
  open: boolean;
  pinned: boolean;
  controlsId: string;
  onHoverChange: (hovering: boolean) => void;
  onToggle: () => void;
};

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
 * click pins it. The refine strip, the draft footer, and previews share this
 * row so the scores read the same everywhere.
 */
export function ScoreEvidenceRow({
  clarity,
  confidence,
  isAnalyzing = false,
  showClarity = true,
  showConfidence = true,
  confidenceDrawer,
  testIds,
  className,
}: {
  clarity?: ScoreEvidenceValue;
  confidence?: ScoreEvidenceValue;
  isAnalyzing?: boolean;
  showClarity?: boolean;
  showConfidence?: boolean;
  /** Hands the confidence chip over to an owner-rendered drawer. */
  confidenceDrawer?: ScoreEvidenceDrawer;
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
            drawer={kind === "confidence" ? confidenceDrawer : undefined}
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
  drawer,
}: {
  kind: ScoreKind;
  value?: ScoreEvidenceValue;
  isAnalyzing?: boolean;
  testId?: string;
  drawer?: ScoreEvidenceDrawer;
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

  const chipLabel = `${label.short} ${score}/${CONFIDENCE_SCORE_MAX}`;
  const content = (
    <>
      <ScoreEvidenceLabel>{label.short}</ScoreEvidenceLabel>
      <ConfidenceMeter
        score={score}
        label={label.name}
        showTooltip={false}
        decorative
        testId={testId ? `${testId}-meter` : undefined}
      />
      <span className={cn("tabular-nums font-semibold", NUMBER_TONE[confidenceBandForScore(score)])}>{score}</span>
    </>
  );

  if (drawer) {
    return (
      <ScoreEvidenceDrawerChip label={chipLabel} testId={testId} drawer={drawer}>
        {content}
      </ScoreEvidenceDrawerChip>
    );
  }

  return (
    <ScoreEvidenceCard name={label.name} body={summary} label={chipLabel} testId={testId}>
      {content}
    </ScoreEvidenceCard>
  );
}

/**
 * The chip for an owner-rendered drawer. Hover peeks the drawer, click
 * pins it; the owner holds the state and renders the drawer itself.
 */
function ScoreEvidenceDrawerChip({
  label,
  testId,
  drawer,
  children,
}: {
  label: string;
  testId?: string;
  drawer: ScoreEvidenceDrawer;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      aria-expanded={drawer.open}
      aria-pressed={drawer.pinned}
      aria-controls={drawer.open ? drawer.controlsId : undefined}
      data-testid={testId}
      data-state={drawer.open ? "open" : "closed"}
      className={cn(ITEM_CLASS, CHIP_BUTTON_CLASS, drawer.pinned && "bg-muted/70")}
      onMouseEnter={() => drawer.onHoverChange(true)}
      onMouseLeave={() => drawer.onHoverChange(false)}
      onFocus={() => drawer.onHoverChange(true)}
      onBlur={() => drawer.onHoverChange(false)}
      onClick={(event) => {
        event.preventDefault();
        drawer.onToggle();
      }}
    >
      {children}
    </button>
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
