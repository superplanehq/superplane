import type { ReactNode } from "react";
import type { MergeConfidenceCardCheck } from "../lib/mergeConfidenceScore";
import { ConfidenceAnalyzingIndicator } from "./ConfidenceMeter";
import { CardScoreBadges, MergeConfidenceChip } from "./ReadinessMark";

export function WorkOrderCardFooterStatus({
  queueStatus,
  ...scores
}: {
  queueStatus?: ReactNode;
  entryId: string;
  clarityScore?: number;
  confidenceScore?: number;
  showClarity?: boolean;
  showConfidenceScore?: boolean;
  mergeConfidence?: { score: number; maxScore: number; checks?: MergeConfidenceCardCheck[] };
  reviewMetrics?: { key: string; name: string; score: number }[];
  isAnalyzing: boolean;
}) {
  if (queueStatus) return queueStatus;
  const hasScore =
    (scores.showClarity && scores.clarityScore != null) ||
    (scores.showConfidenceScore && scores.confidenceScore != null);
  if (!hasScore && !scores.isAnalyzing && !scores.mergeConfidence) return null;
  return (
    <div className="flex shrink-0 items-center gap-1.5">
      {scores.mergeConfidence ? (
        <MergeConfidenceChip
          score={scores.mergeConfidence.score}
          maxScore={scores.mergeConfidence.maxScore}
          checks={scores.mergeConfidence.checks}
          testId={`work-order-card-merge-${scores.entryId}`}
        />
      ) : null}
      {hasScore || scores.isAnalyzing ? (
        <CardScores
          entryId={scores.entryId}
          clarity={scores.clarityScore}
          confidence={scores.confidenceScore}
          showClarity={scores.showClarity}
          showConfidence={scores.showConfidenceScore}
          reviewMetrics={scores.reviewMetrics}
          isAnalyzing={scores.isAnalyzing}
        />
      ) : null}
    </div>
  );
}

function CardScores({
  entryId,
  clarity,
  confidence,
  showClarity = true,
  showConfidence = true,
  reviewMetrics,
  isAnalyzing,
}: {
  entryId: string;
  clarity?: number;
  confidence?: number;
  showClarity?: boolean;
  showConfidence?: boolean;
  reviewMetrics?: { key: string; name: string; score: number }[];
  isAnalyzing: boolean;
}) {
  if (isAnalyzing) {
    return (
      <ConfidenceAnalyzingIndicator
        className="shrink-0"
        testId={`work-order-card-analyzing-${entryId}`}
        showThinkingStates
        showTooltip={false}
      />
    );
  }
  if (clarity == null && confidence == null) {
    return null;
  }
  return (
    <CardScoreBadges
      clarity={clarity}
      confidence={confidence}
      showClarity={showClarity}
      showConfidence={showConfidence}
      reviewMetrics={reviewMetrics}
      testId={`work-order-card-score-${entryId}`}
    />
  );
}
