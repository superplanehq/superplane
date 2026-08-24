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
  /** Markdown implementation plan shown on the Plan tab. */
  planMarkdown: string;
  summary: string;
  readyNote: string;
  sections: ReviewCandidateSection[];
  noBlockingQuestions: string;
}

export type ReviewCandidateTab = "plan" | "ticket" | "analysis";

export const REVIEW_CANDIDATE_COPY = {
  tabsLabel: "Ticket review",
  planTab: "Plan",
  ticketTab: "Ticket",
  analysisTab: "Analysis Run",
  ticketSource: "GitHub Issues",
  ticketRepository: "acme/payments-service",
  openIssue: "Open issue on GitHub",
  opened: "Opened",
  updated: "Updated",
  author: "Author",
  assignees: "Assignees",
  noAssignees: "No assignees",
  reasonsHeading: "Why SuperPlane can implement this",
  planHeading: "Implementation plan",
  planFile: "plan.md",
  editPlan: "Edit",
  donePlan: "Done",
  planEditorLabel: "Plan markdown",
  back: "Back to results",
  approve: "Approve plan and start",
  approved: "Plan approved",
} as const;

export function githubIssueUrl(repository: string, ticketKey: string): string {
  return `https://github.com/${repository}/issues/${ticketKey.replace(/\D/g, "")}`;
}

export function isReviewCandidateTab(value: string): value is ReviewCandidateTab {
  return value === "plan" || value === "ticket" || value === "analysis";
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
