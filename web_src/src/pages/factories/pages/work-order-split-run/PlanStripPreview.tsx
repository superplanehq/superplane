import { useState, type ReactNode } from "react";
import { Bot, ChevronDown, Play } from "lucide-react";

import { Alert, AlertAction, AlertTitle } from "@/components/reui/alert";
import { Button } from "@/components/ui/button";

import { liveDraftReadiness, startEmphasisForTone, type StartEmphasis } from "../../lib/draftReadiness";
import { ConfidenceMeter } from "../../workOrders/ConfidenceMeter";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { ComposerPlanStack } from "./ComposerPlanControls";
import { ANALYSIS_PLANNING_COPY } from "./useAnalysisPlanningSession";

/**
 * Isolated strip playground. Compare the old one-bar Alert with the decision
 * strip (verdict, evidence, Plan toggle) in each readiness state.
 */
export type PlanStripLook =
  | "today"
  | "oneBar"
  | "oneBarOpen"
  | "chips"
  | "chipsOpen"
  | "stripBlocked"
  | "stripCaution"
  | "stripAnalyzing";

const TASK_TITLE = "Duplicate a task";
const REVIEW_HEADLINE = "Review the plan before you start";
const REVIEW_TEXT = "The work is still uncertain. Add more context, or start if you accept the risk.";
const SCORE = 2;
const CONFIDENCE_SCORE = 4;
const CLARITY_TEXT = "The prompt and the copy scope are not defined. Answer the questions in this session.";
const CONFIDENCE_TEXT = "The change is small and a similar action already exists to copy.";
const LOW_CONFIDENCE_TEXT = "The change crosses billing and the API and there is no test for the refund path.";

export function PlanStripPreview({ look }: { look: PlanStripLook }) {
  return (
    <div
      className="flex min-h-[420px] w-full max-w-[48rem] flex-col overflow-hidden rounded-lg border border-border bg-background shadow-sm"
      data-testid={`plan-strip-preview-${look}`}
    >
      <header className="shrink-0 border-b border-border px-5 py-3">
        <h2 className="truncate text-[16px] font-semibold tracking-[-0.02em]">{TASK_TITLE}</h2>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6 text-[14px] leading-6 text-foreground">
        <p className="mb-4 ml-auto max-w-[92%] rounded-2xl border px-3.5 py-2.5">
          I want to duplicate a task and run it with a different agent configuration.
        </p>
        <p>I scored this 2. Prompt, harness, and a second run still need a decision.</p>
      </div>
      <div className="mx-auto w-full max-w-5xl shrink-0 px-4 pb-3">
        <PreviewStrip look={look} />
        <FakeComposer />
      </div>
    </div>
  );
}

export function PlanStripPreviewToggle({ look = "oneBar" }: { look?: "oneBar" | "chips" }) {
  const [open, setOpen] = useState(false);
  return (
    <div
      className="flex min-h-[420px] w-full max-w-[48rem] flex-col overflow-hidden rounded-lg border border-border bg-background shadow-sm"
      data-testid={`plan-strip-preview-toggle-${look}`}
    >
      <header className="flex shrink-0 items-start justify-between gap-3 border-b border-border px-5 py-3">
        <h2 className="truncate text-[16px] font-semibold tracking-[-0.02em]">{TASK_TITLE}</h2>
      </header>
      <div className="min-h-0 flex-1 px-6 py-6 text-[14px] leading-6 text-muted-foreground">
        Click Show plan or Hide plan.
      </div>
      <div className="mx-auto w-full max-w-5xl shrink-0 px-4 pb-3">
        {look === "chips" ? (
          <ChipStrip open={open} onToggle={() => setOpen((current) => !current)} />
        ) : (
          <OneBarStrip open={open} onToggle={() => setOpen((current) => !current)} />
        )}
        <FakeComposer />
      </div>
    </div>
  );
}

function PreviewStrip({ look }: { look: PlanStripLook }) {
  if (look === "today") {
    return <TodayStrip />;
  }
  if (look === "chips" || look === "chipsOpen") {
    return <ChipStrip open={look === "chipsOpen"} />;
  }
  if (look === "stripBlocked") {
    return <ChipStrip open={false} clarity={2} confidence={CONFIDENCE_SCORE} />;
  }
  if (look === "stripCaution") {
    return <ChipStrip open={false} clarity={5} confidence={2} />;
  }
  if (look === "stripAnalyzing") {
    return <ChipStrip open={false} isAnalyzing />;
  }
  return <OneBarStrip open={look === "oneBarOpen"} />;
}

