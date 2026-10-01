import { Button } from "@/components/ui/button";
import { useEffect, useState } from "react";

import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomations";
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
  onDelete,
  deletePending = false,
}: {
  draft: PlanningReviewDraft;
  onSave: (next: PlanningReviewDraft) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  deletePending?: boolean;
}) {
  const savedRules = formatRiskScoreRules(riskScoreCategoriesFromDraft(draft) ?? []);
  const [categories, setCategories] = useState(() => riskScoreCategoriesFromDraft(draft) ?? []);
  const [saving, setSaving] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

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
      <RiskScoreSettingsFooter
        saving={saving}
        saveDisabled={categories.length === 0}
        onSave={() => void save()}
        onDelete={onDelete}
        deletePending={deletePending}
        confirmDelete={confirmDelete}
        onConfirmDelete={setConfirmDelete}
      />
    </>
  );
}

function RiskScoreSettingsFooter({
  saving,
  saveDisabled,
  onSave,
  onDelete,
  deletePending,
  confirmDelete,
  onConfirmDelete,
}: {
  saving: boolean;
  saveDisabled: boolean;
  onSave: () => void;
  onDelete?: () => Promise<void> | void;
  deletePending: boolean;
  confirmDelete: boolean;
  onConfirmDelete: (next: boolean) => void;
}) {
  if (confirmDelete && onDelete) {
    return (
      <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
        <p className="workspace-body-text text-destructive" role="alert">
          {COLUMN_AUTOMATIONS_COPY.confirmDelete}
        </p>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" size="sm" onClick={() => onConfirmDelete(false)}>
            {COLUMN_AUTOMATIONS_COPY.keepLabel}
          </Button>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            disabled={deletePending}
            onClick={() => void onDelete()}
            data-testid="risk-score-settings-delete-confirm"
          >
            {deletePending ? COLUMN_AUTOMATIONS_COPY.deletingLabel : COLUMN_AUTOMATIONS_COPY.deleteLabel}
          </Button>
        </div>
      </footer>
    );
  }

  return (
    <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
      {onDelete ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => onConfirmDelete(true)}
          data-testid="risk-score-settings-delete"
        >
          {COLUMN_AUTOMATIONS_COPY.deleteLabel}
        </Button>
      ) : (
        <span />
      )}
      <Button type="button" disabled={saving || saveDisabled} onClick={onSave} data-testid="risk-score-settings-save">
        {saving ? RISK_SCORE_SETUP_COPY.saving : RISK_SCORE_SETUP_COPY.save}
      </Button>
    </footer>
  );
}
