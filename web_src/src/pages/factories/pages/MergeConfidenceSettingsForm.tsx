import { Button } from "@/components/ui/button";
import { useEffect, useState } from "react";

import { ColumnAutomationFooterAction } from "./columnAutomationFooterSlot";
import { MergeConfidenceCheckList } from "./MergeConfidenceCheckList";
import { MERGE_CONFIDENCE_SETTINGS_COPY } from "./mergeConfidenceCopy";
import {
  defaultMergeConfidenceChecks,
  draftWithMergeConfidence,
  parseEnabledChecks,
  type MergeConfidenceCheck,
} from "./mergeConfidenceChecks";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import { defaultRiskScoreCategories, parseRiskScoreRules } from "./riskScoreCategories";

/** General tab for an installed merge confidence automation. Checks live in the agent prompt. */
export function MergeConfidenceSettingsForm({
  draft,
  onSave,
}: {
  draft: PlanningReviewDraft;
  onSave: (next: PlanningReviewDraft) => Promise<void> | void;
}) {
  const prompt = reviewPrompt(draft);
  const [checks, setChecks] = useState<MergeConfidenceCheck[]>(
    () => parseEnabledChecks(prompt) ?? defaultMergeConfidenceChecks(),
  );
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const nextChecks = parseEnabledChecks(prompt);
    if (!nextChecks || !parseRiskScoreRules(prompt)) {
      return;
    }
    setChecks(nextChecks);
  }, [prompt]);

  const toggle = (check: MergeConfidenceCheck, enabled: boolean) => {
    setChecks((current) => {
      const next = enabled ? [...current, check] : current.filter((item) => item !== check);
      return defaultMergeConfidenceChecks().filter((item) => next.includes(item));
    });
  };

  const save = async () => {
    const categories = parseRiskScoreRules(prompt) ?? defaultRiskScoreCategories();
    const next = draftWithMergeConfidence(draft, { checks, categories });
    if (!next) {
      return;
    }
    setSaving(true);
    try {
      await onSave(next);
    } catch {
      // The agent save path reports the error.
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6" data-testid="merge-confidence-settings">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          <section className="space-y-3">
            <header>
              <h2 className="text-[15px] font-semibold text-foreground">
                {MERGE_CONFIDENCE_SETTINGS_COPY.checksLabel}
              </h2>
              <p className="workspace-body-text mt-1 text-muted-foreground">
                {MERGE_CONFIDENCE_SETTINGS_COPY.checksHelper}
              </p>
            </header>
            <MergeConfidenceCheckList checks={checks} onToggle={toggle} idPrefix="merge-confidence-check" />
          </section>
        </div>
      </div>
      <ColumnAutomationFooterAction>
        <Button
          type="button"
          disabled={saving}
          onClick={() => void save()}
          data-testid="merge-confidence-settings-save"
        >
          {saving ? MERGE_CONFIDENCE_SETTINGS_COPY.saving : MERGE_CONFIDENCE_SETTINGS_COPY.save}
        </Button>
      </ColumnAutomationFooterAction>
    </>
  );
}

function reviewPrompt(draft: PlanningReviewDraft): string {
  const steps = draft.components[0]?.configuration.steps;
  if (!Array.isArray(steps)) {
    return "";
  }
  for (const step of steps) {
    const prompt = (step as { prompt?: string }).prompt;
    if (prompt && parseEnabledChecks(prompt) && parseRiskScoreRules(prompt)) {
      return prompt;
    }
  }
  return "";
}