function ChipStrip({
  open,
  clarity = 5,
  confidence = CONFIDENCE_SCORE,
  isAnalyzing = false,
  onToggle,
}: {
  open: boolean;
  clarity?: number;
  confidence?: number;
  isAnalyzing?: boolean;
  onToggle?: () => void;
}) {
  const tone = liveDraftReadiness({ clarity, confidence, isAnalyzing }).tone;
  return (
    <ComposerPlanStack
      open={open}
      clarity={{ score: clarity, summary: clarity <= 2 ? CLARITY_TEXT : REVIEW_TEXT }}
      confidence={{ score: confidence, summary: confidence <= 2 ? LOW_CONFIDENCE_TEXT : CONFIDENCE_TEXT }}
      isAnalyzing={isAnalyzing}
      planStatus={open ? undefined : "updated"}
      onToggle={onToggle}
      actions={<StripStart emphasis={startEmphasisForTone(tone)} />}
      modelSelect={<FakeModelSelect />}
    />
  );
}

/** Start alone on the verdict row. Filled only when the verdict says go. */
function StripStart({ emphasis }: { emphasis: StartEmphasis }) {
  return (
    <Button type="button" size="sm" variant={emphasis === "filled" ? "default" : "outline"}>
      <Play className="size-3.5" aria-hidden />
      Start
    </Button>
  );
}

function FakeModelSelect() {
  return (
    <Button
      type="button"
      size="sm"
      variant="ghost"
      aria-label="Model: Auto"
      className="gap-1.5 text-muted-foreground hover:text-foreground"
    >
      <Bot className="size-4" aria-hidden />
      Auto
      <ChevronDown className="size-3 opacity-60" aria-hidden />
    </Button>
  );
}

function TodayStrip() {
  return (
    <div className="flex flex-col gap-2 py-2">
      <PlanTitleRow open={false} />
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex shrink-0 items-baseline gap-0.5 rounded-lg border border-red-500/30 bg-red-500/10 px-2.5 py-1.5">
          <span className="text-[22px] font-semibold tabular-nums">{SCORE}</span>
          <span className="text-[12px] text-muted-foreground">/5</span>
          <ConfidenceMeter score={SCORE} showTooltip={false} className="ml-1" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-[13px] font-medium leading-5">{REVIEW_HEADLINE}</p>
          <p className="mt-0.5 text-[12px] leading-4 text-muted-foreground">{REVIEW_TEXT}</p>
        </div>
        <DraftActions />
      </div>
    </div>
  );
}

function OneBarStrip({ open, onToggle }: { open: boolean; onToggle?: () => void }) {
  return (
    <div className="py-2">
      <PlanTitleRow open={open} onToggle={onToggle} actions={<DraftActions />} />
    </div>
  );
}

function PlanTitleRow({ open, onToggle, actions }: { open: boolean; onToggle?: () => void; actions?: ReactNode }) {
  const label = open ? CREATE_WITH_AGENT_COPY.hidePlan : CREATE_WITH_AGENT_COPY.showPlan;

  return (
    <Alert className="grid-cols-[minmax(0,1fr)_auto] items-center" data-testid="plan-strip-preview-alert">
      <div className="flex min-w-0 items-center gap-3">
        {open ? null : <AlertTitle className="min-w-0 truncate">{CREATE_WITH_AGENT_COPY.planUpdated}</AlertTitle>}
        <ConfidenceMeter score={SCORE} showTooltip={false} />
      </div>
      <AlertAction className="col-start-2 max-sm:mt-0 max-sm:justify-end">
        <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
          <Button type="button" variant="outline" size="sm" aria-expanded={open} aria-pressed={open} onClick={onToggle}>
            {label}
          </Button>
          {actions}
        </div>
      </AlertAction>
    </Alert>
  );
}

function DraftActions() {
  return (
    <>
      <Button type="button" size="sm" variant="outline">
        Archive
      </Button>
      <Button type="button" size="sm">
        Start
      </Button>
    </>
  );
}

function FakeComposer() {
  return (
    <div className="sp-user-note flex min-h-[3.5rem] items-center rounded-2xl border px-3.5 text-[13px] text-muted-foreground">
      {ANALYSIS_PLANNING_COPY.composerPlaceholder}
    </div>
  );
}
