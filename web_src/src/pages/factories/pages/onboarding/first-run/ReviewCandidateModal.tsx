import { Avatar } from "@/components/Avatar/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { formatRelative } from "@/lib/datetime";
import { getUserInitials } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ExternalLink, FileText, Pencil, Ticket, Workflow } from "lucide-react";
import { useMemo, useState } from "react";

import { intakeTicketAnalysisFixture } from "../../lineIntakeModel";
import { PopupHeader, PopupShell } from "../../work-order-popup-redesign/popupShared";
import { WorkOrderSplitRunBody } from "../../work-order-split-run/WorkOrderSplitRunPopup";
import {
  REVIEW_CANDIDATE_COPY,
  confidenceBandClassName,
  isReviewCandidateTab,
  type ReviewCandidate,
  type ReviewCandidateTab,
  type ReviewIssuePerson,
} from "./reviewCandidates";

interface ReviewCandidateModalProps {
  candidate: ReviewCandidate;
  onClose: () => void;
}

/**
 * Plan review after first-run analysis. The backlog card opens this
 * instead of the split-run popup. SuperPlane does not start work until
 * the user approves the plan.
 */
export function ReviewCandidateModal({ candidate, onClose }: ReviewCandidateModalProps) {
  const [approved, setApproved] = useState(false);
  const [tab, setTab] = useState<ReviewCandidateTab>("plan");
  const [planMarkdown, setPlanMarkdown] = useState(candidate.planMarkdown);
  const analysisFixture = useMemo(
    () => intakeTicketAnalysisFixture({ id: candidate.workOrderId, title: candidate.title }, { complete: true }),
    [candidate.title, candidate.workOrderId],
  );

  return (
    <PopupShell testId="review-candidate-modal" canvas fixed onDismiss={onClose}>
      <PopupHeader title={candidate.title} onClose={onClose}>
        <Tabs
          value={tab}
          onValueChange={(value) => {
            if (isReviewCandidateTab(value)) setTab(value);
          }}
          className="mt-3"
        >
          <TabsList aria-label={REVIEW_CANDIDATE_COPY.tabsLabel}>
            <TabsTrigger value="plan" data-testid="review-candidate-tab-plan">
              <FileText />
              {REVIEW_CANDIDATE_COPY.planTab}
            </TabsTrigger>
            <TabsTrigger value="ticket" data-testid="review-candidate-tab-ticket">
              <Ticket />
              {REVIEW_CANDIDATE_COPY.ticketTab}
            </TabsTrigger>
            <TabsTrigger value="analysis" data-testid="review-candidate-tab-analysis">
              <Workflow />
              {REVIEW_CANDIDATE_COPY.analysisTab}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </PopupHeader>

      {tab === "analysis" ? (
        <WorkOrderSplitRunBody fixture={analysisFixture} />
      ) : (
        <>
          <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
            {tab === "ticket" ? (
              <TicketPanel candidate={candidate} />
            ) : (
              <PlanPanel candidate={candidate} planMarkdown={planMarkdown} onPlanMarkdownChange={setPlanMarkdown} />
            )}
          </div>
          <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
            <Button type="button" variant="outline" onClick={onClose}>
              {REVIEW_CANDIDATE_COPY.back}
            </Button>
            <Button type="button" disabled={approved} onClick={() => setApproved(true)}>
              {approved ? REVIEW_CANDIDATE_COPY.approved : REVIEW_CANDIDATE_COPY.approve}
            </Button>
          </footer>
        </>
      )}
    </PopupShell>
  );
}

