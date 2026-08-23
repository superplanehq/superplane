import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { FileText, Ticket, Workflow } from "lucide-react";
import { useMemo, useState } from "react";

import { intakeTicketAnalysisFixture } from "../../lineIntakeModel";
import { PopupHeader, PopupShell } from "../../work-order-popup-redesign/popupShared";
import { WorkOrderSplitRunBody } from "../../work-order-split-run/WorkOrderSplitRunPopup";
import {
  REVIEW_CANDIDATE_COPY,
  isReviewCandidateTab,
  type ReviewCandidate,
  type ReviewCandidateSection,
  type ReviewCandidateTab,
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
  const analysisFixture = useMemo(
    () => intakeTicketAnalysisFixture({ id: candidate.workOrderId, title: candidate.title }, { complete: true }),
    [candidate.title, candidate.workOrderId],
  );

  return (
    <PopupShell testId="review-candidate-modal" canvas fixed onDismiss={onClose}>
      <PopupHeader title={REVIEW_CANDIDATE_COPY.kicker} onClose={onClose}>
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
            {tab === "ticket" ? <TicketPanel candidate={candidate} /> : <PlanPanel candidate={candidate} />}
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

function PlanPanel({ candidate }: { candidate: ReviewCandidate }) {
  return (
    <>
      <p className="text-[12px] font-medium tracking-[-0.01em] text-muted-foreground">{candidate.ticketKey}</p>
      <h3 className="mt-1 text-[20px] font-semibold tracking-[-0.03em] text-foreground">{candidate.title}</h3>
      <p className="mt-2 text-[13px] font-medium text-foreground">
        <span className="tabular-nums">{candidate.confidencePct}%</span>
        <span className="ml-2 text-muted-foreground">{candidate.confidenceBand}</span>
      </p>
      <p className="mt-4 text-[13px] leading-6 text-foreground">{candidate.summary}</p>
      <p className="mt-2 text-[13px] leading-6 text-muted-foreground">{candidate.readyNote}</p>
      <ol className="mt-6 flex flex-col gap-6">
        {candidate.sections.map((section) => (
          <ReviewSection key={section.number} section={section} />
        ))}
      </ol>
      <section className="mt-6 border-t border-border pt-4">
        <h4 className="text-[13px] font-semibold tracking-[-0.01em] text-foreground">No blocking questions</h4>
        <p className="mt-1 text-[13px] leading-6 text-muted-foreground">{candidate.noBlockingQuestions}</p>
      </section>
    </>
  );
}

function TicketPanel({ candidate }: { candidate: ReviewCandidate }) {
  return (
    <>
      <p className="text-[12px] font-medium tracking-[-0.01em] text-muted-foreground">{candidate.ticketKey}</p>
      <h3 className="mt-1 text-[20px] font-semibold tracking-[-0.03em] text-foreground">{candidate.title}</h3>
      <p className="mt-2 text-[13px] text-muted-foreground">
        {REVIEW_CANDIDATE_COPY.ticketSource}
        <span className="mx-1.5 text-border">·</span>
        {REVIEW_CANDIDATE_COPY.ticketRepository}
      </p>
      <p className="mt-4 text-[13px] leading-6 whitespace-pre-wrap text-foreground">{candidate.ticketBody}</p>
    </>
  );
}

function ReviewSection({ section }: { section: ReviewCandidateSection }) {
  return (
    <li data-testid={`review-candidate-section-${section.number}`}>
      <h4 className="text-[13px] font-semibold tracking-[-0.01em] text-foreground">
        <span className="mr-2 tabular-nums text-muted-foreground">{section.number}</span>
        {section.title}
      </h4>
      <p className="mt-1 text-[13px] leading-6 text-muted-foreground">{section.intro}</p>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-[13px] leading-6 text-foreground">
        {section.items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </li>
  );
}
