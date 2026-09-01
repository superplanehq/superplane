import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { Workflow } from "lucide-react";
import { useState } from "react";

import { PlanningReviewForm } from "./PlanningReviewForm";
import { PlanningReviewNav } from "./PlanningReviewNav";
import { PLANNING_REVIEW_DEFAULT_SECTION, type PlanningReviewSectionId } from "./planningReviewSections";
import {
  PLANNING_REVIEW_DRAFT,
  singleAgentDraft,
  type PlanningReviewDraft,
  type PlanningReviewStep,
} from "./planningReviewMockup";
import { PopupBody, PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

function planningReviewPopupTitle(draft: PlanningReviewDraft): string {
  return draft.components[0]?.title ?? "Editing Agent";
}

/**
 * Note that puts the agent in context. Agents run as part of an automation,
 * so the full automation editor stays one click away. It sits in the footer
 * because it explains the screen, it is not an action on the agent.
 */
function AutomationNote({ href }: { href?: string }) {
  return (
    <p
      className="flex min-w-0 flex-1 items-center gap-2.5 text-[12px] leading-5 text-muted-foreground"
      data-testid="planning-review-automation-note"
    >
      <Workflow className="size-4 shrink-0" aria-hidden />
      <span className="min-w-0">
        Agents are part of an automation. Open the automation to add an agent or to change the order of the steps.
      </span>
      {href ? (
        <Link
          href={href}
          data-testid="planning-review-edit-automation"
          className="shrink-0 font-medium text-foreground underline underline-offset-2 hover:text-primary"
        >
          Edit Automation
        </Link>
      ) : null}
    </p>
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
  const [section, setSection] = useState<PlanningReviewSectionId>(PLANNING_REVIEW_DEFAULT_SECTION);

  const title = isLoading ? "Editing Agent" : planningReviewPopupTitle(draft);
  const description = isLoading ? undefined : draft.components[0]?.description;
  const saveDisabled = isLoading || isSaving || draft.components.length === 0;
  const stepCount = ((draft.components[0]?.configuration.steps as PlanningReviewStep[]) ?? []).length;

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
      <PopupHeader title={title} onClose={onClose}>
        {description ? (
          <p className="mt-0.5 truncate text-[12px] text-muted-foreground" data-testid="planning-review-description">
            {description}
          </p>
        ) : null}
      </PopupHeader>
      {isLoading ? (
        <PopupBody className="bg-muted px-6 py-5">
          <p className="px-1 py-6 text-sm text-muted-foreground" data-testid="planning-review-loading">
            Loading agent…
          </p>
        </PopupBody>
      ) : (
        <div className="flex min-h-0 flex-1">
          <PlanningReviewNav active={section} onSelect={setSection} stepCount={stepCount} />
          <PopupBody className="min-w-0 bg-muted px-6 py-5">
            <PlanningReviewForm draft={draft} onChange={setDraft} organizationId={organizationId} section={section} />
          </PopupBody>
        </div>
      )}
      <footer className="flex shrink-0 items-center gap-4 border-t border-border px-6 py-4">
        <AutomationNote href={automationHref} />
        <Button type="button" variant="outline" className="shrink-0" onClick={onClose} disabled={isSaving}>
          Cancel
        </Button>
        <Button
          type="button"
          className="shrink-0"
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
