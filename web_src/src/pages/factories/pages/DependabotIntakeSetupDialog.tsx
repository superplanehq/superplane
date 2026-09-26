import { Button } from "@/components/ui/button";
import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { useState } from "react";

import { DependabotIntakeFilterFields } from "./DependabotIntakeFilterFields";
import { DEPENDABOT_INTAKE_SETUP_COPY } from "./dependabotIntakeSetupCopy";
import { IntakeSkipInitialImportField } from "./IntakeSkipInitialImportField";
import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  normalizeDependabotSeverities,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import { FirstRunHeading, FirstRunShell } from "./onboarding/first-run/FirstRunShell";

interface DependabotIntakeSetupDialogProps {
  organizationId: string;
  factoryId: string;
  repository: string;
  setupReady: boolean;
  onClose: () => void;
  onCreated: () => void;
}

export function DependabotIntakeSetupDialog(props: DependabotIntakeSetupDialogProps) {
  const [settings, setSettings] = useState<IntakeSourceSettings>({
    ...DEFAULT_GITHUB_INTAKE_SETTINGS,
    name: "Dependabot alerts",
  });
  const [skipInitialImport, setSkipInitialImport] = useState(false);
  const [error, setError] = useState<string>();
  const createIntake = useCreateFactoryIntake(props.organizationId, props.factoryId);
  const repository = props.repository || DEPENDABOT_INTAKE_SETUP_COPY.repositoryFallback;

  const create = async () => {
    if (!props.setupReady) return;
    setError(undefined);
    try {
      await createIntake.mutateAsync({
        source: "SOURCE_DEPENDABOT_ALERTS",
        settings: {
          dependabotSeverities: normalizeDependabotSeverities(settings.dependabotSeverities),
        },
        ...(skipInitialImport ? { skipInitialImport: true } : {}),
      });
      props.onCreated();
    } catch (cause) {
      setError(getApiErrorMessage(cause, DEPENDABOT_INTAKE_SETUP_COPY.createError));
    }
  };

  return (
    <FirstRunShell
      testId="dependabot-intake-setup"
      busy={createIntake.isPending}
      chrome={{ stepIndex: 0, stepCount: 1, onBack: props.onClose }}
      sphere={{
        testId: "dependabot-intake-setup-sphere",
        level: 0.58,
        caption: "Awaiting alert settings",
        leftChip: { label: "Discover", value: "Dependabot alerts", tone: "ghost" },
      }}
    >
      <FirstRunHeading headline={DEPENDABOT_INTAKE_SETUP_COPY.pageTitle}>
        <p className="text-[15px] leading-6 text-muted-foreground">
          {DEPENDABOT_INTAKE_SETUP_COPY.helper(repository, skipInitialImport)}
        </p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        <div className="space-y-5 rounded-xl border border-border bg-card px-4 py-4 text-left">
          <div>
            <p className="text-[13px] font-medium">{DEPENDABOT_INTAKE_SETUP_COPY.repositorySection}</p>
            <p className="mt-1 text-[13px] text-muted-foreground" data-testid="dependabot-setup-repository">
              {repository}
            </p>
          </div>
          <DependabotIntakeFilterFields
            sourceId="dependabot-alerts"
            settings={settings}
            onSettingsChange={setSettings}
          />
          <p className="text-[13px] leading-5 text-muted-foreground" data-testid="dependabot-setup-instructions-note">
            {DEPENDABOT_INTAKE_SETUP_COPY.instructionsNote}
          </p>
        </div>

        <IntakeSkipInitialImportField
          checked={skipInitialImport}
          onCheckedChange={setSkipInitialImport}
          testId="dependabot-skip-initial-import"
        />

        {!props.setupReady ? (
          <p className="workspace-body-text text-destructive" role="alert">
            {DEPENDABOT_INTAKE_SETUP_COPY.setupRequired}
          </p>
        ) : null}

        {error ? (
          <p className="workspace-body-text text-destructive" role="alert">
            {error}
          </p>
        ) : null}

        <Button
          type="button"
          className="w-full"
          disabled={!props.setupReady || createIntake.isPending}
          onClick={() => void create()}
          data-testid="dependabot-setup-finish"
        >
          {createIntake.isPending ? DEPENDABOT_INTAKE_SETUP_COPY.creating : DEPENDABOT_INTAKE_SETUP_COPY.create}
        </Button>
      </div>
    </FirstRunShell>
  );
}
