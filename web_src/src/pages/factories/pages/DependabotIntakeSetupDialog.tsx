import { Button } from "@/components/ui/button";
import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { useState } from "react";

import { DependabotIntakeFilterFields } from "./DependabotIntakeFilterFields";
import { DependabotIntakeImportStep } from "./DependabotIntakeImportStep";
import { DEPENDABOT_INTAKE_SETUP_COPY } from "./dependabotIntakeSetupCopy";
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

const STEP_COUNT = 2;

/**
 * Two steps: choose the alert filters and create the intake, then pick the
 * packages with open alerts to import. The intake is created with
 * `skipInitialImport` so a repository with many old alerts does not flood
 * the Backlog before the user has a say.
 */
export function DependabotIntakeSetupDialog(props: DependabotIntakeSetupDialogProps) {
  const [settings, setSettings] = useState<IntakeSourceSettings>({
    ...DEFAULT_GITHUB_INTAKE_SETTINGS,
    name: "Dependabot alerts",
  });
  const [intakeId, setIntakeId] = useState<string>();
  const [importBusy, setImportBusy] = useState(false);
  const [error, setError] = useState<string>();
  const createIntake = useCreateFactoryIntake(props.organizationId, props.factoryId);
  const repository = props.repository || DEPENDABOT_INTAKE_SETUP_COPY.repositoryFallback;

  const create = async () => {
    if (!props.setupReady) return;
    setError(undefined);
    try {
      const intake = await createIntake.mutateAsync({
        source: "SOURCE_DEPENDABOT_ALERTS",
        settings: {
          dependabotSeverities: normalizeDependabotSeverities(settings.dependabotSeverities),
        },
        skipInitialImport: true,
      });
      if (!intake.id) {
        props.onCreated();
        return;
      }
      setIntakeId(intake.id);
    } catch (cause) {
      setError(getApiErrorMessage(cause, DEPENDABOT_INTAKE_SETUP_COPY.createError));
    }
  };

  if (intakeId) {
    return (
      <FirstRunShell
        testId="dependabot-intake-setup"
        busy={importBusy}
        chrome={{ stepIndex: 1, stepCount: STEP_COUNT }}
        sphere={{
          testId: "dependabot-intake-setup-sphere",
          level: 0.82,
          caption: DEPENDABOT_INTAKE_SETUP_COPY.import.caption,
          leftChip: { label: "Discover", value: "Dependabot alerts", tone: "ghost" },
        }}
      >
        <DependabotIntakeImportStep
          organizationId={props.organizationId}
          factoryId={props.factoryId}
          intakeId={intakeId}
          onBusyChange={setImportBusy}
          onDone={props.onCreated}
        />
      </FirstRunShell>
    );
  }

  return (
    <FirstRunShell
      testId="dependabot-intake-setup"
      busy={createIntake.isPending}
      chrome={{ stepIndex: 0, stepCount: STEP_COUNT, onBack: props.onClose }}
      sphere={{
        testId: "dependabot-intake-setup-sphere",
        level: 0.58,
        caption: "Awaiting alert settings",
        leftChip: { label: "Discover", value: "Dependabot alerts", tone: "ghost" },
      }}
    >
      <FirstRunHeading headline={DEPENDABOT_INTAKE_SETUP_COPY.pageTitle}>
        <p className="text-[15px] leading-6 text-muted-foreground">{DEPENDABOT_INTAKE_SETUP_COPY.helper(repository)}</p>
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
        </div>

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
