import { Button } from "@/components/ui/button";
import { useState } from "react";

import { PlanningReviewForm } from "./PlanningReviewForm";
import { PLANNING_REVIEW_DRAFT, type PlanningReviewDraft } from "./planningReviewMockup";
import { PopupBody, PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

/**
 * Simple editing mode for a phase. The canvas stays hidden while each
 * automation component exposes its configuration in a collapsible block.
 * The header title is the column name.
 */
export function PlanningReviewPopup({
  onClose,
  initialDraft = PLANNING_REVIEW_DRAFT,
  onSave,
  onRename,
  canRename = true,
  organizationId,
}: {
  onClose: () => void;
  initialDraft?: PlanningReviewDraft;
  onSave?: (draft: PlanningReviewDraft) => void;
  onRename?: (title: string) => void;
  canRename?: boolean;
  organizationId?: string;
}) {
  const [draft, setDraft] = useState(initialDraft);

  return (
    <PopupShell testId="lines-planning-review" fixed onDismiss={onClose}>
      <PopupHeader
        title={draft.title}
        onClose={onClose}
        canEditTitle={canRename}
        onTitleSave={(title) => {
          setDraft({ ...draft, title });
          onRename?.(title);
        }}
        titleTestId="planning-review-title"
        titleAriaLabel="Column title"
      />
      <PopupBody className="bg-muted">
        <PlanningReviewForm draft={draft} onChange={setDraft} organizationId={organizationId} />
      </PopupBody>
      <footer className="flex shrink-0 items-center justify-end gap-2 border-t border-border px-5 py-3">
        <Button type="button" size="sm" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button
          type="button"
          size="sm"
          onClick={() => {
            onSave?.(draft);
            onClose();
          }}
          data-testid="planning-review-save"
        >
          Save automation
        </Button>
      </footer>
    </PopupShell>
  );
}
