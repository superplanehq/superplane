import { Button } from "@/components/ui/button";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useInstallFactory } from "@/pages/home/useInstallFactory";
import { ArrowLeft } from "lucide-react";
import { useState } from "react";

import { factoryPageTitleClassName } from "./factoryPageLayoutStyles";
import { MergeConfidenceCheckList } from "./MergeConfidenceCheckList";
import { MERGE_CONFIDENCE_CHECK_COPY } from "./mergeConfidenceCopy";
import {
  defaultMergeConfidenceChecks,
  formatEnabledChecksValue,
  type MergeConfidenceCheck,
} from "./mergeConfidenceChecks";
import { PRFeedbackSetupPreviewPane, PRFeedbackSetupWizardShell } from "./PRFeedbackSetupWizardChrome";
import { defaultRiskScoreCategories, formatRiskScoreRules } from "./riskScoreCategories";
import { RISK_SCORE_SETUP_COPY } from "./riskScoreSetupCopy";

interface RiskScoreSetupDialogProps {
  organizationId: string;
  factoryId: string;
  githubIntegrationId: string;
  githubInstallationName: string;
  appRepository: string;
  backlogRepository: string;
  defaultBranch: string;
  onClose: () => void;
  onCreated: () => void;
}

export function RiskScoreSetupDialog(props: RiskScoreSetupDialogProps) {
  const { installFactory, isInstalling } = useInstallFactory({ organizationId: props.organizationId });
  const [checks, setChecks] = useState<MergeConfidenceCheck[]>(defaultMergeConfidenceChecks);

  const installationName = props.githubInstallationName.trim();
  const waitingForInstallationName = Boolean(props.githubIntegrationId) && !installationName;

  const finish = async () => {
    if (!props.githubIntegrationId || !installationName) {
      showErrorToast(RISK_SCORE_SETUP_COPY.missingGitHub);
      return;
    }
    try {
      const installed = await installFactory({
        factoryId: "risk-score",
        workspaceFactoryId: props.factoryId,
        integrations: { github: { id: props.githubIntegrationId, name: installationName, ready: true } },
        installParams: {
          appRepository: props.appRepository,
          backlogRepository: props.backlogRepository,
          defaultBranch: props.defaultBranch,
          riskRules: formatRiskScoreRules(defaultRiskScoreCategories()),
          enabledChecks: formatEnabledChecksValue(checks),
        },
        startingTaskPrompt: "",
        navigateOnComplete: false,
        startInitialRun: false,
      });
      if (!installed?.canvasId) {
        return;
      }
      showSuccessToast(RISK_SCORE_SETUP_COPY.created);
      props.onCreated();
    } catch {
      // useInstallFactory already reports the error.
    }
  };

  return (
    <PRFeedbackSetupWizardShell testId="risk-score-setup" preview={<RiskScoreSetupPreview checks={checks} />}>
      <header className="text-left">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="mb-6 text-muted-foreground"
          onClick={props.onClose}
          data-testid="risk-score-setup-back"
        >
          <ArrowLeft className="size-4 shrink-0" aria-hidden />
          {RISK_SCORE_SETUP_COPY.back}
        </Button>
        <h1 className={factoryPageTitleClassName}>{RISK_SCORE_SETUP_COPY.title}</h1>
        <p className="workspace-body-text mt-2 text-muted-foreground">{RISK_SCORE_SETUP_COPY.helper}</p>
      </header>
      <MergeConfidenceCheckList
        checks={checks}
        idPrefix="merge-confidence-setup-check"
        onToggle={(check, enabled) => {
          setChecks((current) => {
            const next = enabled ? [...current, check] : current.filter((item) => item !== check);
            return defaultMergeConfidenceChecks().filter((item) => next.includes(item));
          });
        }}
      />
      <footer className="flex items-center justify-end gap-3 pt-2">
        <Button
          type="button"
          disabled={isInstalling || waitingForInstallationName}
          onClick={() => void finish()}
          data-testid="risk-score-setup-finish"
        >
          {isInstalling ? RISK_SCORE_SETUP_COPY.finishing : RISK_SCORE_SETUP_COPY.finish}
        </Button>
      </footer>
    </PRFeedbackSetupWizardShell>
  );
}

function RiskScoreSetupPreview({ checks }: { checks: readonly MergeConfidenceCheck[] }) {
  const visible = MERGE_CONFIDENCE_CHECK_COPY.filter((check) => checks.includes(check.id));
  return (
    <PRFeedbackSetupPreviewPane
      label={RISK_SCORE_SETUP_COPY.previewLabel}
      caption={RISK_SCORE_SETUP_COPY.previewCaption}
      testId="risk-score-setup-preview"
      captionTestId="risk-score-setup-preview-caption"
    >
      <div className="w-full max-w-sm">
        <p className="mb-3 font-mono text-[11px] tracking-[0.04em] text-muted-foreground">
          {RISK_SCORE_SETUP_COPY.previewTaskLabel} {RISK_SCORE_SETUP_COPY.previewTaskKey}
        </p>
        <article
          className="rounded-lg border border-border bg-card px-4 py-3 shadow-sm"
          data-testid="risk-score-setup-preview-card"
        >
          {visible.length === 0 ? (
            <p className="text-[12px] text-muted-foreground">{RISK_SCORE_SETUP_COPY.previewEmpty}</p>
          ) : (
            <ul className="divide-y divide-border">
              {visible.map((check) => {
                const preview = RISK_SCORE_SETUP_COPY.previewScores[check.id];
                return (
                  <li key={check.id} className="flex items-baseline justify-between gap-3 py-2 first:pt-0 last:pb-0">
                    <span className="text-[12px] font-medium text-muted-foreground">{check.label}</span>
                    <span className="flex items-baseline gap-2">
                      <span className="text-[13px] font-semibold tabular-nums text-foreground">{preview.score}/5</span>
                      <span className="text-[11px] font-medium text-emerald-600 dark:text-emerald-400">
                        {preview.status}
                      </span>
                    </span>
                  </li>
                );
              })}
            </ul>
          )}
        </article>
      </div>
    </PRFeedbackSetupPreviewPane>
  );
}
