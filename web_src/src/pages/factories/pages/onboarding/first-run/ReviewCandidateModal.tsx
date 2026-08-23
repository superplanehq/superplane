import { Button } from "@/components/ui/button";
import { useState } from "react";

import { PopupHeader, PopupShell } from "../../work-order-popup-redesign/popupShared";
import {
  REVIEW_CANDIDATE_COPY,
  type ReviewCandidate,
  type ReviewCandidateSection,
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

  return (
    <PopupShell testId="review-candidate-modal" wide fixed onDismiss={onClose}>
      <PopupHeader title={REVIEW_CANDIDATE_COPY.kicker} onClose={onClose} />
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
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
      </div>
      <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
        <Button type="button" variant="outline" onClick={onClose}>
          {REVIEW_CANDIDATE_COPY.back}
        </Button>
        <Button type="button" disabled={approved} onClick={() => setApproved(true)}>
          {approved ? REVIEW_CANDIDATE_COPY.approved : REVIEW_CANDIDATE_COPY.approve}
        </Button>
      </footer>
    </PopupShell>
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
