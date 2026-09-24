import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { SCORE_PAIR_LABEL } from "../../workOrders/ConfidenceMeter";
import { ScoreEvidenceRow, type ScoreEvidenceValue } from "../../workOrders/ScoreEvidence";

const FOOTER_TEST_IDS = {
  clarity: "split-run-intent-clarity",
  confidence: "split-run-intent-confidence",
} as const;

type WorkOrderIntentConfidenceFooterProps = {
  clarity?: WorkOrderCheckPresentation;
  confidence?: WorkOrderCheckPresentation;
  showClarity?: boolean;
  showConfidence?: boolean;
  isAnalyzing: boolean;
  flush?: boolean;
};

function evidenceValue(check?: WorkOrderCheckPresentation): ScoreEvidenceValue | undefined {
  if (!check) {
    return undefined;
  }
  return { score: check.score, summary: check.summary };
}

/**
 * Plan-pane footer: the two scores as evidence. The decision note below it
 * carries the verdict, so this row only shows the numbers and their summaries.
 */
export function WorkOrderIntentConfidenceFooter({
  clarity,
  confidence,
  showClarity = true,
  showConfidence = true,
  isAnalyzing,
  flush = false,
}: WorkOrderIntentConfidenceFooterProps) {
  const visibleClarity = showClarity ? clarity : undefined;
  const visibleConfidence = showConfidence ? confidence : undefined;
  const hasScore = Boolean(visibleClarity || visibleConfidence);

  if (!hasScore && !isAnalyzing) {
    return (
      <ScoreFooterShell flush={flush}>
        <p className="text-[13px] text-muted-foreground">No scores yet.</p>
      </ScoreFooterShell>
    );
  }

  return (
    <ScoreFooterShell flush={flush}>
      <ScoreEvidenceRow
        clarity={evidenceValue(visibleClarity)}
        confidence={evidenceValue(visibleConfidence)}
        showClarity={showClarity}
        showConfidence={showConfidence}
        isAnalyzing={isAnalyzing && !hasScore}
        testIds={FOOTER_TEST_IDS}
        className="gap-x-3 text-[13px]"
      />
    </ScoreFooterShell>
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
