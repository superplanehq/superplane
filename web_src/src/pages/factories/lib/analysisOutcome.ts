import { CONFIDENCE_CHECK_NAME } from "./confidenceScore";
import { INTENT_ARTIFACT_NAME, SPEC_ARTIFACT_NAME } from "./intentDocument";
import {
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  toArtifactDataRecord,
} from "./workOrderArtifact";

const CONFIDENCE_CHECK_KEY = "confidence";

export function hasAnalysisScore(checks?: Array<{ name?: string; key?: string; score?: number | null }>): boolean {
  return (checks ?? []).some((check) => {
    const named = check.name === CONFIDENCE_CHECK_NAME || check.key === CONFIDENCE_CHECK_KEY;
    return named && check.score != null;
  });
}

export function hasAnalysisPlan(artifacts?: Array<{ data?: unknown }>): boolean {
  return (artifacts ?? []).some((artifact) => {
    const data = toArtifactDataRecord(artifact.data);
    const name = extractArtifactName(data) ?? extractArtifactTitle(data) ?? "";
    const body = extractArtifactMarkdownBody(data)?.trim() ?? "";
    return (name === SPEC_ARTIFACT_NAME || name === INTENT_ARTIFACT_NAME) && body.length > 0;
  });
}

/** First analysis result: one score and a written plan. */
export function analysisFirstResultDelivered(input: {
  checks?: Array<{ name?: string; key?: string; score?: number | null }>;
  artifacts?: Array<{ data?: unknown }>;
}): boolean {
  return hasAnalysisScore(input.checks) && hasAnalysisPlan(input.artifacts);
}

type AnalysisScoreCheck = {
  name?: string;
  key?: string;
  score?: number | null;
  runId?: string;
};

function analysisScoreRunId(checks?: AnalysisScoreCheck[]): string | undefined {
  const check = (checks ?? []).find((entry) => {
    const named = entry.name === CONFIDENCE_CHECK_NAME || entry.key === CONFIDENCE_CHECK_KEY;
    return named && entry.score != null && Boolean(entry.runId);
  });
  return check?.runId;
}

/** Score and plan belong to the run that reported the score, not the work order. */
export function analysisResultDeliveredForRun(
  run: { id?: string },
  input: {
    checks?: AnalysisScoreCheck[];
    artifacts?: Array<{ data?: unknown }>;
    isLast: boolean;
  },
): boolean {
  if (!analysisFirstResultDelivered(input)) {
    return false;
  }
  const scoreRunId = analysisScoreRunId(input.checks);
  if (scoreRunId) {
    return run.id === scoreRunId;
  }
  return input.isLast;
}

export function analysisFinishedStatus<T extends string>(status: T, delivered: boolean): T | "passed" {
  if (delivered && (status === "failed" || status === "cancelled")) {
    return "passed";
  }
  return status;
}

type AnalysisCanvasRun = {
  result?: string;
  state?: string;
  cancelledBy?: { id?: string } | null;
};

/** Timed-out analysis stays running until a score and plan exist. A crash or user stop stays failed. */
export function statusForAnalysisRun<T extends string>(
  run: AnalysisCanvasRun,
  status: T,
  delivered: boolean,
): T | "passed" | "running" {
  if (delivered) {
    return analysisFinishedStatus(status, delivered);
  }
  if (isUnfinishedCancelledAnalysis(run)) {
    return "running";
  }
  return status;
}

export function isUnfinishedCancelledAnalysis(run: AnalysisCanvasRun, delivered = false): boolean {
  if (delivered || hasExplicitAnalysisCancellation(run)) {
    return false;
  }
  return isCancelledAnalysis(run);
}

export function isCancelledAnalysis(run: AnalysisCanvasRun): boolean {
  if (run.result === "RESULT_CANCELLED") {
    return true;
  }
  return run.state === "STATE_CANCELLING" && run.result !== "RESULT_PASSED" && run.result !== "RESULT_FAILED";
}

function hasExplicitAnalysisCancellation(run: AnalysisCanvasRun): boolean {
  return Boolean(run.cancelledBy?.id);
}
