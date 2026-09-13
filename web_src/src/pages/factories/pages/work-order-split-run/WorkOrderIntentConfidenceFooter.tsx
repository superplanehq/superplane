import type { ReactNode } from "react";

import { CONFIDENCE_CHECK_NAME, CONFIDENCE_SCORE_MAX } from "../../lib/confidenceScore";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { ConfidenceAnalyzingIndicator, ConfidenceMeter } from "../../workOrders/ConfidenceMeter";

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
        <div className="flex items-start gap-2">
          <ConfidenceAnalyzingIndicator testId="split-run-intent-confidence-meter" />
          <p className="text-[13px] leading-5 text-muted-foreground">The analysis is still running.</p>
        </div>
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

  return (
    <ConfidenceFooterShell flush={flush}>
      <div key={`${confidence.score}-${confidence.summary ?? ""}`} className="sp-text-reveal flex items-start gap-2.5">
        <div className="flex shrink-0 flex-col items-start gap-1 pt-0.5">
          <ConfidenceMeter score={confidence.score} testId="split-run-intent-confidence-meter" />
          <span className="text-[12px] tabular-nums text-muted-foreground">
            {confidence.score}/{confidence.maxScore || CONFIDENCE_SCORE_MAX}
          </span>
        </div>
        <div className="min-w-0">
          <p className="text-[12px] text-muted-foreground">{CONFIDENCE_CHECK_NAME}</p>
          <p className="text-[13px] leading-5 text-foreground" data-testid="split-run-intent-confidence-copy">
            {confidence.summary?.trim() || "The analysis scored how clear this work is."}
          </p>
        </div>
      </div>
    </ConfidenceFooterShell>
  );
}

function ConfidenceFooterShell({ children, flush }: { children: ReactNode; flush?: boolean }) {
  return (
    <footer
      className={flush ? "px-5 pt-3" : "shrink-0 border-t border-border px-5 py-3"}
      data-testid="split-run-overview-checks"
      aria-label={CONFIDENCE_CHECK_NAME}
    >
      {children}
    </footer>
  );
}
