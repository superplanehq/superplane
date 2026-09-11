import { CONFIDENCE_CHECK_NAME } from "./confidenceScore";
import { INTENT_ARTIFACT_NAME, SPEC_ARTIFACT_NAME } from "./intentDocument";
import {
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  toArtifactDataRecord,
} from "./workOrderArtifact";

const CONFIDENCE_CHECK_KEY = "confidence";

export function hasAnalysisScore(
  checks?: Array<{ name?: string; key?: string; score?: number | null }>,
): boolean {
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

export function analysisFinishedStatus<T extends string>(status: T, delivered: boolean): T | "passed" {
  if (delivered && (status === "failed" || status === "cancelled")) {
    return "passed";
  }
  return status;
}
