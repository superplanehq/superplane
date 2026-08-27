import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { Workflow } from "lucide-react";
import { useState } from "react";

import { PlanningReviewForm } from "./PlanningReviewForm";
import { PLANNING_REVIEW_DRAFT, singleAgentDraft, type PlanningReviewDraft } from "./planningReviewMockup";
import { PopupBody, PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

export function planningReviewPopupTitle(draft: PlanningReviewDraft): string {
  return draft.components[0]?.title ?? "Editing Agent";
}

/**
 * Note that puts the agent in context. Agents run as part of an automation,
 * so the full automation editor stays one click away.
 */
function AutomationNote({ href }: { href?: string }) {
  return (
    <section
      className="flex items-center gap-3 rounded-xl border border-border bg-card px-4 py-3 shadow-sm"
      data-testid="planning-review-automation-note"
    >
      <Workflow className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <p className="min-w-0 flex-1 text-[12px] text-muted-foreground">
        Agents are part of an automation. Open the automation to add an agent or to change the order of the steps.
      </p>
      {href ? (
        <Button asChild size="sm" variant="outline" className="shrink-0">
          <Link href={href} data-testid="planning-review-edit-automation">
            Edit Automation
          </Link>
        </Button>
      ) : null}
    </section>
  );
}

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
}: {
  onClose: () => void;
  initialDraft?: PlanningReviewDraft;
  onSave?: (draft: PlanningReviewDraft) => void;
  organizationId?: string;
  automationHref?: string;
}) {
  const [draft, setDraft] = useState(() => singleAgentDraft(initialDraft));

  return (
    <PopupShell testId="lines-planning-review" fixed onDismiss={onClose}>
      <PopupHeader title={planningReviewPopupTitle(draft)} onClose={onClose} />
      <PopupBody className="bg-muted">
        <div className="flex flex-col gap-3">
          <AutomationNote href={automationHref} />
          <PlanningReviewForm draft={draft} onChange={setDraft} organizationId={organizationId} />
        </div>
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
          Save Agent
        </Button>
      </footer>
    </PopupShell>
  );
}
