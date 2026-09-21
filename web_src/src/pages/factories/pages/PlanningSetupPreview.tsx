import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import { planningSetupPreviewCaption, type PlanningSetupStep } from "./planningSetupCaption";
import { PRFeedbackSetupPreviewPane } from "./PRFeedbackSetupWizardChrome";

export function PlanningSetupPreview({
  step,
  enabled,
  clarity,
  confidence,
}: {
  step: PlanningSetupStep;
  enabled: boolean;
  clarity: boolean;
  confidence: boolean;
}) {
  const caption = planningSetupPreviewCaption({ step, enabled, clarity, confidence });
  const showScores = step === "scores";
  const showSource = step === "refine" && !enabled;

  return (
    <PRFeedbackSetupPreviewPane
      label={PLANNING_SETTINGS_COPY.wizardPreviewLabel}
      caption={caption}
      testId="planning-setup-preview"
      captionTestId="planning-setup-preview-caption"
    >
      <div className="w-full max-w-sm space-y-3" data-testid="planning-setup-preview-card">
        {showSource ? (
          <SourcePreview />
        ) : (
          <PlanPreview showScores={showScores} clarity={clarity} confidence={confidence} />
        )}
      </div>
    </PRFeedbackSetupPreviewPane>
  );
}

function PlanPreview({
  showScores,
  clarity,
  confidence,
}: {
  showScores: boolean;
  clarity: boolean;
  confidence: boolean;
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-4 shadow-sm">
      <p className="text-[11px] font-medium uppercase tracking-[0.12em] text-muted-foreground">Chat</p>
      <div className="mt-3 space-y-2">
        <PreviewBubble align="end">Add a refund retry when the bank times out.</PreviewBubble>
        <PreviewBubble align="start">I will write a plan for this draft.</PreviewBubble>
      </div>
      <div className="mt-4 rounded-lg border border-border bg-background px-3 py-2.5">
        <p className="text-[13px] font-medium text-foreground">Plan</p>
        <p className="mt-1 text-[12px] leading-5 text-muted-foreground">
          Retry the refund when the bank returns a timeout.
        </p>
        {showScores ? <ScoreChips clarity={clarity} confidence={confidence} /> : null}
      </div>
    </div>
  );
}

function SourcePreview() {
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card shadow-sm">
      <div className="grid grid-cols-[38%_1fr]">
        <div className="space-y-3 border-r border-border px-3 py-3">
          <p className="text-[11px] font-medium uppercase tracking-[0.12em] text-muted-foreground">Source</p>
          <p className="text-[11px] font-medium uppercase tracking-[0.12em] text-muted-foreground">Artifacts</p>
        </div>
        <div className="px-3 py-3">
          <p className="text-[13px] leading-5 text-foreground">Retry the refund when the bank returns a timeout.</p>
        </div>
      </div>
      <div className="flex items-center justify-between border-t border-border px-3 py-2.5">
        <p className="text-[12px] text-muted-foreground">This task is ready to start.</p>
        <span className="rounded-md bg-foreground px-2 py-1 text-[11px] font-medium text-background">Start</span>
      </div>
    </div>
  );
}

function ScoreChips({ clarity, confidence }: { clarity: boolean; confidence: boolean }) {
  if (!clarity && !confidence) {
    return null;
  }
  return (
    <div className="mt-2 flex flex-wrap gap-1.5" data-testid="planning-setup-preview-scores">
      {clarity ? <ScoreChip label="Clarity" value="72" /> : null}
      {confidence ? <ScoreChip label="Confidence" value="81" /> : null}
    </div>
  );
}

function ScoreChip({ label, value }: { label: string; value: string }) {
  return (
    <span className="rounded-full border border-border bg-background px-2 py-0.5 text-[11px] text-muted-foreground">
      {label} {value}
    </span>
  );
}

function PreviewBubble({ align, children }: { align: "start" | "end"; children: string }) {
  return (
    <p
      className={
        align === "end"
          ? "ml-8 rounded-lg bg-foreground px-2.5 py-1.5 text-[12px] leading-5 text-background"
          : "mr-8 rounded-lg bg-muted px-2.5 py-1.5 text-[12px] leading-5 text-foreground"
      }
    >
      {children}
    </p>
  );
}
