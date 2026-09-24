import { Button } from "@/components/ui/button";

import { IntakeSkipInitialImportField } from "./IntakeSkipInitialImportField";
import { FirstRunHeading, FirstRunShell } from "./onboarding/first-run/FirstRunShell";
import { GITHUB_INTAKE_SETUP_COPY } from "./githubIntakeSetupCopy";
import { useGitHubIntakeSetup, type GitHubIntakeSetupModel } from "./useGitHubIntakeSetup";

interface GitHubIntakeSetupDialogProps {
  backlogRepository: string;
  onClose: () => void;
  onCreated: () => void;
  organizationId: string;
  factoryId: string;
}

export function GitHubIntakeSetupDialog(props: GitHubIntakeSetupDialogProps) {
  const setup = useGitHubIntakeSetup(props.organizationId, props.factoryId);
  const repository = props.backlogRepository.trim();

  return (
    <FirstRunShell
      testId="github-intake-setup"
      chrome={{ onBack: props.onClose }}
      sphere={{
        testId: "github-intake-setup-sphere",
        level: 0.58,
        caption: "Awaiting repository",
        leftChip: { label: "Discover", value: "GitHub issues", tone: "ghost" },
      }}
    >
      <FirstRunHeading headline={GITHUB_INTAKE_SETUP_COPY.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{GITHUB_INTAKE_SETUP_COPY.helper}</p>
      </FirstRunHeading>
      <div className="mt-8 space-y-4">
        {repository ? (
          <div className="rounded-xl border border-border bg-card px-4 py-3.5 text-left">
            <p className="text-[12px] text-muted-foreground">{GITHUB_INTAKE_SETUP_COPY.repositoryLabel}</p>
            <p
              className="mt-1 text-[13px] font-medium tracking-[-0.01em] text-foreground"
              data-testid="github-setup-repository"
            >
              {repository}
            </p>
          </div>
        ) : (
          <p className="text-[13px] text-destructive" role="alert">
            {GITHUB_INTAKE_SETUP_COPY.repositoryMissing}
          </p>
        )}
        {setup.error ? (
          <p className="text-[13px] text-destructive" role="alert">
            {setup.error}
          </p>
        ) : null}
        <SetupFooter setup={setup} disabled={repository.length === 0} onCreated={props.onCreated} />
      </div>
    </FirstRunShell>
  );
}

function SetupFooter({
  setup,
  disabled,
  onCreated,
}: {
  setup: GitHubIntakeSetupModel;
  disabled: boolean;
  onCreated: () => void;
}) {
  return (
    <div className="space-y-3">
      <IntakeSkipInitialImportField
        checked={!setup.skipInitialImport}
        onCheckedChange={(importExisting) => setup.setSkipInitialImport(!importExisting)}
        helper={
          setup.skipInitialImport
            ? GITHUB_INTAKE_SETUP_COPY.importExistingHelperOff
            : GITHUB_INTAKE_SETUP_COPY.importExistingHelper
        }
        testId="github-skip-initial-import"
      />
      <Button
        type="button"
        className="w-full"
        disabled={disabled || setup.createIntake.isPending}
        onClick={() => {
          void setup.createGithubIntake().then((created) => {
            if (created) {
              onCreated();
            }
          });
        }}
        data-testid="github-setup-finish"
      >
        {setup.createIntake.isPending
          ? GITHUB_INTAKE_SETUP_COPY.wizardFinishing
          : GITHUB_INTAKE_SETUP_COPY.wizardFinish}
      </Button>
    </div>
  );
}