function PlanPanel({
  candidate,
  planMarkdown,
  onPlanMarkdownChange,
}: {
  candidate: ReviewCandidate;
  planMarkdown: string;
  onPlanMarkdownChange: (next: string) => void;
}) {
  const [editingPlan, setEditingPlan] = useState(false);

  return (
    <>
      <p className="text-[22px] font-semibold tracking-[-0.03em] text-foreground" data-testid="review-candidate-score">
        <span className="tabular-nums">{candidate.confidencePct}%</span>
        <span className={cn("ml-2 text-[15px] font-medium", confidenceBandClassName(candidate.confidenceBand))}>
          {candidate.confidenceBand}
        </span>
      </p>
      <h3 className="mt-5 text-[13px] font-semibold tracking-[-0.01em] text-foreground">
        {REVIEW_CANDIDATE_COPY.reasonsHeading}
      </h3>
      <ul
        className="mt-2 list-disc space-y-1.5 pl-5 text-[13px] leading-6 text-foreground"
        data-testid="review-candidate-reasons"
      >
        {candidate.reasons.map((reason) => (
          <li key={reason}>{reason}</li>
        ))}
      </ul>
      <div className="mt-8 border-t border-border pt-6" data-testid="review-candidate-plan-divider">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2">
            <h3 className="text-[13px] font-semibold tracking-[-0.01em] text-foreground">
              {REVIEW_CANDIDATE_COPY.planHeading}
            </h3>
            <Badge
              variant="outline"
              className="rounded-full font-mono text-[11px] font-normal text-muted-foreground"
              data-testid="review-candidate-plan-file"
            >
              {REVIEW_CANDIDATE_COPY.planFile}
            </Badge>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="shrink-0"
            onClick={() => setEditingPlan((open) => !open)}
            data-testid="review-candidate-edit-plan"
          >
            {editingPlan ? null : <Pencil className="size-3.5" aria-hidden />}
            {editingPlan ? REVIEW_CANDIDATE_COPY.donePlan : REVIEW_CANDIDATE_COPY.editPlan}
          </Button>
        </div>
        {editingPlan ? (
          <Textarea
            value={planMarkdown}
            onChange={(event) => onPlanMarkdownChange(event.target.value)}
            aria-label={REVIEW_CANDIDATE_COPY.planEditorLabel}
            spellCheck={false}
            className="mt-3 min-h-[280px] resize-y font-mono text-[13px] leading-relaxed"
          />
        ) : (
          <MarkdownContent
            className="mt-3"
            content={planMarkdown}
            variant="workspace"
            data-testid="review-candidate-plan"
          />
        )}
      </div>
    </>
  );
}

function TicketPanel({ candidate }: { candidate: ReviewCandidate }) {
  const { issue } = candidate;

  return (
    <div data-testid="review-candidate-ticket">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[12px] font-medium tracking-[-0.01em] text-muted-foreground">{candidate.ticketKey}</p>
          <p className="mt-1 text-[13px] text-muted-foreground">
            {REVIEW_CANDIDATE_COPY.ticketSource}
            <span className="mx-1.5 text-border">·</span>
            {REVIEW_CANDIDATE_COPY.ticketRepository}
          </p>
        </div>
        <a
          href={issue.url}
          target="_blank"
          rel="noopener noreferrer"
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "shrink-0")}
          data-testid="review-candidate-open-issue"
        >
          {REVIEW_CANDIDATE_COPY.openIssue}
          <ExternalLink className="size-3.5" aria-hidden />
        </a>
      </div>
      <h3 className="mt-5 text-[18px] font-semibold tracking-[-0.02em] text-foreground">{candidate.title}</h3>
      <p className="mt-2 text-[13px] text-muted-foreground">
        {REVIEW_CANDIDATE_COPY.opened} {formatRelative(issue.createdAt)}
        <span className="mx-1.5 text-border">·</span>
        {REVIEW_CANDIDATE_COPY.updated} {formatRelative(issue.updatedAt)}
      </p>
      <ul className="mt-3 flex flex-wrap gap-1.5" data-testid="review-candidate-issue-labels">
        {issue.labels.map((label) => (
          <li key={label.name}>
            <Badge variant="outline" className="rounded-full font-normal">
              {label.name}
            </Badge>
          </li>
        ))}
      </ul>
      <MarkdownContent className="mt-5" content={issue.bodyMarkdown} variant="workspace" />
      <dl className="mt-8 grid gap-5 sm:grid-cols-2">
        <div>
          <dt className="text-[12px] font-medium tracking-[-0.01em] text-muted-foreground">
            {REVIEW_CANDIDATE_COPY.author}
          </dt>
          <dd className="mt-2" data-testid="review-candidate-issue-author">
            <IssuePerson person={issue.author} />
          </dd>
        </div>
        <div>
          <dt className="text-[12px] font-medium tracking-[-0.01em] text-muted-foreground">
            {REVIEW_CANDIDATE_COPY.assignees}
          </dt>
          <dd className="mt-2" data-testid="review-candidate-issue-assignees">
            {issue.assignees.length === 0 ? (
              <p className="text-[13px] text-muted-foreground">{REVIEW_CANDIDATE_COPY.noAssignees}</p>
            ) : (
              <ul className="space-y-2">
                {issue.assignees.map((person) => (
                  <li key={person.login}>
                    <IssuePerson person={person} />
                  </li>
                ))}
              </ul>
            )}
          </dd>
        </div>
      </dl>
    </div>
  );
}

function IssuePerson({ person }: { person: ReviewIssuePerson }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-1.5">
      <Avatar src={person.avatarUrl} initials={getUserInitials(person.name)} alt={person.name} className="size-5" />
      <span className="truncate text-[13px] text-foreground">{person.name}</span>
      <span className="truncate text-[13px] text-muted-foreground">@{person.login}</span>
    </span>
  );
}
