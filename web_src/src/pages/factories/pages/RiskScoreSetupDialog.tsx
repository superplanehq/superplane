import { Button } from "@/components/ui/button";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { useInstallFactory } from "@/pages/home/useInstallFactory";
import { ArrowLeft } from "lucide-react";

import { factoryPageTitleClassName } from "./factoryPageLayoutStyles";
import { PRFeedbackSetupPreviewPane, PRFeedbackSetupWizardShell } from "./PRFeedbackSetupWizardChrome";
import { defaultRiskScoreCategories, formatRiskScoreRules } from "./riskScoreCategories";
import { RISK_SCORE_SETUP_COPY } from "./riskScoreSetupCopy";

interface RiskScoreSetupDialogProps {
  organizationId: string;
  factoryId: string;
  githubIntegrationId: string;
  appRepository: string;
  backlogRepository: string;
  defaultBranch: string;
  onClose: () => void;
  onCreated: () => void;
}

export function RiskScoreSetupDialog(props: RiskScoreSetupDialogProps) {
  const { installFactory, isInstalling } = useInstallFactory({ organizationId: props.organizationId });

  const finish = async () => {
    if (!props.githubIntegrationId) {
      showErrorToast(RISK_SCORE_SETUP_COPY.missingGitHub);
      return;
    }
    try {
      const installed = await installFactory({
        factoryId: "risk-score",
        workspaceFactoryId: props.factoryId,
        integrations: { github: { id: props.githubIntegrationId, name: "GitHub", ready: true } },
        installParams: {
          appRepository: props.appRepository,
          backlogRepository: props.backlogRepository,
          defaultBranch: props.defaultBranch,
          riskRules: formatRiskScoreRules(defaultRiskScoreCategories()),
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
    <PRFeedbackSetupWizardShell testId="risk-score-setup" preview={<RiskScoreSetupPreview />}>
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
      <RiskScale />
      <footer className="flex items-center justify-end gap-3 pt-2">
        <Button
          type="button"
          disabled={isInstalling}
          onClick={() => void finish()}
          data-testid="risk-score-setup-finish"
        >
          {isInstalling ? RISK_SCORE_SETUP_COPY.finishing : RISK_SCORE_SETUP_COPY.finish}
        </Button>
      </footer>
    </PRFeedbackSetupWizardShell>
  );
}

function RiskScale() {
  return (
    <section>
      <h2 className="text-[13px] font-medium text-foreground">{RISK_SCORE_SETUP_COPY.scaleHeading}</h2>
      <p className="workspace-body-text mt-1 text-muted-foreground">{RISK_SCORE_SETUP_COPY.scaleHelper}</p>
      <ul className="mt-3 divide-y divide-border rounded-lg border border-border" data-testid="risk-score-setup-scale">
        {RISK_SCORE_SETUP_COPY.scale.map((level) => (
          <li
            key={level.score}
            className="flex items-center gap-3 px-3 py-2.5 text-[13px]"
            data-testid={`risk-score-setup-scale-${level.score}`}
          >
            <span className="w-8 shrink-0 font-semibold tabular-nums text-foreground">{level.score}/5</span>
            <span className="min-w-0 flex-1 truncate font-medium text-foreground">{level.label}</span>
            <span className={cn("shrink-0 text-[12px] font-medium", scaleToneClassName(level.tone))}>
              {level.status}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function scaleToneClassName(tone: "low" | "caution" | "critical"): string {
  if (tone === "critical") {
    return "text-red-600 dark:text-red-400";
  }
  if (tone === "caution") {
    return "text-amber-600 dark:text-amber-400";
  }
  return "text-emerald-600 dark:text-emerald-400";
}

function RiskScoreSetupPreview() {
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
          <span className="block text-[12px] font-medium text-muted-foreground">
            {RISK_SCORE_SETUP_COPY.previewCheckName}
          </span>
          <span className="mt-1 flex items-baseline justify-between gap-2">
            <span className="flex items-baseline gap-0.5">
              <span className="text-xl font-semibold tabular-nums tracking-tight text-foreground">2</span>
              <span className="text-[12px] text-muted-foreground">/5</span>
            </span>
            <span className="text-[11px] font-medium text-emerald-600 dark:text-emerald-400">
              {RISK_SCORE_SETUP_COPY.scale[1].status}
            </span>
          </span>
          <span aria-hidden className="mt-2 block h-1 overflow-hidden rounded-full bg-muted">
            <span className="block h-full w-2/5 rounded-full bg-emerald-500" />
          </span>
          <p className="mt-3 text-[12px] text-muted-foreground">{RISK_SCORE_SETUP_COPY.previewSummary}</p>
        </article>
      </div>
    </PRFeedbackSetupPreviewPane>
  );
}
