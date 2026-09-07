import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { Workflow } from "lucide-react";
import { useState } from "react";

import { PlanningReviewForm } from "./PlanningReviewForm";
import { PlanningReviewNav } from "./PlanningReviewNav";
import {
  PLANNING_REVIEW_DRAFT,
  singleAgentDraft,
  type PlanningReviewDraft,
  type PlanningReviewStep,
} from "./planningReviewMockup";
import { PLANNING_REVIEW_DEFAULT_SECTION, type PlanningReviewSectionId } from "./planningReviewSections";
import { PopupBody } from "./work-order-popup-redesign/popupShared";

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

export type PlanningReviewAgentSlot = {
  draft?: PlanningReviewDraft;
  isLoading?: boolean;
  organizationId?: string;
  onSave?: (draft: PlanningReviewDraft) => void | Promise<void>;
};

/** Agent editor body. The column menu popup and the automation view Agent tab share this. */
export function PlanningReviewEditor({
  initialDraft = PLANNING_REVIEW_DRAFT,
  onSave,
  onCancel,
  organizationId,
  automationHref,
  isLoading = false,
  showAutomationNote = true,
  showCancel = true,
}: {
  initialDraft?: PlanningReviewDraft;
  onSave?: (draft: PlanningReviewDraft) => void | Promise<void>;
  onCancel?: () => void;
  organizationId?: string;
  automationHref?: string;
  isLoading?: boolean;
  showAutomationNote?: boolean;
  showCancel?: boolean;
}) {
  const [draft, setDraft] = useState(() => singleAgentDraft(initialDraft));
  const [isSaving, setIsSaving] = useState(false);
  const [section, setSection] = useState<PlanningReviewSectionId>(PLANNING_REVIEW_DEFAULT_SECTION);
  const saveDisabled = isLoading || isSaving || draft.components.length === 0;
  const stepCount = ((draft.components[0]?.configuration.steps as PlanningReviewStep[]) ?? []).length;

  const handleSave = async () => {
    if (!onSave) {
      onCancel?.();
      return;
    }
    setIsSaving(true);
    try {
      await onSave(draft);
      onCancel?.();
    } catch {
      // Caller reports the error and keeps the editor open.
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col" data-testid="planning-review-editor">
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
        {showAutomationNote ? <AutomationNote href={automationHref} /> : <span className="min-w-0 flex-1" />}
        {showCancel ? (
          <Button type="button" variant="outline" className="shrink-0" onClick={onCancel} disabled={isSaving}>
            Cancel
          </Button>
        ) : null}
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
    </div>
  );
}
