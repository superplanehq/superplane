import { Button } from "@/components/ui/button";
import { useEffect, useState } from "react";

import { ColumnAutomationFooterAction } from "./columnAutomationFooterSlot";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import {
  draftWithRiskScoreCategories,
  formatRiskScoreRules,
  parseRiskScoreRules,
  riskScoreCategoriesFromDraft,
} from "./riskScoreCategories";
import { RiskScoreCategoryEditor } from "./RiskScoreCategoryEditor";
import { RISK_SCORE_SETUP_COPY } from "./riskScoreSetupCopy";

/** General tab for an installed Risk score automation. Categories live in the agent prompt. */
export function RiskScoreSettingsForm({
  draft,
  onSave,
}: {
  draft: PlanningReviewDraft;
  onSave: (next: PlanningReviewDraft) => Promise<void> | void;
}) {
  const savedRules = formatRiskScoreRules(riskScoreCategoriesFromDraft(draft) ?? []);
  const [categories, setCategories] = useState(() => riskScoreCategoriesFromDraft(draft) ?? []);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setCategories(parseRiskScoreRules(savedRules) ?? []);
  }, [savedRules]);

  const save = async () => {
    const next = draftWithRiskScoreCategories(draft, categories);
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
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6" data-testid="risk-score-settings">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-4">
          <header>
            <h2 className="text-[15px] font-semibold text-foreground">{RISK_SCORE_SETUP_COPY.categoriesTitle}</h2>
            <p className="workspace-body-text mt-1 text-muted-foreground">
              {RISK_SCORE_SETUP_COPY.categoriesHelper} {RISK_SCORE_SETUP_COPY.scaleHelper}
            </p>
          </header>
          <RiskScoreCategoryEditor categories={categories} onChange={setCategories} />
        </div>
      </div>
      <ColumnAutomationFooterAction>
        <Button
          type="button"
          disabled={saving || categories.length === 0}
          onClick={() => void save()}
          data-testid="risk-score-settings-save"
        >
          {saving ? RISK_SCORE_SETUP_COPY.saving : RISK_SCORE_SETUP_COPY.save}
        </Button>
      </ColumnAutomationFooterAction>
    </>
  );
}
