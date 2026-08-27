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
  isLoading = false,
}: {
  onClose: () => void;
  initialDraft?: PlanningReviewDraft;
  onSave?: (draft: PlanningReviewDraft) => void | Promise<void>;
  organizationId?: string;
  automationHref?: string;
  isLoading?: boolean;
}) {
  const [draft, setDraft] = useState(() => singleAgentDraft(initialDraft));
  const [isSaving, setIsSaving] = useState(false);

  const title = isLoading ? "Editing Agent" : planningReviewPopupTitle(draft);
  const saveDisabled = isLoading || isSaving || draft.components.length === 0;

  const handleSave = async () => {
    if (!onSave) {
      onClose();
      return;
    }
    setIsSaving(true);
    try {
      await onSave(draft);
      onClose();
    } catch {
      // Caller reports the error and keeps this popup open.
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <PopupShell testId="lines-planning-review" fixed onDismiss={onClose}>
      <PopupHeader title={title} onClose={onClose} />
      <PopupBody className="bg-muted">
        <div className="flex flex-col gap-3">
          <AutomationNote href={automationHref} />
          {isLoading ? (
            <p className="px-1 py-6 text-[13px] text-muted-foreground" data-testid="planning-review-loading">
              Loading agent…
            </p>
          ) : (
            <PlanningReviewForm draft={draft} onChange={setDraft} organizationId={organizationId} />
          )}
        </div>
      </PopupBody>
      <footer className="flex shrink-0 items-center justify-end gap-2 border-t border-border px-5 py-3">
        <Button type="button" size="sm" variant="outline" onClick={onClose} disabled={isSaving}>
          Cancel
        </Button>
        <Button
          type="button"
          size="sm"
          onClick={() => void handleSave()}
          disabled={saveDisabled}
          data-testid="planning-review-save"
        >
          {isSaving ? "Saving…" : "Save Agent"}
        </Button>
      </footer>
    </PopupShell>
  );
}
