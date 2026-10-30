import { isCodexActionCancelled } from "../codex/codexActionContext";
import { useIntakeOwnerDraft } from "./useIntakeOwnerDraft";
import { IntakeSetupDefaultOwner } from "./IntakeDefaultOwnerField";
import { Button } from "@/components/ui/button";
import { useCreateFactoryIntake, useImportFactoryIntakeItem } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { useState, useRef } from "react";

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
 * Two steps: choose alert filters, then pick packages to import. The intake is
 * created only when the user finishes step two. Until then nothing listens
 * for Dependabot alerts.
 */
export function DependabotIntakeSetupDialog(props: DependabotIntakeSetupDialogProps) {
  const [settings, setSettings] = useState<IntakeSourceSettings>({
    ...DEFAULT_GITHUB_INTAKE_SETTINGS,
    name: "Dependabot alerts",
  });
  const createdIntakeId = useRef<string | undefined>(undefined);
  const importedIds = useRef(new Set<string>());
  const [step, setStep] = useState(0);
  const [importBusy, setImportBusy] = useState(false);
  const [importProgress, setImportProgress] = useState<{ done: number; total: number }>();
  const [error, setError] = useState<string>();
  const createIntake = useCreateFactoryIntake(props.organizationId, props.factoryId);
  const owner = useIntakeOwnerDraft(props.organizationId, props.factoryId);
  const importItem = useImportFactoryIntakeItem(props.organizationId, props.factoryId);
  const repository = props.repository || DEPENDABOT_INTAKE_SETUP_COPY.repositoryFallback;
  const dependabotSeverities = normalizeDependabotSeverities(settings.dependabotSeverities);

  const finishWizard = async (packageIds: string[]) => {
    setError(undefined);
    setImportBusy(true);
    try {
      const intake = createdIntakeId.current
        ? { id: createdIntakeId.current }
        : await createIntake.mutateAsync({
            source: "SOURCE_DEPENDABOT_ALERTS",
            settings: { dependabotSeverities, ...owner.settings() },
            skipInitialImport: true,
          });
      if (!intake.id) {
        props.onCreated();
        return;
      }
      createdIntakeId.current = intake.id;
      for (const [index, itemId] of packageIds.entries()) {
        setImportProgress({ done: index, total: packageIds.length });
        if (importedIds.current.has(itemId)) continue;
        await importItem.mutateAsync({ intakeId: intake.id, itemId });
        importedIds.current.add(itemId);
      }
      props.onCreated();
    } catch (cause) {
      if (isCodexActionCancelled(cause)) return;
      setError(getApiErrorMessage(cause, DEPENDABOT_INTAKE_SETUP_COPY.createError));
    } finally {
      setImportProgress(undefined);
      setImportBusy(false);
    }
  };

  if (step === 1) {
    return (
      <FirstRunShell
        testId="dependabot-intake-setup"
        busy={importBusy}
        chrome={{ stepIndex: 1, stepCount: STEP_COUNT, onBack: () => setStep(0) }}
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
          repository={repository}
          dependabotSeverities={dependabotSeverities}
          importProgress={importProgress}
          error={error}
          onSkip={() => void finishWizard([])}
          onChangeFilters={() => setStep(0)}
          onImportSelected={(ids) => void finishWizard(ids)}
        />
      </FirstRunShell>
    );
  }

  return (
    <FirstRunShell
      testId="dependabot-intake-setup"
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
          <DependabotRepository value={repository} />
          <DependabotIntakeFilterFields
            sourceId="dependabot-alerts"
            settings={settings}
            onSettingsChange={setSettings}
          />
        </div>

        <IntakeSetupDefaultOwner owner={owner} />
        {!props.setupReady ? (
          <p className="workspace-body-text text-destructive" role="alert">
            {DEPENDABOT_INTAKE_SETUP_COPY.setupRequired}
          </p>
        ) : null}

        <Button
          type="button"
          className="w-full"
          disabled={!props.setupReady || (owner.required && !owner.value)}
          onClick={() => setStep(1)}
          data-testid="dependabot-setup-finish"
        >
          {DEPENDABOT_INTAKE_SETUP_COPY.continue}
        </Button>
      </div>
    </FirstRunShell>
  );
}

function DependabotRepository({ value }: { value: string }) {
  return (
    <div>
      <p className="text-[13px] font-medium">{DEPENDABOT_INTAKE_SETUP_COPY.repositorySection}</p>
      <p className="mt-1 text-[13px] text-muted-foreground" data-testid="dependabot-setup-repository">
        {value}
      </p>
    </div>
  );
}
