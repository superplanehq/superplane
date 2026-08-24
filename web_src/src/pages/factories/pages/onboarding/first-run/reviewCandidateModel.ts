import { confidenceBandForScore as bandForConfidenceScore, type ConfidenceBand } from "../../../lib/confidenceScore";

export type ReviewConfidenceBand = ConfidenceBand;

export interface ReviewCandidateSection {
  number: string;
  title: string;
  intro: string;
  items: string[];
}

export interface ReviewIssuePerson {
  name: string;
  login: string;
  avatarUrl?: string;
}

export interface ReviewIssueLabel {
  name: string;
}

export interface ReviewIssue {
  url: string;
  bodyMarkdown: string;
  createdAt: string;
  updatedAt: string;
  author: ReviewIssuePerson;
  assignees: ReviewIssuePerson[];
  labels: ReviewIssueLabel[];
}

export interface ReviewCandidate {
  workOrderId: string;
  ticketKey: string;
  title: string;
  ticketBody: string;
  issue: ReviewIssue;
  confidenceScore: number;
  confidenceBand: ReviewConfidenceBand;
  /** Three reasons SuperPlane can implement this ticket. */
  reasons: [string, string, string];
  /** Markdown implementation plan attached as plan.md on Create plan. */
  planMarkdown: string;
  summary: string;
  readyNote: string;
  sections: ReviewCandidateSection[];
  noBlockingQuestions: string;
}

export const REVIEW_CANDIDATE_COPY = {
  ticketRepository: "acme/payments-service",
} as const;

export function githubIssueUrl(repository: string, ticketKey: string): string {
  return `https://github.com/${repository}/issues/${ticketKey.replace(/\D/g, "")}`;
}

export function confidenceBandForScore(score: number): ReviewConfidenceBand {
  return bandForConfidenceScore(score);
}

export function implementationPlanMarkdown(parts: {
  goal: string;
  files: readonly string[];
  steps: readonly string[];
  verify: readonly string[];
}): string {
  return [
    "## Goal",
    "",
    parts.goal,
    "",
    "## Files to change",
    "",
    ...parts.files.map((file) => `- ${file}`),
    "",
    "## Steps",
    "",
    ...parts.steps.map((step, index) => `${index + 1}. ${step}`),
    "",
    "## Verify",
    "",
    ...parts.verify.map((item) => `- ${item}`),
  ].join("\n");
}

export function confidenceBandClassName(band: ReviewConfidenceBand): string {
  if (band === "High") {
    return "text-emerald-700 dark:text-emerald-300";
  }
  if (band === "Medium") {
    return "text-orange-600 dark:text-orange-400";
  }
  return "text-red-600 dark:text-red-400";
}
