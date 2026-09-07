import { useState } from "react";

import { PlanningReviewEditor, planningReviewEditorTitle } from "./PlanningReviewEditor";
import { PLANNING_REVIEW_DRAFT, singleAgentDraft, type PlanningReviewDraft } from "./planningReviewMockup";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

/**
 * Simple editing mode for one agent in a phase. Extra agents stay on the
 * automation canvas, one click away from the note at the top.
 */
export function PlanningReviewPopup({
  onClose,
  initialDraft = PLANNING_REVIEW_DRAFT,
  onSave,
  organizationId,
  automationHref,
  isLoading = false,
}: {
  onClose: () => void;
  initialDraft?: PlanningReviewDraft;
  onSave?: (draft: PlanningReviewDraft) => void | Promise<void>;
  organizationId?: string;
  automationHref?: string;
  isLoading?: boolean;
}) {
  const [draft] = useState(() => singleAgentDraft(initialDraft));
  const title = isLoading ? "Editing Agent" : planningReviewEditorTitle(draft);
  const description = isLoading ? undefined : draft.components[0]?.description;

  return (
    <PopupShell testId="lines-planning-review" fixed onDismiss={onClose}>
      <PopupHeader title={title} onClose={onClose}>
        {description ? (
          <p className="mt-0.5 truncate text-[12px] text-muted-foreground" data-testid="planning-review-description">
            {description}
          </p>
        ) : null}
      </PopupHeader>
      <PlanningReviewEditor
        initialDraft={initialDraft}
        onSave={onSave}
        onCancel={onClose}
        organizationId={organizationId}
        automationHref={automationHref}
        isLoading={isLoading}
      />
    </PopupShell>
  );
}
